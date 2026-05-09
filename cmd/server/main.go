package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	hostname, _ := os.Hostname()
	workerID = fmt.Sprintf("%s-%d", hostname, time.Now().UTC().UnixNano())

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

	// CSRF Setup
	csrfKey := os.Getenv("CSRF_KEY")
	if len(csrfKey) < 32 {
		csrfKey = "01234567890123456789012345678901" // 32-byte fallback
	}
	
	// Determine if we should use Secure cookies.
	// Only use Secure if ENV is production AND we are NOT on localhost.
	useSecure := os.Getenv("ENV") == "production"
	if os.Getenv("LOCAL_DEV") == "true" {
		useSecure = false
	}

	csrfMiddleware := csrf.Protect(
		[]byte(csrfKey),
		csrf.Secure(useSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.TrustedOrigins([]string{"localhost:8080", "127.0.0.1:8080"}),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("CSRF Failure for %s %s: %v", r.Method, r.URL.Path, csrf.FailureReason(r))
			sendJSON(w, http.StatusForbidden, APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Forbidden - CSRF error: %v", csrf.FailureReason(r)),
			})
		})),
	)

	// 2. Initialize MCP Server
	mcpServer := server.NewMCPServer("scheduled-actions", "1.0.0")

	// Register Tools
	registerTools(mcpServer)

	// 3. Setup SSE Handler with Auth Middleware
	mux := http.NewServeMux()

	// Assuming the SDK provides an SSE handler endpoint.
	sseServer := server.NewSSEServer(mcpServer)
	mux.Handle("/sse", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		go GlobalSessionManager.MaintainHeartbeat(r.Context(), userID, mcpServer)
		sseServer.SSEHandler().ServeHTTP(w, r)
	}), mcpServer))
	mux.Handle("/message", authMiddleware(sseServer.MessageHandler(), mcpServer))

	// Phase 4: Telemetry & Observability
	mux.Handle("/metrics", promhttp.Handler())

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve React Frontend assets
	frontendFS := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/assets/", frontendFS)

	// Catch-all handler for React SPA
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 4 && path[:5] == "/api/" {
			http.NotFound(w, r)
			return
		}
		
		fpath := "frontend/dist" + path
		if _, err := os.Stat(fpath); os.IsNotExist(err) || path == "/" {
			http.ServeFile(w, r, "frontend/dist/index.html")
		} else {
			frontendFS.ServeHTTP(w, r)
		}
	})

	// Auth API Handlers
	mux.Handle("/api/auth/csrf", csrfMiddleware(http.HandlerFunc(apiCSRFHandler)))
	mux.Handle("/api/auth/signup", csrfMiddleware(http.HandlerFunc(apiSignupHandler)))
	mux.Handle("/api/auth/login", csrfMiddleware(http.HandlerFunc(apiLoginHandler)))
	mux.Handle("/api/auth/logout", csrfMiddleware(http.HandlerFunc(apiLogoutHandler)))

	// Health Check
	mux.Handle("/api/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := dbPool.Ping(r.Context()); err != nil {
			sendJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database unavailable"})
			return
		}
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			sendJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Redis unavailable"})
			return
		}
		sendJSON(w, http.StatusOK, APIResponse{Success: true, Message: "OK"})
	}))

	// Protected API Handlers
	mux.Handle("/api/dashboard", csrfMiddleware(sessionMiddleware(http.HandlerFunc(apiDashboardHandler))))
	mux.Handle("/api/rotate-api-key", csrfMiddleware(sessionMiddleware(http.HandlerFunc(apiRotateAPIKeyHandler))))
	mux.Handle("/api/monitor", csrfMiddleware(sessionMiddleware(RequireRole("staff", "admin")(http.HandlerFunc(apiMonitorHandler)))))
	mux.Handle("/api/admin/users", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(apiAdminUsersHandler)))))
	mux.Handle("/api/admin/users/update", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(apiAdminUpdateUserHandler)))))

	// 4. Start Background Scheduler & Reaper
	go runScheduler(ctx, mcpServer)
	go runReaper(ctx)

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

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

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

func getUser(r *http.Request) *User {
	user, _ := r.Context().Value("user").(*User)
	return user
}
