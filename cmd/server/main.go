package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"schedule-mcp/db"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"github.com/exaring/otelpgx"
)

func initRedis(redisUrl string) {
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}
	opts, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	RedisClient = redis.NewClient(opts)

	_, err = RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

func initTracer(ctx context.Context) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "jaeger:4317"
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		log.Printf("Failed to create OTLP trace exporter: %v", err)
		return func(context.Context) error { return nil }
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("scheduled-actions"),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
		return func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}
}

func main() {
	hostname, _ := os.Hostname()
	workerID = fmt.Sprintf("%s-%d", hostname, time.Now().UTC().UnixNano())

	cfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		log.Fatalf("Invalid runtime configuration: %v", err)
	}
	appConfig = cfg

	// 0. Initialize Encryption
	initCrypto()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer := initTracer(ctx)
	defer shutdownTracer(ctx)

	// 1. Initialize PostgreSQL Connection Pool
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/mcp?sslmode=disable"
	}

	pgxCfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatalf("Unable to parse database URL: %v", err)
	}
	pgxCfg.ConnConfig.Tracer = otelpgx.NewTracer()

	dbPool, err = pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Initialize and run migrations using golang-migrate
	// Ensure the migrations path is correct relative to the executable.
	// If running from inside a container, it might need adjustment.
	// For this example, we assume './migrations' relative to the executable.
	// The DB URL should be derived from environment variables.
	// Ensure database connection is valid before attempting migrations
	ctxForMigrations := context.Background() // Use a background context for migrations
	if err := dbPool.Ping(ctxForMigrations); err != nil {
		log.Fatalf("Database not available for migrations: %v", err)
	}

	// Use migrate.New with the database URL for simplicity
	m, err := migrate.New(
		"file://migrations", // Path to migration files
		dbUrl,               // Database connection URL
	)

	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}

	// Apply pending migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		// If migration fails, log the error and consider stopping the application startup
		// depending on whether migrations are critical for startup.
		log.Fatalf("Failed to apply migrations: %v", err)
	} else if err == migrate.ErrNoChange {
		log.Println("No pending migrations to apply.")
	} else {
		log.Println("Migrations applied successfully.")
	}

	// Optional: You can also implement rollback logic or version checking here if needed.
	// For example, to check current version:
	// version, dirty, err := m.Version()
	// if err != nil && err != migrate.ErrNilVersion {
	//     log.Printf("Failed to get migration version: %v", err)
	// } else {
	//     log.Printf("Current migration version: %d, dirty: %t", version, dirty)
	// }

	queries = db.New(dbPool)

	// 1.5 Initialize Redis Client
	initRedis(cfg.RedisURL)
	defer RedisClient.Close()

	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		SubscribeToEvents(context.Background(), func(ctx context.Context, event PubSubEvent) {
			handleSystemEvent(ctx, event)
		})
	}()

	globalRateLimiter.client = RedisClient
	GlobalSessionManager.Init(RedisClient)

	// 2. Initialize MCP Server
	mcpServer := server.NewMCPServer("scheduled-actions", "1.0.0")

	// Register Tools
	registerTools(mcpServer)

	// 3. Setup Echo Server
	e := echo.New()

	// URL Fix Middleware must be at the very top for CSRF same-origin check
	// URL Fix Middleware must be at the very top for CSRF same-origin check
	e.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			
			// Force populate r.URL Host/Scheme from request headers if missing
			// gorilla/csrf's same-origin check requires these to be present.
			if r.URL.Host == "" {
				r.URL.Host = r.Host
			}
			if r.URL.Scheme == "" {
				if c.IsTLS() || r.Header.Get("X-Forwarded-Proto") == "https" {
					r.URL.Scheme = "https"
				} else {
					r.URL.Scheme = "http"
				}
			}

			// Special case for local dev: if Origin matches Host, ensure they are string-identical
			// to bypass any subtle gorilla/csrf comparison issues.
			if cfg.LocalDev {
				origin := r.Header.Get("Origin")
				if origin != "" {
					if u, err := url.Parse(origin); err == nil {
						// If the origin host matches our request host, align them perfectly
						if u.Host == r.Host {
							r.URL.Host = u.Host
							r.URL.Scheme = u.Scheme
						}
					}
				}
			}

			return next(c)
		}
	})

	// Standard Echo Middleware
	e.Use(otelecho.Middleware("scheduled-actions"))
	//lint:ignore SA1019 simple logger is sufficient
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(prometheusMiddleware)

	// CSRF Setup
	csrfKey := cfg.CSRFKey
	if len(csrfKey) < 32 {
		csrfKey = defaultCSRFKey // non-production fallback
	}
	useSecure := cfg.secureCookies()
	trustedOrigins := cfg.csrfTrustedOrigins()
	log.Printf("CSRF Protection enabled. Secure: %v, Trusted Origins: %v", useSecure, trustedOrigins)

	csrfCore := csrf.Protect(
		[]byte(csrfKey),
		csrf.Secure(useSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.TrustedOrigins(trustedOrigins),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			failureReason := csrf.FailureReason(r)
			origin := r.Header.Get("Origin")
			referer := r.Header.Get("Referer")
			host := r.Host

			log.Printf("CSRF Failure for %s %s: %v", r.Method, r.URL.Path, failureReason)
			log.Printf("CSRF Debug - Origin: %q, Referer: %q, Host: %q, RequestURL: %q", origin, referer, host, r.URL.String())
			
			errorMessage := fmt.Sprintf("Forbidden - CSRF error: %v", failureReason)
			if strings.Contains(failureReason.Error(), "Origin") {
				errorMessage = fmt.Sprintf("CSRF Origin mismatch: %q is not in trusted list. Host is %q, RequestURL is %q.", origin, host, r.URL.String())
			} else if strings.Contains(failureReason.Error(), "token") {
				errorMessage = "CSRF token missing or mismatch. Ensure cookies are enabled and the X-CSRF-Token header is set."
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   errorMessage,
			})
		})),
	)

	// Wrap CSRF with absolute reliability for local development
	csrfMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		// Initialize the standard handler
		csrfHandler := echo.WrapMiddleware(func(handler http.Handler) http.Handler {
			return csrfCore(handler)
		})(next)

		return func(c echo.Context) error {
			r := c.Request()

			if cfg.LocalDev {
				origin := r.Header.Get("Origin")
				// If origin is local, skip the check entirely in dev.
				// This is the only way to reliably bypass gorilla/csrf's strict internal checks in complex Docker/network setups.
				isLocalOrigin := origin == "" ||
					strings.Contains(origin, "localhost") ||
					strings.Contains(origin, "127.0.0.1") ||
					strings.Contains(origin, "192.168.") ||
					strings.Contains(origin, "10.") ||
					strings.Contains(origin, "172.17.") ||
					strings.Contains(origin, "172.18.") ||
					strings.Contains(origin, "172.19.") ||
					strings.Contains(origin, "172.20.") ||
					strings.Contains(origin, "172.21.") ||
					strings.Contains(origin, "172.22.") ||
					strings.Contains(origin, "172.23.") ||
					strings.Contains(origin, "172.24.") ||
					strings.Contains(origin, "172.25.") ||
					strings.Contains(origin, "172.26.") ||
					strings.Contains(origin, "172.27.") ||
					strings.Contains(origin, "172.28.") ||
					strings.Contains(origin, "172.29.") ||
					strings.Contains(origin, "172.30.") ||
					strings.Contains(origin, "172.31.")

				if isLocalOrigin {
					// gorilla/csrf supports skipping the check via UnsafeSkipCheck
					c.SetRequest(csrf.UnsafeSkipCheck(r))
				}
			}

			return csrfHandler(c)
		}
	}

	// MCP SSE Handlers (using net/http compatible wrappers)
	sseServer := server.NewSSEServer(mcpServer)
	e.GET("/sse", echo.WrapHandler(NetHttpAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
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
		if err := RedisClient.Ping(c.Request().Context()).Err(); err != nil {
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

	// Unified V1 API
	v1 := e.Group("/api/v1")
	v1.POST("/webhooks/inbound/:token", handleInboundWebhook)

	// Protected API Handlers (v1)
	api := v1.Group("", csrfMiddleware, EchoSessionMiddleware, EchoRateLimitMiddleware)
	api.GET("/dashboard", apiDashboardHandler)
	api.POST("/rotate-api-key", apiRotateAPIKeyHandler)
	api.GET("/tasks", apiListTasksHandler)
	api.POST("/tasks", apiCreateTaskHandler)
	api.GET("/tasks/export", apiExportTasksHandler)
	api.POST("/tasks/import", apiImportTasksHandler)
	api.POST("/tasks/:id/link", apiLinkTaskHandler)
	api.POST("/tasks/:id/pause", apiPauseTaskHandler)
	api.POST("/tasks/:id/resume", apiResumeTaskHandler)
	api.DELETE("/tasks/:id", apiDeleteTaskHandler)
	api.PATCH("/tasks/:id", apiUpdateTaskHandler)
	api.GET("/tasks/:id/versions", apiListTaskVersionsHandler)
	api.GET("/tasks/:id/executions", apiListTaskExecutionsHandler)
	api.GET("/tasks/:id/traces/:execution_id", apiGetExecutionTracesHandler)
	api.POST("/tasks/:id/restore/:version_id", apiRestoreTaskVersionHandler)
	api.POST("/tasks/:id/approve", apiApproveTaskHandler)
	api.POST("/tasks/:id/deny", apiDenyTaskHandler)
	api.POST("/tasks/:id/route", apiManualRouteHandler)
	api.GET("/events", apiEventsHandler)
	api.GET("/secrets", apiListSecretsHandler)
	api.POST("/secrets", apiUpsertSecretHandler)
	api.DELETE("/secrets/:name", apiDeleteSecretHandler)
	api.GET("/webhooks", apiListWebhooksHandler)
	api.POST("/webhooks", apiCreateWebhookHandler)
	api.DELETE("/webhooks/:id", apiDeleteWebhookHandler)
	api.GET("/webhooks/:id/deliveries", apiWebhookDeliveriesHandler)

	// Additional v1 routes moved from legacy v1 block
	api.GET("/workspaces", handleGetWorkspaces)
	api.POST("/workspaces", handleCreateWorkspace)
	api.GET("/workspaces/:id/env", handleListWorkspaceEnvVars)
	api.POST("/workspaces/:id/env", handleUpsertWorkspaceEnvVar)
	api.DELETE("/workspaces/:id/env/:name", handleDeleteWorkspaceEnvVar)
	api.GET("/templates", handleListPublicTemplates)
	api.POST("/templates", handleCreateTemplate)
	api.POST("/templates/:id/increment-uses", handleIncrementTemplateUses)
	api.POST("/blueprints/deploy", apiDeployBlueprintHandler)

	// Phase 8: The Monetization API (Billing)
	api.POST("/billing/create-checkout-session", apiCreateCheckoutSession)

	staff := api.Group("", EchoRequireRole("staff", "admin"))
	staff.GET("/monitor", apiMonitorHandler)

	admin := api.Group("/admin", EchoRequireRole("admin"))
	admin.GET("/users", apiAdminUsersHandler)
	admin.POST("/users/update", apiAdminUpdateUserHandler)
	admin.GET("/audit-logs", apiAdminAuditLogsHandler)
	admin.GET("/usage", apiAdminUsageHandler)
	admin.GET("/insights", handleGetSystemInsights)
	admin.GET("/analytics/trends", handleGetTrends)
	admin.GET("/workers", handleGetWorkers)
	admin.GET("/seo", apiGetSEOHandler)
	admin.POST("/seo", apiUpdateSEOHandler)

	e.POST("/webhooks/stripe", apiStripeWebhook)

	// Catch-all handler for React SPA
	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		if path == "/" {
			return c.File("frontend/dist/index.html")
		}

		// Clean the path to prevent traversal
		cleanPath := filepath.Clean(path)
		if strings.Contains(cleanPath, "..") {
			return c.File("frontend/dist/index.html")
		}

		fpath := filepath.Join("frontend/dist", cleanPath)
		if info, err := os.Stat(fpath); err != nil || info.IsDir() {
			return c.File("frontend/dist/index.html")
		}
		return c.File(fpath)
	})

	// 4. Start Background Scheduler & Reaper
	go listenForTaskClaims(ctx, dbUrl)
	go runScheduler(ctx)
	go runReaper(ctx)
	go runWorkerHeartbeat(ctx)

	// Bootstrap Admin
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail != "" {
		log.Printf("Bootstrapping admin role for: %s", adminEmail)
		err := queries.SetUserRoleByEmail(ctx, db.SetUserRoleByEmailParams{
			Role:  pgtype.Text{String: "admin", Valid: true},
			Email: pgtype.Text{String: adminEmail, Valid: true},
		})
		if err != nil {
			log.Printf("Failed to bootstrap admin: %v", err)
		}
	}

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

	if dbPool != nil {
		dbPool.Close()
	}
	if RedisClient != nil {
		RedisClient.Close()
	}

	log.Println("Server exited properly")
}
