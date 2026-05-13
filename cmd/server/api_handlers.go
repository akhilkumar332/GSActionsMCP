package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Email and password are required"})
	}

	user, err := RegisterUser(c.Request().Context(), input.Email, input.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "auth.signup",
		ResourceType: "user",
		ResourceID:   user.ID,
		Metadata: map[string]interface{}{
			"email": user.Email,
		},
	})

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
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Email and password are required"})
	}

	sessionID, err := LoginUser(c.Request().Context(), input.Email, input.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid email or password"})
	}

	// Determine if we should use Secure cookies.
	useSecure := appConfig.secureCookies()

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
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "auth.login",
		ResourceType: "session",
		ResourceID:   sessionID,
	})

	return c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		Data:      user,
		CSRFToken: csrf.Token(c.Request()),
	})
}

func apiLogoutHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	cookie, err := c.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		var sessID pgtype.UUID
		if err := parseUUID(cookie.Value, &sessID); err == nil {
			_ = queries.DeleteWebSession(c.Request().Context(), sessID)
		}
	}

	// Determine if we should use Secure cookies.
	useSecure := appConfig.secureCookies()

	c.SetCookie(&http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   useSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	if user != nil && err == nil && cookie.Value != "" {
		writeAuditLog(c.Request().Context(), AuditEvent{
			UserID:       user.ID,
			Action:       "auth.logout",
			ResourceType: "session",
			ResourceID:   cookie.Value,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Logged out successfully"})
}

func apiDashboardHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

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
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	newKey, err := RotateAPIKey(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to rotate API key"})
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "user.rotate_api_key",
		ResourceType: "user",
		ResourceID:   user.ID,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"api_key": newKey}})
}

func apiExportTasksHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	tasks, err := exportUserTasks(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to export tasks"})
	}

	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "task.export",
		ResourceType: "task_bundle",
		Metadata: map[string]interface{}{
			"count": len(tasks),
		},
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"tasks":           tasks,
		"exported_at":     time.Now().UTC().Format(time.RFC3339),
		"includesSecrets": false,
	}})
}

func apiImportTasksHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	var input ImportTasksRequest
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}
	if len(input.Tasks) == 0 {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "At least one task is required"})
	}

	mapping, err := importUserTasks(c.Request().Context(), user.ID, input.Tasks)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
	}

	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "task.import",
		ResourceType: "task_bundle",
		Metadata: map[string]interface{}{
			"count": len(input.Tasks),
		},
	})

	return c.JSON(http.StatusCreated, APIResponse{Success: true, Data: map[string]interface{}{
		"imported_count": len(input.Tasks),
		"legacy_to_new":  mapping,
	}})
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

	if input.UserID == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "user_id is required"})
	}

	// Verify user exists
	_, err := queries.GetUser(c.Request().Context(), input.UserID)
	if err != nil {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "User not found"})
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
	adminUser := getUserFromEcho(c)
	adminID := ""
	if adminUser != nil {
		adminID = adminUser.ID
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       adminID,
		Action:       "admin.update_user",
		ResourceType: "user",
		ResourceID:   input.UserID,
		Metadata: map[string]interface{}{
			"role": input.Role,
			"tier": input.Tier,
		},
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "User updated successfully"})
}

func apiApproveTaskHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}
	exists, err := queries.CheckTaskOwnership(c.Request().Context(), db.CheckTaskOwnershipParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to verify task ownership"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	err = queries.UpdateTaskApprovalStatus(c.Request().Context(), db.UpdateTaskApprovalStatusParams{
		LastApprovalStatus: pgtype.Text{String: "approved", Valid: true},
		Status:             pgtype.Text{String: StatusActive, Valid: true},
		ID:                 id,
		UserID:             user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to approve task"})
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "task.approve",
		ResourceType: "task",
		ResourceID:   idStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task approved"})
}

func apiDenyTaskHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}
	exists, err := queries.CheckTaskOwnership(c.Request().Context(), db.CheckTaskOwnershipParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to verify task ownership"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	err = queries.UpdateTaskApprovalStatus(c.Request().Context(), db.UpdateTaskApprovalStatusParams{
		LastApprovalStatus: pgtype.Text{String: "denied", Valid: true},
		Status:             pgtype.Text{String: StatusPaused, Valid: true},
		ID:                 id,
		UserID:             user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to deny task"})
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "task.deny",
		ResourceType: "task",
		ResourceID:   idStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task denied"})
}

func apiListSecretsHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	rows, err := queries.ListUserSecrets(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch secrets"})
	}
	if rows == nil {
		rows = []db.ListUserSecretsRow{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: rows})
}

func apiDeleteSecretHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	name := c.Param("name")

	err := queries.DeleteUserSecret(c.Request().Context(), db.DeleteUserSecretParams{
		UserID: user.ID,
		Name:   name,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete secret"})
	}
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "secret.delete",
		ResourceType: "secret",
		ResourceID:   name,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Secret deleted"})
}

type UpsertSecretInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func apiListWebhooksHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	rows, err := queries.ListOutboundWebhooks(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch webhooks"})
	}

	var hooks []WebhookSubscription
	for _, row := range rows {
		var eventTypes []string
		_ = json.Unmarshal(row.EventTypes, &eventTypes)

		hooks = append(hooks, WebhookSubscription{
			ID:          formatUUID(row.ID),
			EndpointURL: row.EndpointUrl,
			EventTypes:  eventTypes,
			IsActive:    row.IsActive,
			CreatedAt:   row.CreatedAt.Time,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: hooks})
}

func apiCreateWebhookHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	var input WebhookCreateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}
	if input.EndpointURL == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "endpoint_url is required"})
	}
	if len(input.EventTypes) == 0 {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "event_types are required"})
	}
	secret := input.SigningSecret
	if secret == "" {
		generated, err := generateSigningSecret()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to generate signing secret"})
		}
		secret = generated
	}
	encryptedSecret, err := Encrypt([]byte(secret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to encrypt signing secret"})
	}
	eventTypesJSON, _ := json.Marshal(input.EventTypes)

	row, err := queries.CreateOutboundWebhook(c.Request().Context(), db.CreateOutboundWebhookParams{
		UserID:                 user.ID,
		EndpointUrl:            input.EndpointURL,
		EventTypes:             eventTypesJSON,
		EncryptedSigningSecret: encryptedSecret,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create webhook"})
	}

	id := formatUUID(row.ID)
	createdAt := row.CreatedAt.Time

	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "webhook.create",
		ResourceType: "webhook",
		ResourceID:   id,
		Metadata: map[string]interface{}{
			"endpoint_url": input.EndpointURL,
			"event_types":  input.EventTypes,
		},
	})

	return c.JSON(http.StatusCreated, APIResponse{Success: true, Data: map[string]interface{}{
		"id":             id,
		"endpoint_url":   input.EndpointURL,
		"event_types":    input.EventTypes,
		"is_active":      true,
		"created_at":     createdAt.UTC().Format(time.RFC3339),
		"signing_secret": secret,
	}})
}

func apiDeleteWebhookHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid webhook ID"})
	}

	err := queries.DeleteOutboundWebhook(c.Request().Context(), db.DeleteOutboundWebhookParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete webhook"})
	}

	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "webhook.delete",
		ResourceType: "webhook",
		ResourceID:   idStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Webhook deleted"})
}

func apiWebhookDeliveriesHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := parseUUID(idStr, &id); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid webhook ID"})
	}

	rows, err := queries.ListWebhookDeliveries(c.Request().Context(), db.ListWebhookDeliveriesParams{
		WebhookID: id,
		UserID:    user.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch webhook deliveries"})
	}

	var deliveries []WebhookDelivery
	for _, row := range rows {
		var statusCode *int32
		if row.StatusCode.Valid {
			sc := row.StatusCode.Int32
			statusCode = &sc
		}
		var body *string
		if row.ResponseBody.Valid {
			b := row.ResponseBody.String
			body = &b
		}

		deliveries = append(deliveries, WebhookDelivery{
			ID:           formatUUID(row.ID),
			WebhookID:    formatUUID(row.WebhookID),
			EventType:    row.EventType,
			StatusCode:   statusCode,
			Success:      row.Success,
			ResponseBody: body,
			CreatedAt:    row.CreatedAt.Time,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: deliveries})
}

func apiUpsertSecretHandler(c echo.Context) error {
	user := getUserFromEcho(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	var input UpsertSecretInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	if input.Name == "" || input.Value == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Name and value are required"})
	}

	// Basic validation: names should be alphanumeric or underscores
	nameRegex := `^[a-zA-Z0-9_-]+$`
	matched, _ := regexp.MatchString(nameRegex, input.Name)
	if !matched {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid secret name. Use alphanumeric characters, dashes, or underscores."})
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
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       user.ID,
		Action:       "secret.upsert",
		ResourceType: "secret",
		ResourceID:   input.Name,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Secret stored successfully"})
}

type SEOUpdateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	OGImage     string `json:"og_image"`
}

func apiGetSEOHandler(c echo.Context) error {
	settings, err := queries.GetSEOSettings(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch SEO settings"})
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    settings,
	})
}

func apiUpdateSEOHandler(c echo.Context) error {
	var input SEOUpdateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	err := queries.UpdateSEOSettings(c.Request().Context(), db.UpdateSEOSettingsParams{
		Title:       input.Title,
		Description: input.Description,
		Keywords:    input.Keywords,
		OgImage:     pgtype.Text{String: input.OGImage, Valid: input.OGImage != ""},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update SEO settings"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "SEO settings updated successfully"})
}

func apiAdminAuditLogsHandler(c echo.Context) error {
	limit := 100
	if raw := c.QueryParam("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 500 {
			return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "limit must be between 1 and 500"})
		}
		limit = parsed
	}

	rows, err := queries.ListAuditLogs(c.Request().Context(), int32(limit))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch audit logs"})
	}

	var logs []AuditLogEntry
	for _, row := range rows {
		var entry AuditLogEntry
		entry.ID = formatUUID(row.ID)
		if row.UserID.Valid {
			uid := row.UserID.String
			entry.UserID = &uid
		}
		entry.Action = row.Action
		entry.ResourceType = row.ResourceType
		if row.ResourceID.Valid {
			rid := row.ResourceID.String
			entry.ResourceID = &rid
		}
		entry.CreatedAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		entry.Metadata = map[string]interface{}{}
		_ = json.Unmarshal(row.Metadata, &entry.Metadata)
		logs = append(logs, entry)
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: logs})
}

func apiAdminUsageHandler(c echo.Context) error {
	metrics, err := queries.GetSystemUsageMetrics(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch metrics"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]int64{
		"users":            metrics.UserCount,
		"tasks":            metrics.TaskCount,
		"task_successes":   metrics.SuccessCount,
		"task_failures":    metrics.FailureCount,
		"task_missed":      metrics.MissedCount,
		"audit_log_events": metrics.AuditCount,
	}})
}
