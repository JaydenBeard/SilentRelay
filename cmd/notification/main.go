package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// NotificationService handles push notifications
type NotificationService struct {
	redis *redis.Client
}

type PushNotification struct {
	UserID    string          `json:"user_id"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data,omitempty"`
	Priority  string          `json:"priority"` // high, normal
	Timestamp time.Time       `json:"timestamp"`
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8082"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// Connect to Redis with optional password
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: os.Getenv("REDIS_PASSWORD"), // Empty string if not set
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	service := &NotificationService{redis: rdb}

	// Subscribe to notification events
	go service.subscribeToNotifications()

	// Setup routes
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
			log.Printf("Failed to encode health response: %v", err)
		}
	}).Methods("GET")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	router.HandleFunc("/notifications/send", service.SendNotification).Methods("POST")
	router.HandleFunc("/notifications/register", service.RegisterDevice).Methods("POST")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("🔔 Notification Service listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func (s *NotificationService) subscribeToNotifications() {
	ctx := context.Background()
	pubsub := s.redis.PSubscribe(ctx, "notifications:*")
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()
	for msg := range ch {
		var notification PushNotification
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			log.Printf("Failed to parse notification: %v", err)
			continue
		}

		s.deliverNotification(&notification)
	}
}

func (s *NotificationService) deliverNotification(notification *PushNotification) {
	// Get user's push tokens
	ctx := context.Background()
	key := "push_tokens:" + notification.UserID

	tokens, err := s.redis.SMembers(ctx, key).Result()
	if err != nil || len(tokens) == 0 {
		log.Printf("No push tokens for user %s", notification.UserID)
		return
	}

	// In production, send to FCM/APNs here
	for _, token := range tokens {
		log.Printf("📱 Would send push to token %s: %s", token[:20]+"...", notification.Title)
		// TODO: Integrate with Firebase Cloud Messaging or Apple Push Notification Service
	}
}

func (s *NotificationService) SendNotification(w http.ResponseWriter, r *http.Request) {
	var notification PushNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	notification.Timestamp = time.Now().UTC()

	// Publish to Redis for processing
	ctx := context.Background()
	data, _ := json.Marshal(notification)
	s.redis.Publish(ctx, "notifications:"+notification.UserID, data)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "queued"}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *NotificationService) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string `json:"user_id"`
		PushToken string `json:"push_token"`
		Platform  string `json:"platform"` // ios, android, web
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store push token in Redis
	ctx := context.Background()
	key := "push_tokens:" + req.UserID
	s.redis.SAdd(ctx, key, req.PushToken)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "registered"}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
