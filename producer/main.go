package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

const (
	redditURL    = "https://www.reddit.com/r/conspiracy/new.json"
	topic        = "news_raw"
	pollInterval = 30 * time.Second
)

var latestTimestamp float64 = 0

type RedditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID         string  `json:"id"`
				Title      string  `json:"title"`
				Permalink  string  `json:"permalink"`
				URL        string  `json:"url"`
				CreatedUTC float64 `json:"created_utc"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func fetchRedditPosts() ([]map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", redditURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "golang:reddit.producer:v1.0 (by /u/yourusername)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var redditData RedditResponse
	if err := json.Unmarshal(body, &redditData); err != nil {
		return nil, err
	}

	var posts []map[string]interface{}
	for _, post := range redditData.Data.Children {
		createdUTC := post.Data.CreatedUTC
		if createdUTC <= latestTimestamp {
			continue // Skip older posts
		}

		postData := map[string]interface{}{
			"id":          post.Data.ID,
			"title":       post.Data.Title,
			"permalink":   "https://www.reddit.com" + post.Data.Permalink,
			"url":         post.Data.URL,
			"created_utc": createdUTC,
		}
		posts = append(posts, postData)
	}

	if len(posts) > 0 {
		latestTimestamp = posts[0]["created_utc"].(float64) // Update latest timestamp
	}
	return posts, nil
}

func produceMessages(client *kgo.Client, posts []map[string]interface{}) {
	ctx := context.Background()
	var wg sync.WaitGroup

	for _, post := range posts {
		wg.Add(1)
		postJSON, _ := json.Marshal(post)

		record := &kgo.Record{
			Topic: topic,
			Value: postJSON,
		}

		client.Produce(ctx, record, func(record *kgo.Record, err error) {
			defer wg.Done()
			if err != nil {
				log.Printf("Error sending message: %v", err)
			} else {
				log.Printf("Message sent: topic=%s, offset=%d, post_title=%s",
					topic, record.Offset, post["title"])
			}
		})
	}
	wg.Wait()
}

func main() {
	seeds := []string{"cv8f7929anc1h5oheft0.any.us-east-1.mpx.prd.cloud.redpanda.com:9092"}

	opts := []kgo.Opt{
		kgo.SeedBrokers(seeds...),
		kgo.DialTLSConfig(new(tls.Config)), // TLS
		kgo.SASL(scram.Auth{User: "testnews", Pass: "GulRyfN80j5zBcnbtRPvscApn6aFd8"}.AsSha256Mechanism()), // SASL
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatalf("Failed to start Kafka client: %v", err)
	}
	defer client.Close()

	for {
		posts, err := fetchRedditPosts()
		if err != nil {
			log.Printf("Failed to fetch Reddit posts: %v", err)
		} else if len(posts) > 0 {
			produceMessages(client, posts)
		} else {
			log.Println("No new posts to send.")
		}
		time.Sleep(pollInterval)
	}
}
