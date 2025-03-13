package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
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

// Global variables to manage the in-memory cache.
var (
	messageCache []string   // Holds the list of cached messages.
	cacheMu      sync.Mutex // Mutex to protect concurrent access to messageCache.
)

// loadCache reads the cached messages from disk, if available.
func loadCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	// Using os.ReadFile instead of ioutil.ReadFile (deprecated in newer Go versions).
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist yet; initialize an empty cache.
			messageCache = []string{}
			return
		}
		log.Printf("Error reading cache file: %v", err)
		return
	}

	// Unmarshal the JSON data into the messageCache slice.
	if err = json.Unmarshal(data, &messageCache); err != nil {
		log.Printf("Error unmarshaling cache file: %v", err)
		messageCache = []string{}
	}
	log.Printf("Loaded %d cached messages.", len(messageCache))
}

// saveCache persists the current message cache to disk.
func saveCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	// Marshal the cache into JSON format.
	data, err := json.Marshal(messageCache)
	if err != nil {
		log.Printf("Error marshaling cache: %v", err)
		return
	}

	// Write the JSON data to the cache file with appropriate permissions.
	if err = os.WriteFile(cacheFile, data, 0644); err != nil {
		log.Printf("Error writing cache file: %v", err)
	}
}

// addToCache appends a new message to the cache (ensuring maximum size)
// and triggers an asynchronous save of the updated cache to disk.
func addToCache(message string) {
	cacheMu.Lock()
	// Append new message.
	messageCache = append(messageCache, message)
	// Trim the cache if it exceeds the maximum size.
	if len(messageCache) > maxCacheSize {
		messageCache = messageCache[len(messageCache)-maxCacheSize:]
	}
	cacheMu.Unlock() // Unlock before starting the goroutine for saving.

	// Save the updated cache asynchronously.
	go saveCache()
}

// upgrader configures the WebSocket upgrade with a permissive origin check.
// Adjust the CheckOrigin function as needed for your security requirements.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// clients holds the active WebSocket connections.
// clientsMu protects concurrent access to the clients map.
var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

// broadcastMessage sends the provided message to all connected WebSocket clients.
func broadcastMessage(message string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	// Iterate over all clients and send the message.
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error writing to websocket client: %v", err)
			// Close and remove client on error.
			client.Close()
			delete(clients, client)
		}
	}
}

// wsHandler handles HTTP requests and upgrades them to WebSocket connections.
// It registers new clients and sends the current cached messages upon connection.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Register the new WebSocket client.
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	// Send all cached messages to the newly connected client.
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
		// ReadMessage blocks until a message is received or an error occurs.
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	// Remove the client once the connection is closed.
	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	conn.Close()
}

func main() {
	// Load persisted cache from disk.
	loadCache()

	// Load environment variables from .env file.
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Define Kafka topic to consume from.
	topic := "news_fake"
	ctx := context.Background()

	// Retrieve Kafka broker address from environment variable.
	seeds := []string{os.Getenv("broker")}
	opts := []kgo.Opt{
		kgo.SeedBrokers(seeds...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	// Set up TLS configuration for secure Kafka connection.
	opts = append(opts, kgo.DialTLSConfig(new(tls.Config)))

	// Configure SASL/SCRAM 256 authentication using environment variables.
	opts = append(opts, kgo.SASL(scram.Auth{
		User: os.Getenv("username"),
		Pass: os.Getenv("password"),
	}.AsSha256Mechanism()))

	// Initialize Kafka client.
	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Log and broadcast the successful Kafka connection.
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

	// Continuously poll Kafka for new messages.
	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			log.Fatal(fmt.Sprint(errs))
		}
		// Iterate through each fetched record.
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			topicInfo := fmt.Sprintf("topic: %s (%d|%d)", record.Topic, record.Partition, record.Offset)
			messageInfo := fmt.Sprintf("key: %s, value: %s", record.Key, record.Value)
			finalMessage := fmt.Sprintf("Message consumed: %s, %s", topicInfo, messageInfo)
			log.Println(finalMessage)

			// Cache the message and broadcast it to connected WebSocket clients.
			addToCache(finalMessage)
			broadcastMessage(finalMessage)
		}
	}
}
