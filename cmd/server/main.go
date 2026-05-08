package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v78/webhook"
)

func main() {
	hostname, _ := os.Hostname()
	workerID = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize PostgreSQL Connection Pool
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/mcp?sslmode=disable"
	}
	
	var err error
	dbPool, err = pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	// 1.5 Initialize Redis Client
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}
	redisClient = redis.NewClient(opt)
	defer redisClient.Close()
	
	// Check Redis Connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}

	globalRateLimiter.client = redisClient
	GlobalSessionManager.Init(redisClient)

	// 2. Initialize MCP Server
	mcpServer := server.NewMCPServer("scheduled-actions", "1.0.0")

	// Register Tools
	registerTools(mcpServer)

	// 3. Setup SSE Handler with Auth Middleware
	mux := http.NewServeMux()

	// Assuming the SDK provides an SSE handler endpoint.
	// We wrap it with auth middleware to extract X-API-Key and verify user_id
	sseServer := server.NewSSEServer(mcpServer)
	mux.Handle("/sse", authMiddleware(sseServer.SSEHandler(), mcpServer))
	mux.Handle("/message", authMiddleware(sseServer.MessageHandler(), mcpServer))

	// Phase 4: Telemetry & Observability
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := dbPool.Ping(r.Context()); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Phase 8: The Monetization API (Billing)
	mux.HandleFunc("/webhooks/stripe", func(w http.ResponseWriter, r *http.Request) {
		endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if endpointSecret == "" {
			log.Println("STRIPE_WEBHOOK_SECRET not set, cannot verify webhook.")
			http.Error(w, "Server Configuration Error", http.StatusInternalServerError)
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Verify Stripe signature
		event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), endpointSecret)
		if err != nil {
			log.Printf("Error verifying webhook signature: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if event.Type == "checkout.session.completed" {
			// Extract email or user ID from the session to upgrade
			var session map[string]interface{}
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				log.Printf("Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			
			// Assuming customer_email or client_reference_id holds the user info
			email, _ := session["customer_details"].(map[string]interface{})["email"].(string)
			if email != "" {
				_, err = dbPool.Exec(r.Context(), "UPDATE users SET tier = 'pro' WHERE email = $1", email)
				if err != nil {
					log.Printf("Failed to upgrade user %s: %v", email, err)
				} else {
					log.Printf("Upgraded user %s to pro tier via webhook", email)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.URL.Query().Get("api_key")
		if apiKey == "" {
			http.Error(w, "Missing api_key in URL parameters", http.StatusBadRequest)
			return
		}

		var email, tier string
		err := dbPool.QueryRow(r.Context(), "SELECT email, tier FROM users WHERE api_key = $1", apiKey).Scan(&email, &tier)
		if err != nil {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		html := fmt.Sprintf(`
		<html><body>
			<h1>Schedule MCP Dashboard</h1>
			<p>User Email: <strong>%s</strong></p>
			<p>Your API Key: <strong>%s</strong></p>
			<p>Current Tier: <strong>%s</strong></p>
			<a href="https://buy.stripe.com/checkout">Upgrade to Pro</a>
		</body></html>
		`, email, apiKey, tier)
		w.Write([]byte(html))
	})

	// 4. Start Background Scheduler
	go runScheduler(ctx, mcpServer)

	// 5. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Cancel root context to stop runScheduler from claiming new tasks
	cancel()

	// Wait for ongoing tasks
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for all running worker goroutines to finish or timeout
	done := make(chan struct{})
	go func() {
		workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All background tasks completed normally.")
	case <-shutdownCtx.Done():
		log.Println("Timeout waiting for background tasks.")
	}

	// Graceful Shutdown Logic: Revert 'processing' tasks back to 'active' ONLY for this worker
	log.Printf("Reverting tasks locked by worker %s to active...", workerID)
	res, err := dbPool.Exec(context.Background(), "UPDATE tasks SET status = 'active', locked_by = NULL WHERE locked_by = $1", workerID)
	if err != nil {
		log.Printf("Failed to revert processing tasks: %v", err)
	} else {
		log.Printf("Reverted %d tasks", res.RowsAffected())
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed: %+v", err)
	}
	log.Println("Server exited properly")
}
