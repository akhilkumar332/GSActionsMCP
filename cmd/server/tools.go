package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"schedule-mcp/db"
)

// registerTools sets up the MCP tools for managing schedules
func registerTools(s *server.MCPServer) {
	createTaskTool := mcp.NewTool("create_task",
		mcp.WithDescription("Creates a new scheduled task"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Task name")),
		mcp.WithString("trigger_type", mcp.Required(), mcp.Description("Trigger type (e.g. interval, cron)")),
		mcp.WithString("agent_prompt", mcp.Required(), mcp.Description("Agent prompt")),
		mcp.WithObject("trigger_config", mcp.Required(), mcp.Description("Trigger configuration")),
		mcp.WithString("missed_task_policy", mcp.Description("Policy for missed tasks (skip, run_immediate)")),
		mcp.WithString("depends_on_task_id", mcp.Description("Optional UUID of a task this task depends on")),
		mcp.WithObject("secrets", mcp.Description("Optional secrets to be stored securely (e.g. API keys)")),
		mcp.WithBoolean("requires_approval", mcp.Description("If true, the task will require manual approval before each execution")),
	)

	s.AddTool(createTaskTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		val := ctx.Value("user_tier")
		userTier, ok := val.(string)
		if !ok {
			userTier = TierFree
		}

		// Phase 2.2: Tool Quotas
		taskCount, err := queries.CountUserTasks(ctx, userID)
		if err != nil {
			log.Printf("Quota check error: %v", err)
			return mcp.NewToolResultError("Quota check failed. Please try again later."), nil
		}
		
		if userTier == TierFree && int(taskCount) >= QuotaFree {
			return mcp.NewToolResultError(fmt.Sprintf("quota exceeded: free tier allows maximum %d tasks", QuotaFree)), nil
		} else if userTier == TierPlus && int(taskCount) >= QuotaPlus {
			return mcp.NewToolResultError(fmt.Sprintf("quota exceeded: plus tier allows maximum %d tasks", QuotaPlus)), nil
		} else if userTier == TierPro && int(taskCount) >= QuotaPro {
			return mcp.NewToolResultError(fmt.Sprintf("quota exceeded: pro tier allows maximum %d tasks", QuotaPro)), nil
		}

		name, ok := args["name"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'name'"), nil
		}
		if len(name) > 100 {
			return mcp.NewToolResultError("name too long: maximum 100 characters"), nil
		}

		triggerType, ok := args["trigger_type"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'trigger_type'"), nil
		}
		agentPrompt, ok := args["agent_prompt"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'agent_prompt'"), nil
		}
		if len(agentPrompt) > 10000 {
			return mcp.NewToolResultError("agent_prompt too long: maximum 10,000 characters"), nil
		}
		triggerConfig, ok := args["trigger_config"].(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'trigger_config'"), nil
		}
		
		// Optional Phase 3 fields
		missedPolicy := PolicySkip
		if mp, ok := args["missed_task_policy"].(string); ok && (mp == PolicySkip || mp == PolicyRunImmediate) {
			missedPolicy = mp
		}

		requiresApproval := false
		if ra, ok := args["requires_approval"].(bool); ok {
			requiresApproval = ra
		}

		var encryptedSecrets []byte
		if secrets, ok := args["secrets"].(map[string]interface{}); ok && len(secrets) > 0 {
			secretsBytes, err := json.Marshal(secrets)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid secrets JSON: %v", err)), nil
			}
			encryptedSecrets, err = Encrypt(secretsBytes)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("encryption error: %v", err)), nil
			}
		}
		
		var dependsOn pgtype.UUID
		if dep, ok := args["depends_on_task_id"].(string); ok && dep != "" {
			if err := parseUUID(dep, &dependsOn); err != nil {
				return mcp.NewToolResultError("invalid depends_on_task_id format, expected UUID"), nil
			}
			// Check ownership
			exists, err := queries.CheckTaskOwnership(ctx, db.CheckTaskOwnershipParams{
				ID:     dependsOn,
				UserID: userID,
			})
			if err != nil || !exists {
				return mcp.NewToolResultError("invalid depends_on_task_id: task not found or unauthorized"), nil
			}
		}
		
		// trigger_config needs to be saved as JSONB
		triggerConfigBytes, err := json.Marshal(triggerConfig)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid trigger_config JSON: %v", err)), nil
		}

		// Calculate initial next_run
		nextRun, err := calculateNextRun(triggerType, triggerConfig, time.Now().UTC())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid trigger configuration: %v", err)), nil
		}

		task, err := queries.CreateTask(ctx, db.CreateTaskParams{
			UserID:           userID,
			Name:             name,
			TriggerType:      pgtype.Text{String: triggerType, Valid: true},
			TriggerConfig:    triggerConfigBytes,
			AgentPrompt:      agentPrompt,
			MissedTaskPolicy: pgtype.Text{String: missedPolicy, Valid: true},
			DependsOnTaskID:  dependsOn,
			NextRun:          pgtype.Timestamptz{Time: nextRun, Valid: true},
			RequiresApproval: pgtype.Bool{Bool: requiresApproval, Valid: true},
			EncryptedSecrets: encryptedSecrets,
		})

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to insert task: %v", err)), nil
		}

		resBytes, _ := json.Marshal(map[string]string{"status": "success", "task_id": formatUUID(task.ID), "next_run": nextRun.Format(time.RFC3339)})
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	listTasksTool := mcp.NewTool("list_tasks",
		mcp.WithDescription("Lists user's active tasks"),
	)
	s.AddTool(listTasksTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}

		rows, err := queries.ListUserTasks(ctx, userID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("db error: %v", err).Error()), nil
		}

		var tasks []map[string]interface{}
		var md strings.Builder
		md.WriteString("| ID | Prompt | Status | Next Run | Approval |\n")
		md.WriteString("|---|---|---|---|---|\n")

		for _, t := range rows {
			idStr := formatUUID(t.ID)
			approval := "Optional"
			if t.RequiresApproval.Bool {
				approval = "Required"
			}
			nextRunStr := t.NextRun.Time.Format("2006-01-02 15:04")

			cleanName := strings.ReplaceAll(t.Name, "|", "\\|")
			md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				idStr, cleanName, t.Status.String, nextRunStr, approval))

			tasks = append(tasks, map[string]interface{}{
				"id":                   idStr,
				"name":                 t.Name,
				"trigger_type":         t.TriggerType.String,
				"status":               t.Status.String,
				"next_run":             t.NextRun.Time.Format(time.RFC3339),
				"requires_approval":    t.RequiresApproval.Bool,
				"has_secrets":          len(t.EncryptedSecrets) > 0,
				"last_approval_status": t.LastApprovalStatus.String,
			})
		}

		resBytes, _ := json.Marshal(tasks)
		if string(resBytes) == "null" {
			resBytes = []byte("[]")
		}

		finalOutput := fmt.Sprintf("%s\n<!-- JSON: %s -->", md.String(), string(resBytes))
		return mcp.NewToolResultText(finalOutput), nil
	})

	pauseTaskTool := mcp.NewTool("pause_task",
		mcp.WithDescription("Pauses a scheduled task"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
	)
	s.AddTool(pauseTaskTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		id, ok := args["id"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'id'"), nil
		}
		
		var tid pgtype.UUID
		if err := parseUUID(id, &tid); err != nil {
			return mcp.NewToolResultError("invalid task ID format"), nil
		}

		err := queries.UpdateTaskStatusByUserID(ctx, db.UpdateTaskStatusByUserIDParams{
			Status: pgtype.Text{String: StatusPaused, Valid: true},
			ID:     tid,
			UserID: userID,
		})
		// Note: The original code didn't check ownership here, but it's good practice.
		// For now I'll stick to original logic but using sqlc.
		// Actually, I'll add ownership check to queries.sql for safety.
		// But wait, the original UPDATE had `WHERE id = $2 AND user_id = $3`.
		// I should update my query in queries.sql.

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		// Emit Redis event
		evtPayload, _ := json.Marshal(map[string]interface{}{
			"task_id": id,
			"status":  StatusPaused,
		})
		_ = PublishEvent(ctx, PubSubEvent{
			UserID:    userID,
			EventType: "task_status_changed",
			Payload:   string(evtPayload),
		})

		resBytes, _ := json.Marshal(map[string]string{"status": StatusPaused})
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	resumeTaskTool := mcp.NewTool("resume_task",
		mcp.WithDescription("Resumes a scheduled task"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
	)
	s.AddTool(resumeTaskTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		id, ok := args["id"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'id'"), nil
		}
		
		var tid pgtype.UUID
		if err := parseUUID(id, &tid); err != nil {
			return mcp.NewToolResultError("invalid task ID format"), nil
		}

		err := queries.ResetTaskFailureCount(ctx, db.ResetTaskFailureCountParams{
			Status: pgtype.Text{String: StatusActive, Valid: true},
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		// Emit Redis event
		payload, _ := json.Marshal(map[string]string{"task_id": id, "status": StatusActive})
		_ = PublishEvent(ctx, PubSubEvent{
			UserID:    userID,
			EventType: "task_status_changed",
			Payload:   string(payload),
		})

		resBytes, _ := json.Marshal(map[string]string{"status": StatusActive})
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	deleteTaskTool := mcp.NewTool("delete_task",
		mcp.WithDescription("Deletes a scheduled task"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
	)
	s.AddTool(deleteTaskTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		id, ok := args["id"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'id'"), nil
		}
		
		var tid pgtype.UUID
		if err := parseUUID(id, &tid); err != nil {
			return mcp.NewToolResultError("invalid task ID format"), nil
		}

		err := queries.DeleteTask(ctx, db.DeleteTaskParams{
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}
		resBytes, _ := json.Marshal(map[string]string{"status": "deleted"})
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	storeSecretTool := mcp.NewTool("store_secret",
		mcp.WithDescription("Stores an encrypted secret for the user"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Secret name")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Secret value")),
	)
	s.AddTool(storeSecretTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		name, ok := args["name"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'name'"), nil
		}
		value, ok := args["value"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'value'"), nil
		}

		encrypted, err := Encrypt([]byte(value))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encryption error: %v", err)), nil
		}

		_, err = queries.UpsertUserSecret(ctx, db.UpsertUserSecretParams{
			UserID:         userID,
			Name:           name,
			EncryptedValue: encrypted,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		return mcp.NewToolResultText("Secret stored successfully"), nil
	})

	listSecretsTool := mcp.NewTool("list_secrets",
		mcp.WithDescription("Lists user's secret names and creation dates"),
	)
	s.AddTool(listSecretsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}

		rows, err := queries.ListUserSecrets(ctx, userID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		var md strings.Builder
		md.WriteString("| Name | Created At |\n")
		md.WriteString("|---|---|\n")

		for _, r := range rows {
			createdAt := r.CreatedAt.Time.Format("2006-01-02 15:04")
			md.WriteString(fmt.Sprintf("| %s | %s |\n", r.Name, createdAt))
		}

		if len(rows) == 0 {
			return mcp.NewToolResultText("No secrets found."), nil
		}

		return mcp.NewToolResultText(md.String()), nil
	})

	deleteSecretTool := mcp.NewTool("delete_secret",
		mcp.WithDescription("Deletes a user secret"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Secret name")),
	)
	s.AddTool(deleteSecretTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		name, ok := args["name"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'name'"), nil
		}

		err := queries.DeleteUserSecret(ctx, db.DeleteUserSecretParams{
			UserID: userID,
			Name:   name,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		return mcp.NewToolResultText("Secret deleted successfully"), nil
	})
}
