package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaydenbeard/messaging-app/internal/config"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Scheduler runs periodic maintenance jobs:
// - Disappearing messages cleanup
// - Expired media cleanup
// - Key rotation reminders
// - Pre-key replenishment checks
// - Rate limit cleanup
func main() {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://messaging:messaging@localhost:5432/messaging?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	// Connect to Redis with optional password
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: os.Getenv("REDIS_PASSWORD"), // Empty string if not set
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("Failed to close Redis: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("🕐 Scheduler started")

	// Start scheduled jobs
	go runDisappearingMessagesCleanup(ctx, db)
	go runExpiredMediaCleanup(ctx, db)
	go runKeyRotationCheck(ctx, db, rdb)
	go runJWTSecretRotation(ctx)
	go runPreKeyReplenishmentCheck(ctx, db, rdb)
	go runRateLimitCleanup(ctx, db)
	go runVerificationCodeCleanup(ctx, db)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Scheduler shutting down...")
	cancel()
}

// runDisappearingMessagesCleanup deletes expired messages every minute
func runDisappearingMessagesCleanup(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var deletedCount int
			err := db.QueryRowContext(ctx, "SELECT cleanup_expired_messages()").Scan(&deletedCount)
			if err != nil {
				log.Printf("Error cleaning up expired messages: %v", err)
				continue
			}
			if deletedCount > 0 {
				log.Printf("🗑️ Cleaned up %d expired messages", deletedCount)
			}
		}
	}
}

// runExpiredMediaCleanup deletes expired media every 5 minutes
func runExpiredMediaCleanup(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var deletedCount int
			err := db.QueryRowContext(ctx, "SELECT cleanup_expired_media()").Scan(&deletedCount)
			if err != nil {
				log.Printf("Error cleaning up expired media: %v", err)
				continue
			}
			if deletedCount > 0 {
				log.Printf("🗑️ Cleaned up %d expired media files", deletedCount)
			}
		}
	}
}

// runKeyRotationCheck checks for users who need key rotation (weekly)
func runKeyRotationCheck(ctx context.Context, db *sql.DB, rdb *redis.Client) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Find users whose signed pre-key is older than 7 days
			rows, err := db.QueryContext(ctx, `
				SELECT user_id FROM users 
				WHERE signed_prekey_updated_at < NOW() - INTERVAL '7 days'
				AND is_active = true
				LIMIT 100
			`)
			if err != nil {
				log.Printf("Error checking key rotation: %v", err)
				continue
			}

			var usersNeedingRotation []string
			for rows.Next() {
				var userID string
				if err := rows.Scan(&userID); err != nil {
					continue
				}
				usersNeedingRotation = append(usersNeedingRotation, userID)
			}
			if err := rows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}

			if len(usersNeedingRotation) > 0 {
				log.Printf("🔑 %d users need key rotation", len(usersNeedingRotation))

				// Publish notification to each user to rotate keys
				for _, userID := range usersNeedingRotation {
					rdb.Publish(ctx, "notifications:"+userID, `{"type":"key_rotation_needed"}`)
				}
			}
		}
	}
}

// runPreKeyReplenishmentCheck finds users low on pre-keys
func runPreKeyReplenishmentCheck(ctx context.Context, db *sql.DB, rdb *redis.Client) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Find users with less than 20 unused pre-keys
			rows, err := db.QueryContext(ctx, `
				SELECT u.user_id, COUNT(p.id) as prekey_count
				FROM users u
				LEFT JOIN prekeys p ON u.user_id = p.user_id AND p.used_at IS NULL
				WHERE u.is_active = true
				GROUP BY u.user_id
				HAVING COUNT(p.id) < 20
				LIMIT 100
			`)
			if err != nil {
				log.Printf("Error checking pre-key counts: %v", err)
				continue
			}

			var usersNeedingPrekeys []string
			for rows.Next() {
				var userID string
				var count int
				if err := rows.Scan(&userID, &count); err != nil {
					continue
				}
				usersNeedingPrekeys = append(usersNeedingPrekeys, userID)
			}
			if err := rows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}

			if len(usersNeedingPrekeys) > 0 {
				log.Printf("🔐 %d users need pre-key replenishment", len(usersNeedingPrekeys))

				for _, userID := range usersNeedingPrekeys {
					rdb.Publish(ctx, "notifications:"+userID, `{"type":"prekey_replenishment_needed"}`)
				}
			}
		}
	}
}

// runRateLimitCleanup cleans up old rate limit entries
func runRateLimitCleanup(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := db.ExecContext(ctx, "SELECT cleanup_rate_limits()")
			if err != nil {
				log.Printf("Error cleaning up rate limits: %v", err)
			}
		}
	}
}

// runVerificationCodeCleanup cleans up expired verification codes
func runVerificationCodeCleanup(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := db.ExecContext(ctx, "SELECT cleanup_expired_codes()")
			if err != nil {
				log.Printf("Error cleaning up verification codes: %v", err)
			}
		}
	}
}

// runJWTSecretRotation automatically rotates JWT secrets based on configured interval
func runJWTSecretRotation(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	log.Println("🔄 JWT secret rotation scheduler started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if rotation is needed
			if config.ShouldRotate() {
				log.Println("🔄 JWT secret rotation triggered - generating new secret")

				// Generate a new cryptographically secure secret
				newSecretBytes := make([]byte, 64) // 512 bits
				if _, err := rand.Read(newSecretBytes); err != nil {
					log.Printf("Error generating new JWT secret: %v", err)
					continue
				}
				newSecret := hex.EncodeToString(newSecretBytes)

				// Rotate the secret
				if err := config.RotateSecret(newSecret); err != nil {
					log.Printf("Error rotating JWT secret: %v", err)
					continue
				}

				log.Println("✅ JWT secret rotation completed successfully")
				log.Println("ℹ️  Transition period active - both old and new secrets will be accepted")
			}
		}
	}
}
