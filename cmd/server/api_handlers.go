package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AuthInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func apiSignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method Not Allowed"})
		return
	}

	var input AuthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
		return
	}

	user, err := RegisterUser(r.Context(), input.Email, input.Password)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}

	sendJSON(w, http.StatusCreated, APIResponse{Success: true, Data: user})
}

func apiLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method Not Allowed"})
		return
	}

	var input AuthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
		return
	}

	sessionID, err := LoginUser(r.Context(), input.Email, input.Password)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid email or password"})
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

	// Fetch user info to return
	var user User
	err = dbPool.QueryRow(r.Context(),
		"SELECT id, email, api_key, role, tier, created_at FROM users WHERE id = (SELECT user_id FROM web_sessions WHERE id = $1)",
		sessionID,
	).Scan(&user.ID, &user.Email, &user.APIKey, &user.Role, &user.Tier, &user.CreatedAt)

	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch user info"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: user})
}

func apiLogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		_, err = dbPool.Exec(r.Context(), "DELETE FROM web_sessions WHERE id = $1", cookie.Value)
		if err != nil {
			// log.Printf("Error deleting session: %v", err)
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

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Logged out successfully"})
}

func apiDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
		return
	}

	var taskCount int
	err := dbPool.QueryRow(r.Context(), "SELECT COUNT(*) FROM tasks WHERE user_id = $1", user.ID).Scan(&taskCount)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch task count"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user":      user,
			"taskCount": taskCount,
		},
	})
}

func apiRotateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method Not Allowed"})
		return
	}
	user := getUser(r)
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
		return
	}

	newKey, err := RotateAPIKey(r.Context(), user.ID)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to rotate API key"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"api_key": newKey}})
}

func apiMonitorHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(r.Context(), `
		SELECT l.id, l.task_id, l.user_id, l.execution_time, l.status, l.llm_response, l.error_message, t.name as task_name, u.email as user_email
		FROM task_logs l
		JOIN tasks t ON l.task_id = t.id
		JOIN users u ON l.user_id = u.id
		ORDER BY l.execution_time DESC
		LIMIT 100
	`)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch logs"})
		return
	}
	defer rows.Close()

	var logs []TaskLog
	for rows.Next() {
		var l TaskLog
		err := rows.Scan(&l.ID, &l.TaskID, &l.UserID, &l.ExecutionTime, &l.Status, &l.LLMResponse, &l.ErrorMessage, &l.TaskName, &l.UserEmail)
		if err != nil {
			continue
		}
		logs = append(logs, l)
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: logs})
}

func apiAdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(r.Context(), "SELECT id, email, role, tier, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.Tier, &u.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, u)
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: users})
}

type AdminUpdateUserInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Tier   string `json:"tier"`
}

func apiAdminUpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method Not Allowed"})
		return
	}

	var input AdminUpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
		return
	}

	if input.Role != "" {
		_, err := dbPool.Exec(r.Context(), "UPDATE users SET role = $1 WHERE id = $2", input.Role, input.UserID)
		if err != nil {
			sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update user role"})
			return
		}
	}

	if input.Tier != "" {
		_, err := dbPool.Exec(r.Context(), "UPDATE users SET tier = $1 WHERE id = $2", input.Tier, input.UserID)
		if err != nil {
			sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update user tier"})
			return
		}
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Message: "User updated successfully"})
}
