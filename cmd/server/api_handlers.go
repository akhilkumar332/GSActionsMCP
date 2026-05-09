package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"schedule-mcp/db"
)

type AuthInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	CSRFToken string      `json:"csrfToken,omitempty"`
}

func apiCSRFHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		CSRFToken: csrf.Token(c.Request()),
	})
}

func apiSignupHandler(c echo.Context) error {
	var input AuthInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	user, err := RegisterUser(c.Request().Context(), input.Email, input.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, APIResponse{
		Success:   true,
		Data:      user,
		CSRFToken: csrf.Token(c.Request()),
	})
}

func apiLoginHandler(c echo.Context) error {
	var input AuthInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	sessionID, err := LoginUser(c.Request().Context(), input.Email, input.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid email or password"})
	}

	// Determine if we should use Secure cookies.
	useSecure := os.Getenv("ENV") == "production"
	if os.Getenv("LOCAL_DEV") == "true" {
		useSecure = false
	}

	c.SetCookie(&http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   useSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
	})

	// Parse session ID into pgtype.UUID
	var sessID pgtype.UUID
	if err := parseUUID(sessionID, &sessID); err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Internal Error"})
	}

	// Fetch user info to return
	u, err := queries.GetUserBySessionID(c.Request().Context(), db.GetUserBySessionIDParams{
		ID:        sessID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch user info"})
	}

	user := &User{
		ID:        u.ID,
		Email:     u.Email.String,
		APIKey:    u.ApiKey,
		Role:      u.Role.String,
		Tier:      u.Tier.String,
		CreatedAt: u.CreatedAt.Time,
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		Data:      user,
		CSRFToken: csrf.Token(c.Request()),
	})
}

func apiLogoutHandler(c echo.Context) error {
	cookie, err := c.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		var sessID pgtype.UUID
		if err := parseUUID(cookie.Value, &sessID); err == nil {
			_ = queries.DeleteWebSession(c.Request().Context(), sessID)
		}
	}

	// Determine if we should use Secure cookies.
	useSecure := os.Getenv("ENV") == "production"
	if os.Getenv("LOCAL_DEV") == "true" {
		useSecure = false
	}

	c.SetCookie(&http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   useSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Logged out successfully"})
}

func apiDashboardHandler(c echo.Context) error {
	user := getUserFromEcho(c)

	taskCount, err := queries.CountUserTasks(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch task count"})
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user":      user,
			"taskCount": taskCount,
		},
	})
}

func apiRotateAPIKeyHandler(c echo.Context) error {
	user := getUserFromEcho(c)

	newKey, err := RotateAPIKey(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to rotate API key"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"api_key": newKey}})
}

func apiMonitorHandler(c echo.Context) error {
	rows, err := queries.GetTaskLogs(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch logs"})
	}

	var logs []TaskLog
	for _, l := range rows {
		var llmResp, errMsg *string
		if l.LlmResponse.Valid {
			llmResp = &l.LlmResponse.String
		}
		if l.ErrorMessage.Valid {
			errMsg = &l.ErrorMessage.String
		}
		logs = append(logs, TaskLog{
			ID:            formatUUID(l.ID),
			TaskID:        formatUUID(l.TaskID),
			UserID:        l.UserID,
			ExecutionTime: l.ExecutionTime.Time,
			Status:        l.Status,
			LLMResponse:   llmResp,
			ErrorMessage:  errMsg,
			TaskName:      l.TaskName,
			UserEmail:     l.UserEmail.String,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: logs})
}

func apiAdminUsersHandler(c echo.Context) error {
	rows, err := queries.ListUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch users"})
	}

	var users []User
	for _, u := range rows {
		users = append(users, User{
			ID:        u.ID,
			Email:     u.Email.String,
			APIKey:    u.ApiKey,
			Role:      u.Role.String,
			Tier:      u.Tier.String,
			CreatedAt: u.CreatedAt.Time,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: users})
}

type AdminUpdateUserInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Tier   string `json:"tier"`
}

func apiAdminUpdateUserHandler(c echo.Context) error {
	var input AdminUpdateUserInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	if input.Role != "" {
		err := queries.UpdateUserRole(c.Request().Context(), db.UpdateUserRoleParams{
			Role: pgtype.Text{String: input.Role, Valid: true},
			ID:   input.UserID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update user role"})
		}
	}

	if input.Tier != "" {
		err := queries.UpdateUserTier(c.Request().Context(), db.UpdateUserTierParams{
			Tier: pgtype.Text{String: input.Tier, Valid: true},
			ID:   input.UserID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update user tier"})
		}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "User updated successfully"})
}

func apiApproveTaskHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}

	err := queries.UpdateTaskApprovalStatus(c.Request().Context(), db.UpdateTaskApprovalStatusParams{
		LastApprovalStatus: pgtype.Text{String: "approved", Valid: true},
		Status:             pgtype.Text{String: StatusActive, Valid: true},
		ID:                 id,
		UserID:             user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to approve task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task approved"})
}

func apiDenyTaskHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}

	err := queries.UpdateTaskApprovalStatus(c.Request().Context(), db.UpdateTaskApprovalStatusParams{
		LastApprovalStatus: pgtype.Text{String: "denied", Valid: true},
		Status:             pgtype.Text{String: StatusPaused, Valid: true},
		ID:                 id,
		UserID:             user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to deny task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task denied"})
}

func apiListSecretsHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	rows, err := queries.ListUserSecrets(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch secrets"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: rows})
}

func apiDeleteSecretHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	name := c.Param("name")

	err := queries.DeleteUserSecret(c.Request().Context(), db.DeleteUserSecretParams{
		UserID: user.ID,
		Name:   name,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete secret"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Secret deleted"})
}

type UpsertSecretInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func apiUpsertSecretHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	var input UpsertSecretInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	if input.Name == "" || input.Value == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Name and value are required"})
	}

	encrypted, err := Encrypt([]byte(input.Value))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Encryption error"})
	}

	_, err = queries.UpsertUserSecret(c.Request().Context(), db.UpsertUserSecretParams{
		UserID:         user.ID,
		Name:           input.Name,
		EncryptedValue: encrypted,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to store secret"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Secret stored successfully"})
}

func getUserFromEcho(c echo.Context) *User {
	user, _ := c.Get("user").(*User)
	return user
}
