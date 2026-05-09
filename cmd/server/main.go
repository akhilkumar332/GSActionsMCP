package main

import (
	"context"
	"fmt"
	"html/template"
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

	// Initialize templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// CSRF Setup
	csrfKey := os.Getenv("CSRF_KEY")
	if len(csrfKey) < 32 {
		csrfKey = "01234567890123456789012345678901" // 32-byte fallback
	}
	csrfMiddleware := csrf.Protect([]byte(csrfKey), csrf.Secure(os.Getenv("ENV") == "production"))

	// 2. Initialize MCP Server
	mcpServer := server.NewMCPServer("scheduled-actions", "1.0.0")

	// Register Tools
	registerTools(mcpServer)

	// 3. Setup SSE Handler with Auth Middleware
	mux := http.NewServeMux()

	// Assuming the SDK provides an SSE handler endpoint.
	// We wrap it with auth middleware to extract X-API-Key and verify user_id
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

	// API Auth Handlers
	mux.Handle("/api/auth/signup", csrfMiddleware(http.HandlerFunc(apiSignupHandler)))
	mux.Handle("/api/auth/login", csrfMiddleware(http.HandlerFunc(apiLoginHandler)))
	mux.Handle("/api/auth/logout", csrfMiddleware(http.HandlerFunc(apiLogoutHandler)))
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

	// API Handlers
	mux.Handle("/api/dashboard", csrfMiddleware(sessionMiddleware(http.HandlerFunc(apiDashboardHandler))))
	mux.Handle("/api/rotate-api-key", csrfMiddleware(sessionMiddleware(http.HandlerFunc(apiRotateAPIKeyHandler))))
	mux.Handle("/api/monitor", csrfMiddleware(sessionMiddleware(RequireRole("staff", "admin")(http.HandlerFunc(apiMonitorHandler)))))
	mux.Handle("/api/admin/users", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(apiAdminUsersHandler)))))
	mux.Handle("/api/admin/users/update", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(apiAdminUpdateUserHandler)))))

	// Auth Handlers
	mux.Handle("/signup", csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderTemplate(w, r, "signup", PageData{})
			return
		}
		if r.Method == http.MethodPost {
			email := r.FormValue("email")
			password := r.FormValue("password")
			_, err := RegisterUser(r.Context(), email, password)
			if err != nil {
				renderTemplate(w, r, "signup", PageData{Error: err.Error()})
				return
			}
			http.Redirect(w, r, "/login?message=Account+created+successfully", http.StatusSeeOther)
		}
	})))

	mux.Handle("/login", csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			msg := r.URL.Query().Get("message")
			renderTemplate(w, r, "login", PageData{Message: msg})
			return
		}
		if r.Method == http.MethodPost {
			email := r.FormValue("email")
			password := r.FormValue("password")
			sessionID, err := LoginUser(r.Context(), email, password)
			if err != nil {
				renderTemplate(w, r, "login", PageData{Error: "Invalid email or password"})
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				Secure:   os.Getenv("ENV") == "production",
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().UTC().Add(24 * time.Hour),
			})
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		}
	})))

	mux.Handle("/logout", csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err == nil && cookie.Value != "" {
			_, err = dbPool.Exec(r.Context(), "DELETE FROM web_sessions WHERE id = $1", cookie.Value)
			if err != nil {
				log.Printf("Error deleting session: %v", err)
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   os.Getenv("ENV") == "production",
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})))

	// Dashboard
	mux.Handle("/dashboard", csrfMiddleware(sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var taskCount int
		err := dbPool.QueryRow(r.Context(), "SELECT COUNT(*) FROM tasks WHERE user_id = $1", user.ID).Scan(&taskCount)
		if err != nil {
			log.Printf("Error fetching task count: %v", err)
		}

		renderTemplate(w, r, "dashboard", PageData{
			User:        user,
			CurrentPage: "dashboard",
			Data: map[string]interface{}{
				"TaskCount": taskCount,
			},
		})
	}))))

	mux.Handle("/rotate-api-key", csrfMiddleware(sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		user := getUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		_, err := RotateAPIKey(r.Context(), user.ID)
		if err != nil {
			log.Printf("Error rotating API key: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard?message=API+key+rotated+successfully", http.StatusSeeOther)
	}))))

	// Staff & Admin Views
	mux.Handle("/monitor", csrfMiddleware(sessionMiddleware(RequireRole("staff", "admin")(http.HandlerFunc(monitorHandler)))))
	mux.Handle("/admin/users", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(adminUsersHandler)))))
	mux.Handle("/admin/users/update", csrfMiddleware(sessionMiddleware(RequireRole("admin")(http.HandlerFunc(adminUpdateUserHandler)))))

	// Phase 8: The Monetization API (Billing)

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

type PageData struct {
	User        *User
	Error       string
	Message     string
	CurrentPage string
	Data        interface{}
	CSRFField   template.HTML
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data PageData) {
	data.CSRFField = csrf.TemplateField(r)
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getUser(r *http.Request) *User {
	user, _ := r.Context().Value("user").(*User)
	return user
}

func monitorHandler(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT l.id, l.task_id, l.user_id, l.execution_time, l.status, l.llm_response, l.error_message, t.name as task_name, u.email as user_email
		FROM task_logs l
		JOIN tasks t ON l.task_id = t.id
		JOIN users u ON l.user_id = u.id
		ORDER BY l.execution_time DESC
		LIMIT 100
	`)
	if err != nil {
		log.Printf("Error fetching logs: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []TaskLog
	for rows.Next() {
		var l TaskLog
		err := rows.Scan(&l.ID, &l.TaskID, &l.UserID, &l.ExecutionTime, &l.Status, &l.LLMResponse, &l.ErrorMessage, &l.TaskName, &l.UserEmail)
		if err != nil {
			log.Printf("Error scanning log: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	renderTemplate(w, r, "monitor", PageData{
		User:        user,
		CurrentPage: "monitor",
		Data: map[string]interface{}{
			"Logs": logs,
		},
	})
}

func adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := dbPool.Query(r.Context(), "SELECT id, email, role, tier, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.Tier, &u.CreatedAt)
		if err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		users = append(users, u)
	}

	renderTemplate(w, r, "admin_users", PageData{
		User:        user,
		CurrentPage: "admin_users",
		Data: map[string]interface{}{
			"Users": users,
		},
	})
}

func adminUpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.FormValue("user_id")
	role := r.FormValue("role")
	tier := r.FormValue("tier")

	if role != "" {
		_, err := dbPool.Exec(r.Context(), "UPDATE users SET role = $1 WHERE id = $2", role, userID)
		if err != nil {
			log.Printf("Error updating user role: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if tier != "" {
		_, err := dbPool.Exec(r.Context(), "UPDATE users SET tier = $1 WHERE id = $2", tier, userID)
		if err != nil {
			log.Printf("Error updating user tier: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
