package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// cacheFile is the file name where cached messages are persisted.
const cacheFile = "cache.json"

// maxCacheSize defines the maximum number of messages to hold in the cache.
const maxCacheSize = 50

// messageCache holds the cached messages.
var (
	messageCache []string
	cacheMu      sync.Mutex
)

// loadCache loads the message cache from disk (if available).
func loadCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			messageCache = []string{}
			return
		}
		log.Printf("Error reading cache file: %v", err)
		return
	}
	if err = json.Unmarshal(data, &messageCache); err != nil {
		log.Printf("Error unmarshaling cache file: %v", err)
		messageCache = []string{}
	}
	log.Printf("Loaded %d cached messages.", len(messageCache))
}

// saveCache writes the current cache to disk.
func saveCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	data, err := json.Marshal(messageCache)
	if err != nil {
		log.Printf("Error marshaling cache: %v", err)
		return
	}
	if err = ioutil.WriteFile(cacheFile, data, 0644); err != nil {
		log.Printf("Error writing cache file: %v", err)
	}
}

// addToCache adds a new message to the cache (maintaining a maximum size)
// and then persists the updated cache to disk.
func addToCache(message string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	messageCache = append(messageCache, message)
	if len(messageCache) > maxCacheSize {
		messageCache = messageCache[len(messageCache)-maxCacheSize:]
	}
	// Save the cache after update.
	go saveCache()
}

// Upgrader with permissive origin check (adjust as needed)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// A simple clients registry with a mutex for safe concurrent access.
var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

// broadcastMessage sends the given message to all connected WebSocket clients.
func broadcastMessage(message string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error writing to websocket client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// wsHandler upgrades HTTP connections to WebSocket connections,
// registers them, and sends the cached messages upon connection.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Register the new client.
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	// Send the cached messages to the newly connected client.
	cacheMu.Lock()
	for _, msg := range messageCache {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Printf("Error sending cached message: %v", err)
			break
		}
	}
	cacheMu.Unlock()

	// Keep the connection open until the client disconnects.
	for {
		// ReadMessage is used here to detect a disconnect.
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	conn.Close()
}

func main() {
	// Load persisted cache.
	loadCache()
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	topic := "news_fake"
	ctx := context.Background()

	seeds := []string{os.Getenv("broker")}
	opts := []kgo.Opt{
		kgo.SeedBrokers(seeds...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	// Initialize TLS config for secure connection.
	opts = append(opts, kgo.DialTLSConfig(new(tls.Config)))

	// Initialize SASL/SCRAM 256 authentication.
	opts = append(opts, kgo.SASL(scram.Auth{
		User: os.Getenv("username"),
		Pass: os.Getenv("password"),
	}.AsSha256Mechanism()))

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Notify that the Kafka connection is established.
	kafkaMessage := "Kafka connection established"
	log.Println(kafkaMessage)
	broadcastMessage(kafkaMessage)
	addToCache(kafkaMessage)

	// Start the WebSocket server in a separate goroutine.
	go func() {
		http.HandleFunc("/ws", wsHandler)
		log.Println("WebSocket server listening on :8080/ws")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()

	// Poll for Kafka messages.
	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			log.Fatal(fmt.Sprint(errs))
		}
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			topicInfo := fmt.Sprintf("topic: %s (%d|%d)", record.Topic, record.Partition, record.Offset)
			messageInfo := fmt.Sprintf("key: %s, value: %s", record.Key, record.Value)
			finalMessage := fmt.Sprintf("Message consumed: %s, %s", topicInfo, messageInfo)
			log.Println(finalMessage)

			// Cache the message and then broadcast it.
			addToCache(finalMessage)
			broadcastMessage(finalMessage)
		}
	}
}
