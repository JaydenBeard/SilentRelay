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
	"github.com/jaydenbeard/messaging-app/internal/auth"
	"github.com/jaydenbeard/messaging-app/internal/config"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// PresenceService tracks user online/offline status
type PresenceService struct {
	redis       *redis.Client
	authService *auth.AuthService
}

type PresenceResponse struct {
	UserID   string    `json:"user_id"`
	IsOnline bool      `json:"is_online"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://messaging:messaging@localhost:5432/messaging?sslmode=disable"
	}

	// Load configuration with secure JWT secret handling
	cfg := config.Load()

	// Initialize JWT key manager with secure secret
	config.InitializeKeyManager(cfg.JWTSecret)

	// Validate JWT secret meets security requirements
	if err := config.ValidateJWTSecret(cfg.JWTSecret); err != nil {
		log.Fatalf("FATAL: JWT secret validation failed: %v", err)
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
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Initialize auth service with secure JWT secret management
	authService, err := auth.NewAuthService(database, config.GetCurrentSecret())
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}

	service := &PresenceService{
		redis:       rdb,
		authService: authService,
	}

	// Setup routes
	router := mux.NewRouter()

	// Apply authentication middleware to all routes, but skip /health and /metrics
	skipAuth := func(r *http.Request) bool {
		return r.URL.Path == "/health" || r.URL.Path == "/metrics"
	}
	router.Use(middleware.AuthMiddleware(authService, skipAuth))

	// Health endpoint (public)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
			log.Printf("Warning: failed to encode health response: %v", err)
		}
	}).Methods("GET")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Protected routes - require JWT authentication
	router.HandleFunc("/presence/{userId}", service.GetPresence).Methods("GET")
	router.HandleFunc("/presence/batch", service.GetBatchPresence).Methods("POST")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("🟢 Presence Service listening on port %s", port)
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
		log.Printf("Warning: server shutdown error: %v", err)
	}
}

func (s *PresenceService) GetPresence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	presence := s.getUserPresence(userID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(presence); err != nil {
		log.Printf("Warning: failed to encode presence response: %v", err)
	}
}

func (s *PresenceService) GetBatchPresence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	results := make([]PresenceResponse, len(req.UserIDs))
	for i, userID := range req.UserIDs {
		results[i] = s.getUserPresence(userID)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("Warning: failed to encode results: %v", err)
	}
}

func (s *PresenceService) getUserPresence(userID string) PresenceResponse {
	ctx := context.Background()
	key := "presence:" + userID

	val, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return PresenceResponse{
			UserID:   userID,
			IsOnline: false,
		}
	}

	if val == "online" {
		return PresenceResponse{
			UserID:   userID,
			IsOnline: true,
			LastSeen: time.Now().UTC(),
		}
	}

	// Parse last seen time
	lastSeen, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return PresenceResponse{
			UserID:   userID,
			IsOnline: false,
		}
	}

	return PresenceResponse{
		UserID:   userID,
		IsOnline: false,
		LastSeen: lastSeen,
	}
}
