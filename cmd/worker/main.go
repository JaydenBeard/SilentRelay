package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"context"

	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Worker processes messages from the queue for analytics and archival
func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://messaging:messaging@localhost:5432/messaging?sslmode=disable"
	}

	consumerGroup := os.Getenv("CONSUMER_GROUP")
	if consumerGroup == "" {
		consumerGroup = "message_processors"
	}

	consumerName := os.Getenv("CONSUMER_NAME")
	if consumerName == "" {
		consumerName = "worker-1"
	}

	// Connect to Redis with optional password
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: os.Getenv("REDIS_PASSWORD"), // Empty string if not set
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Connect to PostgreSQL
	database, err := db.NewPostgresDB(postgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	// Create message queue
	mq := queue.NewMessageQueue(rdb, "message_events")

	log.Printf("🔄 Queue Worker started: group=%s, consumer=%s", consumerGroup, consumerName)

	// Start processing in a goroutine
	go mq.StartConsumer(consumerGroup, consumerName, func(msg *queue.QueuedMessage) error {
		log.Printf("📦 Processing message event: %s (type: %s)", msg.MessageID, msg.EventType)

		switch msg.EventType {
		case "archived":
			// Store in long-term archive
			// In production, could write to a data warehouse
			log.Printf("📁 Archiving message %s", msg.MessageID)

		case "delivered":
			// Update analytics
			log.Printf("📊 Recording delivery for message %s", msg.MessageID)

		case "read":
			// Update analytics
			log.Printf("📊 Recording read receipt for message %s", msg.MessageID)

		case "pending_delivery":
			// Could retry delivery or escalate
			log.Printf("⏳ Message %s pending delivery", msg.MessageID)
		}

		return nil
	})

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Worker shutting down...")
}
