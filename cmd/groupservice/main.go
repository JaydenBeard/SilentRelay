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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/auth"
	"github.com/jaydenbeard/messaging-app/internal/config"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// GroupService handles group membership and message fan-out
type GroupService struct {
	redis *redis.Client
	db    *db.PostgresDB
}

// MemberStatus represents a group member's online status
type MemberStatus struct {
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
	ServerID string    `json:"server_id,omitempty"`
}

// FanOutResult contains the result of checking group members' status
type FanOutResult struct {
	GroupID        uuid.UUID              `json:"group_id"`
	TotalMembers   int                    `json:"total_members"`
	OnlineMembers  []MemberStatus         `json:"online_members"`
	OfflineMembers []MemberStatus         `json:"offline_members"`
	ServerGroups   map[string][]uuid.UUID `json:"server_groups"` // serverID -> userIDs
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8083"
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

	service := &GroupService{
		redis: rdb,
		db:    database,
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
	router.HandleFunc("/groups/{groupId}/members", service.GetGroupMembers).Methods("GET")
	router.HandleFunc("/groups/{groupId}/fanout", service.GetFanOutInfo).Methods("GET")
	router.HandleFunc("/groups/{groupId}/online", service.GetOnlineMembers).Methods("GET")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("👥 Group Service listening on port %s", port)
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

// GetGroupMembers returns all members of a group
func (s *GroupService) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr := vars["groupId"]

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	members, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		http.Error(w, "Failed to get group members", http.StatusInternalServerError)
		return
	}

	userIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"group_id": groupID,
		"user_ids": userIDs,
		"count":    len(userIDs),
	}); err != nil {
		log.Printf("Warning: failed to encode response: %v", err)
	}
}

// GetFanOutInfo returns detailed info for message fan-out
// This is the key endpoint used for group message delivery
func (s *GroupService) GetFanOutInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr := vars["groupId"]

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Get all group members
	members, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		http.Error(w, "Failed to get group members", http.StatusInternalServerError)
		return
	}

	result := &FanOutResult{
		GroupID:        groupID,
		TotalMembers:   len(members),
		OnlineMembers:  make([]MemberStatus, 0),
		OfflineMembers: make([]MemberStatus, 0),
		ServerGroups:   make(map[string][]uuid.UUID),
	}

	ctx := context.Background()

	// Check each member's status
	for _, member := range members {
		// Check presence
		presenceKey := "presence:" + member.UserID.String()
		val, err := s.redis.Get(ctx, presenceKey).Result()

		if err != nil || val != "online" {
			// User is offline
			result.OfflineMembers = append(result.OfflineMembers, MemberStatus{
				UserID:   member.UserID,
				IsOnline: false,
			})
			continue
		}

		// User is online - find their server(s)
		connectionKey := "connections:" + member.UserID.String()
		servers, err := s.redis.HGetAll(ctx, connectionKey).Result()

		if err != nil || len(servers) == 0 {
			// No active connections found
			result.OfflineMembers = append(result.OfflineMembers, MemberStatus{
				UserID:   member.UserID,
				IsOnline: false,
			})
			continue
		}

		// Get unique servers for this user
		serverSet := make(map[string]bool)
		for _, serverID := range servers {
			serverSet[serverID] = true

			// Group users by server for parallel delivery
			if result.ServerGroups[serverID] == nil {
				result.ServerGroups[serverID] = make([]uuid.UUID, 0)
			}
			// Only add once per server
			found := false
			for _, uid := range result.ServerGroups[serverID] {
				if uid == member.UserID {
					found = true
					break
				}
			}
			if !found {
				result.ServerGroups[serverID] = append(result.ServerGroups[serverID], member.UserID)
			}
		}

		// Pick first server for status
		var firstServer string
		for s := range serverSet {
			firstServer = s
			break
		}

		result.OnlineMembers = append(result.OnlineMembers, MemberStatus{
			UserID:   member.UserID,
			IsOnline: true,
			ServerID: firstServer,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Warning: failed to encode result: %v", err)
	}
}

// GetOnlineMembers returns just the online members (quick check)
func (s *GroupService) GetOnlineMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr := vars["groupId"]

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	members, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		http.Error(w, "Failed to get group members", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	onlineUsers := make([]uuid.UUID, 0)

	for _, member := range members {
		presenceKey := "presence:" + member.UserID.String()
		val, _ := s.redis.Get(ctx, presenceKey).Result()
		if val == "online" {
			onlineUsers = append(onlineUsers, member.UserID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"group_id":      groupID,
		"online_users":  onlineUsers,
		"online_count":  len(onlineUsers),
		"total_members": len(members),
	}); err != nil {
		log.Printf("Warning: failed to encode response: %v", err)
	}
}
