package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"schedule-mcp/db"
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

	queries = db.New(dbPool)

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

	// 3. Setup Echo Server
	e := echo.New()

	// Standard Echo Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CSRF Setup
	csrfKey := os.Getenv("CSRF_KEY")
	if len(csrfKey) < 32 {
		csrfKey = "01234567890123456789012345678901" // 32-byte fallback
	}
	useSecure := os.Getenv("ENV") == "production"
	if os.Getenv("LOCAL_DEV") == "true" {
		useSecure = false
	}

	csrfMiddleware := echo.WrapMiddleware(csrf.Protect(
		[]byte(csrfKey),
		csrf.Secure(useSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.TrustedOrigins([]string{"localhost:8080", "127.0.0.1:8080"}),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("CSRF Failure for %s %s: %v", r.Method, r.URL.Path, csrf.FailureReason(r))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Forbidden - CSRF error: %v", csrf.FailureReason(r)),
			})
		})),
	))

	// MCP SSE Handlers (using net/http compatible wrappers)
	sseServer := server.NewSSEServer(mcpServer)
	e.GET("/sse", echo.WrapHandler(NetHttpAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		go GlobalSessionManager.MaintainHeartbeat(r.Context(), userID, mcpServer)
		sseServer.SSEHandler().ServeHTTP(w, r)
	}), mcpServer)))
	e.POST("/message", echo.WrapHandler(NetHttpAuthMiddleware(sseServer.MessageHandler(), mcpServer)))

	// Telemetry & Observability
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/api/healthz", func(c echo.Context) error {
		if err := dbPool.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database unavailable"})
		}
		if err := redisClient.Ping(c.Request().Context()).Err(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Redis unavailable"})
		}
		return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "OK"})
	})

	// Static files
	e.Static("/static", "static")
	e.Static("/assets", "frontend/dist/assets")

	// Auth API Handlers
	authGroup := e.Group("/api/auth", csrfMiddleware)
	authGroup.GET("/csrf", apiCSRFHandler)
	authGroup.POST("/signup", apiSignupHandler)
	authGroup.POST("/login", apiLoginHandler)
	authGroup.POST("/logout", apiLogoutHandler)

	// Protected API Handlers
	api := e.Group("/api", csrfMiddleware, EchoSessionMiddleware)
	api.GET("/dashboard", apiDashboardHandler)
	api.POST("/rotate-api-key", apiRotateAPIKeyHandler)
	
	staff := api.Group("", EchoRequireRole("staff", "admin"))
	staff.GET("/monitor", apiMonitorHandler)
	
	admin := api.Group("/admin", EchoRequireRole("admin"))
	admin.GET("/users", apiAdminUsersHandler)
	admin.POST("/users/update", apiAdminUpdateUserHandler)

	// Catch-all handler for React SPA
	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		// Check if file exists in dist, otherwise serve index.html
		fpath := "frontend/dist" + path
		if _, err := os.Stat(fpath); os.IsNotExist(err) || path == "/" {
			return c.File("frontend/dist/index.html")
		}
		return c.File(fpath)
	})

	// 4. Start Background Scheduler & Reaper
	go runScheduler(ctx, mcpServer)
	go runReaper(ctx)

	// 5. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("shuting down the server: %v", err)
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
	err = queries.RevertProcessingTasks(context.Background(), pgtype.Text{String: workerID, Valid: true})
	if err != nil {
		log.Printf("Failed to revert processing tasks: %v", err)
	} else {
		log.Printf("Reverted tasks for worker %s", workerID)
	}

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed: %+v", err)
	}
	log.Println("Server exited properly")
}

func getUser(r *http.Request) *User {
	user, _ := r.Context().Value("user").(*User)
	return user
}
