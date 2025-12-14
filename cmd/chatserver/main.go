package main

import (
	"context"
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
	"github.com/jaydenbeard/messaging-app/internal/handlers"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/pubsub"
	"github.com/jaydenbeard/messaging-app/internal/registry"
	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/jaydenbeard/messaging-app/internal/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
	// Load configuration with secure JWT secret handling
	cfg := config.Load()

	// Initialize JWT key manager with secure secret
	config.InitializeKeyManager(cfg.JWTSecret)

	// Validate JWT secret meets security requirements
	if err := config.ValidateJWTSecret(cfg.JWTSecret); err != nil {
		log.Fatalf("FATAL: JWT secret validation failed: %v", err)
	}

	log.Printf("🚀 Starting Chat Server: %s", cfg.ServerID)

	// Initialize database connection
	database, err := db.NewPostgresDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Initialize Redis connection
	redisClient, err := pubsub.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("Warning: failed to close Redis: %v", err)
		}
	}()

	// Initialize service registry (Consul)
	serviceRegistry, err := registry.NewConsulRegistry(cfg.ConsulURL, cfg.ServerID, cfg.ServerPort)
	if err != nil {
		log.Fatalf("Failed to connect to Consul: %v", err)
	}

	// Register this server with service discovery
	if err := serviceRegistry.Register(); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	defer func() {
		if err := serviceRegistry.Deregister(); err != nil {
			log.Printf("Warning: failed to deregister service: %v", err)
		}
	}()

	// Initialize JWT key manager with secure secret
	config.InitializeKeyManager(cfg.JWTSecret)

	// Initialize key rotation scheduler
	keyRotationScheduler := security.NewKeyRotationScheduler()

	// Set rotation interval (default 24 hours)
	keyRotationScheduler.SetRotationInterval(24 * time.Hour)

	// Start automatic key rotation
	keyRotationScheduler.Start()

	// Initialize audit logger for security event tracking
	auditLogger := security.NewAuditLogger(database.GetDB())

	// Initialize auth service with secure JWT secret management
	authService, err := auth.NewAuthService(database, config.GetCurrentSecret())
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}

	// Initialize WebSocket hub with HMAC secret for message authentication
	hmacSecret := os.Getenv("HMAC_SECRET")
	hub := websocket.NewHub(cfg.ServerID, redisClient, database, hmacSecret, auditLogger)
	go hub.Run()

	// Subscribe to cross-server messages and presence updates
	go redisClient.SubscribeToMessages(hub)
	go redisClient.SubscribeToServerMessages(cfg.ServerID, hub)
	go redisClient.SubscribeToPresenceUpdates(hub)

	// Setup HTTP router
	router := mux.NewRouter()

	// Health check endpoint (for load balancer)
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Security endpoints
	router.HandleFunc("/csp-report", handlers.CSPReportHandler()).Methods("POST")

	// API Documentation (Swagger UI)
	router.HandleFunc("/api/docs", handlers.SwaggerUI()).Methods("GET")
	router.HandleFunc("/api/docs/", handlers.SwaggerUI()).Methods("GET")
	router.HandleFunc("/api/docs/openapi.yaml", handlers.OpenAPISpec("docs/openapi.yaml")).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Create enhanced rate limiter with multi-tier protection and Redis support
	enhancedRateLimiter := middleware.NewEnhancedRateLimiter(&middleware.RateLimitConfig{
		IPLimits:       make(map[string]*middleware.TieredLimitConfig),
		UserLimits:     make(map[string]*middleware.TieredLimitConfig),
		EndpointLimits: make(map[string]*middleware.TieredLimitConfig),
		GlobalLimits: &middleware.TieredLimitConfig{
			Normal: &middleware.LimitConfig{
				MaxRequests: 1000,
				Window:      1 * time.Minute,
			},
			Strict: &middleware.LimitConfig{
				MaxRequests: 500,
				Window:      1 * time.Minute,
			},
		},
		AbuseDetection: &middleware.AbuseDetectionConfig{
			Threshold:          100,
			Window:             5 * time.Minute,
			PenaltyDuration:    15 * time.Minute,
			StrictModeDuration: 30 * time.Minute,
		},
	}, redisClient.GetClient())

	// Set up specific endpoint configurations
	// SMS endpoints - very strict to prevent SMS spam
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/auth/request-code", true)

	// Auth endpoints - strict but allow reasonable login attempts
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/auth/verify", true)
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/auth/register", true)
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/auth/login", true)

	// Device approval endpoints - prevent enumeration attacks
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/device-approval/request", true)
	enhancedRateLimiter.SetEndpointStrictMode("POST /api/v1/device-approval/verify", true)
	enhancedRateLimiter.SetEndpointStrictMode("GET /api/v1/device-approval/{requestId}/status", true)

	// Search endpoints - prevent scraping
	enhancedRateLimiter.SetEndpointStrictMode("GET /api/v1/users/search", true)

	// Auth routes (no auth required, but rate limited)
	api.Handle("/auth/request-code", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.RequestVerificationCode(authService, auditLogger)))).Methods("POST")
	api.Handle("/auth/verify", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.VerifyCode(authService, database)))).Methods("POST")
	api.Handle("/auth/register", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.Register(authService, database)))).Methods("POST")
	api.Handle("/auth/login", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.Login(authService, database)))).Methods("POST")
	api.HandleFunc("/auth/refresh", handlers.RefreshToken(authService)).Methods("POST")

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware(authService, nil))

	// User routes
	protected.HandleFunc("/users/me", handlers.GetCurrentUser(database)).Methods("GET")
	protected.HandleFunc("/users/me", handlers.UpdateUser(database)).Methods("PUT", "PATCH")
	protected.HandleFunc("/users/me", handlers.DeleteUser(database)).Methods("DELETE")
	protected.HandleFunc("/users/me/prekeys", handlers.UploadPrekeys(database)).Methods("POST")
	protected.HandleFunc("/users/{userId}/keys", handlers.GetUserKeys(database)).Methods("GET")
	protected.HandleFunc("/users/keys", handlers.UpdateKeys(database, hub)).Methods("POST")
	protected.HandleFunc("/users/{userId}/profile", handlers.GetUserProfile(database, redisClient)).Methods("GET")
	protected.HandleFunc("/users/check-username/{username}", handlers.CheckUsername(database)).Methods("GET")
	protected.Handle("/users/search", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.SearchUsers(database)))).Methods("GET")

	// Device routes
	protected.HandleFunc("/devices", handlers.GetDevices(database)).Methods("GET")
	protected.HandleFunc("/devices/{deviceId}", handlers.RemoveDevice(database)).Methods("DELETE")
	protected.HandleFunc("/devices/{deviceId}/primary", handlers.SetPrimaryDevice(database)).Methods("PUT")

	// PIN routes (server-side sync)
	protected.HandleFunc("/pin", handlers.GetPIN(database)).Methods("GET")
	protected.HandleFunc("/pin", handlers.SetPIN(database)).Methods("POST")
	protected.HandleFunc("/pin", handlers.DeletePIN(database)).Methods("DELETE")

	// Privacy settings routes
	protected.HandleFunc("/privacy", handlers.GetPrivacySettings(database)).Methods("GET")
	protected.HandleFunc("/privacy", handlers.UpdatePrivacySetting(database, hub)).Methods("POST")

	// Block user routes
	protected.HandleFunc("/users/blocked", handlers.GetBlockedUsers(database)).Methods("GET")
	protected.HandleFunc("/users/block", handlers.BlockUser(database, hub)).Methods("POST")
	protected.HandleFunc("/users/unblock", handlers.UnblockUser(database)).Methods("POST")
	protected.HandleFunc("/users/{userId}/blocked", handlers.IsBlocked(database)).Methods("GET")

	// Device approval routes (secure device linking)
	// NOTE: request/verify/status endpoints are intentionally public because new devices
	// don't have auth tokens yet. Rate limiting applied to prevent enumeration.
	router.Handle("/api/v1/device-approval/request", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.RequestDeviceApproval(database, hub)))).Methods("POST")
	router.Handle("/api/v1/device-approval/verify", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.VerifyApprovalCode(database)))).Methods("POST")
	router.Handle("/api/v1/device-approval/{requestId}/status", enhancedRateLimiter.Middleware(http.HandlerFunc(handlers.CheckApprovalStatus(database)))).Methods("GET")
	protected.HandleFunc("/device-approval/pending", handlers.GetPendingApprovals(database)).Methods("GET")
	protected.HandleFunc("/device-approval/{requestId}/approve", handlers.ApproveDevice(database, hub)).Methods("POST")
	protected.HandleFunc("/device-approval/{requestId}/deny", handlers.DenyDevice(database, hub)).Methods("POST")

	// Message routes
	protected.HandleFunc("/messages", handlers.GetMessages(database)).Methods("GET")
	protected.HandleFunc("/messages/{messageId}/status", handlers.UpdateMessageStatus(database, hub)).Methods("PUT")

	// Group routes
	protected.HandleFunc("/groups", handlers.CreateGroup(database)).Methods("POST")
	protected.HandleFunc("/groups/{groupId}", handlers.GetGroup(database)).Methods("GET")
	protected.HandleFunc("/groups/{groupId}/members", handlers.AddGroupMember(database)).Methods("POST")
	protected.HandleFunc("/groups/{groupId}/members/{userId}", handlers.RemoveGroupMember(database)).Methods("DELETE")

	// NOTE: Conversation sync happens device-to-device for security.
	// Server never stores conversation metadata (who talks to whom).
	// See WebSocket sync_request/sync_data messages for multi-device sync.

	// Media routes
	protected.HandleFunc("/media/upload-url", handlers.GetUploadURL(cfg)).Methods("POST")
	protected.HandleFunc("/media/upload", handlers.GetUploadURL(cfg)).Methods("POST") // Alias
	protected.HandleFunc("/media/download-url/{mediaId}", handlers.GetMediaURL(database, cfg)).Methods("GET")
	protected.HandleFunc("/media/{mediaId}", handlers.GetMediaURL(database, cfg)).Methods("GET")
	// Proxy endpoints (to avoid mixed content errors)
	protected.HandleFunc("/media/upload-proxy/{mediaId}", handlers.UploadProxy(cfg)).Methods("PUT", "POST")
	protected.HandleFunc("/media/download-proxy/{mediaId}", handlers.DownloadProxy(cfg)).Methods("GET")

	// WebRTC routes
	protected.HandleFunc("/rtc/turn-credentials", handlers.GetTurnCredentials()).Methods("GET")

	// WebSocket endpoint (requires auth via query param or header)
	router.HandleFunc("/ws", handlers.WebSocketHandler(hub, authService)).Methods("GET")

	// CORS configuration - restrict to known origins in production
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"https://silentrelay.com.au",
			"https://www.silentrelay.com.au",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Device-ID"},
		AllowCredentials: true,
	})

	// Create HTTP server with security timeouts to prevent Slowloris attacks
	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           corsHandler.Handler(router),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second, // Prevents Slowloris attacks (gosec G112)
	}

	// Start server in goroutine
	go func() {
		log.Printf("📡 Chat Server listening on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("🛑 Received signal %v - starting graceful shutdown...", sig)

	// Step 1: Immediately deregister from service discovery
	// This tells HAProxy to stop sending new connections to this server
	log.Println("📤 Deregistering from service discovery (HAProxy will stop routing here)...")
	if err := serviceRegistry.Deregister(); err != nil {
		log.Printf("Warning: Failed to deregister from service discovery: %v", err)
	} else {
		log.Println("✅ Deregistered from service discovery")
	}

	// Step 2: Wait for load balancer health check to detect we're gone
	// HAProxy typically checks every 2 seconds, so 5 seconds is safe
	log.Println("⏳ Waiting 5 seconds for load balancer to update...")
	time.Sleep(5 * time.Second)

	// Step 3: Stop accepting new connections and wait for existing ones to drain
	log.Println("🔌 Stopping HTTP server (new connections rejected)...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This will:
	// - Stop accepting new connections
	// - Wait for all active HTTP requests to complete (up to 30s)
	// - WebSocket connections are NOT affected by this directly
	serverShutdownDone := make(chan struct{})
	go func() {
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Warning: HTTP server shutdown error: %v", err)
		}
		close(serverShutdownDone)
	}()

	// Step 4: Gracefully close WebSocket connections
	log.Println("🔌 Closing WebSocket connections gracefully...")
	hub.Shutdown()

	// Step 5: Stop background tasks
	log.Println("⏹️ Stopping key rotation scheduler...")
	keyRotationScheduler.Stop()

	// Wait for HTTP server to finish
	<-serverShutdownDone

	log.Println("✅ Server stopped gracefully - safe to restart")
}
