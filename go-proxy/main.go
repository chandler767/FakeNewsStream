package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Upgrader with permissive origin check (adjust as needed)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// A simple clients registry with a mutex for safe concurrent access
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

// wsHandler upgrades HTTP connections to WebSocket connections and registers them.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	// Keep the connection open until the client disconnects.
	for {
		// ReadMessage is used here to detect a disconnect.
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	conn.Close()
}

func main() {
	topic := "news_fake"
	ctx := context.Background()

	seeds := []string{"cv8f7929anc1h5oheft0.any.us-east-1.mpx.prd.cloud.redpanda.com:9092"}
	opts := []kgo.Opt{
		kgo.SeedBrokers(seeds...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	// Initialize TLS config for secure connection.
	opts = append(opts, kgo.DialTLSConfig(new(tls.Config)))

	// Initialize SASL/SCRAM 256 authentication.
	opts = append(opts, kgo.SASL(scram.Auth{
		User: "goproxy",
		Pass: "VSFx0oLraAJUtXQ4uS2G0HB0X8OKhy",
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

	// Start the WebSocket server in a separate goroutine.
	go func() {
		http.HandleFunc("/ws", wsHandler)
		log.Println("WebSocket server listening on :8080/ws")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()

	// Limit the initial fetch to 50 messages.
	const initialFetchLimit = 50
	var initialCount int

	// Poll for Kafka messages and handle them.
	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			log.Fatal(fmt.Sprint(errs))
		}
		iter := fetches.RecordIter()
		for !iter.Done() {
			// If we've already processed 50 messages, break out.
			if initialCount >= initialFetchLimit {
				log.Printf("Reached max initial fetch of %d messages.", initialFetchLimit)
				return
			}

			record := iter.Next()
			topicInfo := fmt.Sprintf("topic: %s (%d|%d)", record.Topic, record.Partition, record.Offset)
			messageInfo := fmt.Sprintf("key: %s, value: %s", record.Key, record.Value)
			finalMessage := fmt.Sprintf("Message consumed: %s, %s", topicInfo, messageInfo)
			log.Println(finalMessage)
			broadcastMessage(finalMessage)
			initialCount++
		}
	}
}
