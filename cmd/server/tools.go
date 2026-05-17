package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"aktionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTools sets up the MCP tools for managing schedules
func registerTools(s *server.MCPServer) {
	createTaskTool := mcp.NewTool("create_task",
		mcp.WithDescription("Creates a new scheduled task"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Task name")),
		mcp.WithString("trigger_type", mcp.Required(), mcp.Description("Trigger type (e.g. interval, cron, date)")),
		mcp.WithString("agent_prompt", mcp.Description("Agent prompt")),
		mcp.WithObject("trigger_config", mcp.Required(), mcp.Description("Trigger configuration")),
		mcp.WithString("missed_task_policy", mcp.Description("Policy for missed tasks (skip, run_immediately)")),
		mcp.WithString("depends_on_task_id", mcp.Description("Optional UUID of a task this task depends on")),
		mcp.WithObject("secrets", mcp.Description("Optional secrets to be stored securely (e.g. API keys)")),
		mcp.WithBoolean("requires_approval", mcp.Description("If true, the task will require manual approval before each execution")),
		mcp.WithObject("branch_condition", mcp.Description("Optional branch condition for dependent tasks (e.g. {\"if\": \"contains\", \"value\": \"success\"})")),
		mcp.WithBoolean("is_bundle_root", mcp.Description("If true, this task is the root of a workflow bundle")),
		mcp.WithString("task_type", mcp.Description("Optional task type (e.g. decision_router, native_action)")),
		mcp.WithString("native_code", mcp.Description("Optional JS code for native_action tasks")),
		mcp.WithObject("swarm_config", mcp.Description("Optional swarm configuration for swarm_router tasks")),
	)

	s.AddTool(createTaskTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}
		userID, ok := ctx.Value(userIDKey).(string)
		if !ok {
			return mcp.NewToolResultError("unauthorized"), nil
		}
		val := ctx.Value(userTierKey)
		userTier, ok := val.(string)
		if !ok {
			userTier = TierFree
		}

		// Central Quota Enforcement
		if err := CheckUserQuota(ctx, userID, userTier); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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
		agentPrompt := ""
		if ap, ok := args["agent_prompt"].(string); ok {
			agentPrompt = ap
		}
		if len(agentPrompt) > 10000 {
			return mcp.NewToolResultError("agent_prompt too long: maximum 10,000 characters"), nil
		}
		triggerConfig, ok := args["trigger_config"].(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'trigger_config'"), nil
		}

		taskType := "mcp_sampling"
		if tt, ok := args["task_type"].(string); ok && tt != "" {
			taskType = tt
		}
		nativeCode := ""
		if nc, ok := args["native_code"].(string); ok {
			nativeCode = nc
		}

		// Optional Phase 3 fields
		missedPolicy := PolicySkip
		if mp, ok := args["missed_task_policy"].(string); ok {
			switch mp {
			case PolicySkip, PolicyRunImmediate:
				missedPolicy = mp
			case "run_immediate":
				// Backward-compatible alias for older clients/docs.
				missedPolicy = PolicyRunImmediate
			}
		}

		requiresApproval := false
		if ra, ok := args["requires_approval"].(bool); ok {
			requiresApproval = ra
		}

		var branchCondition []byte
		if bc, ok := args["branch_condition"].(map[string]interface{}); ok {
			var err error
			branchCondition, err = json.Marshal(bc)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid branch_condition JSON: %v", err)), nil
			}
		}

		isBundleRoot := false
		if ibr, ok := args["is_bundle_root"].(bool); ok {
			isBundleRoot = ibr
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

		var swarmConfig []byte
		if sc, ok := args["swarm_config"].(map[string]interface{}); ok {
			var err error
			swarmConfig, err = json.Marshal(sc)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid swarm_config JSON: %v", err)), nil
			}
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
			BranchCondition:  branchCondition,
			IsBundleRoot:     pgtype.Bool{Bool: isBundleRoot, Valid: true},
			TaskType:         pgtype.Text{String: taskType, Valid: true},
			NativeCode:       pgtype.Text{String: nativeCode, Valid: true},
			SwarmConfig:      swarmConfig,
		})

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to insert task: %v", err)), nil
		}
		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "task.create",
			ResourceType: "task",
			ResourceID:   formatUUID(task.ID),
			Metadata: map[string]interface{}{
				"trigger_type": triggerType,
			},
		})

		resMap := map[string]string{"status": "success", "task_id": formatUUID(task.ID), "next_run": nextRun.Format(time.RFC3339)}
		resBytes, err := json.Marshal(resMap)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("internal error marshaling response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	listTasksTool := mcp.NewTool("list_tasks",
		mcp.WithDescription("Lists user's active tasks"),
	)
	s.AddTool(listTasksTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value(userIDKey).(string)
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
		userID, ok := ctx.Value(userIDKey).(string)
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
		exists, err := queries.CheckTaskOwnership(ctx, db.CheckTaskOwnershipParams{
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}
		if !exists {
			return mcp.NewToolResultError("task not found"), nil
		}

		err = queries.UpdateTaskStatusByUserID(ctx, db.UpdateTaskStatusByUserIDParams{
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
		evtPayload, err := json.Marshal(map[string]interface{}{
			"task_id": id,
			"status":  StatusPaused,
		})
		if err == nil {
			if err := PublishEvent(ctx, PubSubEvent{
				UserID:    userID,
				EventType: "task_status_changed",
				Payload:   string(evtPayload),
			}); err != nil {
				log.Printf("Error publishing task_status_changed event: %v", err)
			}
		}

		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "task.pause",
			ResourceType: "task",
			ResourceID:   id,
		})

		resBytes, err := json.Marshal(map[string]string{"status": StatusPaused})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("internal error: %v", err)), nil
		}
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
		userID, ok := ctx.Value(userIDKey).(string)
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
		exists, err := queries.CheckTaskOwnership(ctx, db.CheckTaskOwnershipParams{
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}
		if !exists {
			return mcp.NewToolResultError("task not found"), nil
		}

		err = queries.ResetTaskFailureCount(ctx, db.ResetTaskFailureCountParams{
			Status: pgtype.Text{String: StatusActive, Valid: true},
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}

		// Emit Redis event
		evtPayload, err := json.Marshal(map[string]string{"task_id": id, "status": StatusActive})
		if err == nil {
			if err := PublishEvent(ctx, PubSubEvent{
				UserID:    userID,
				EventType: "task_status_changed",
				Payload:   string(evtPayload),
			}); err != nil {
				log.Printf("Error publishing task_status_changed event: %v", err)
			}
		}

		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "task.resume",
			ResourceType: "task",
			ResourceID:   id,
		})

		resBytes, err := json.Marshal(map[string]string{"status": StatusActive})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("internal error: %v", err)), nil
		}
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
		userID, ok := ctx.Value(userIDKey).(string)
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
		exists, err := queries.CheckTaskOwnership(ctx, db.CheckTaskOwnershipParams{
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}
		if !exists {
			return mcp.NewToolResultError("task not found"), nil
		}

		err = queries.DeleteTask(ctx, db.DeleteTaskParams{
			ID:     tid,
			UserID: userID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("db error: %v", err)), nil
		}
		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "task.delete",
			ResourceType: "task",
			ResourceID:   id,
		})
		resBytes, err := json.Marshal(map[string]string{"status": "deleted"})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("internal error: %v", err)), nil
		}
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
		userID, ok := ctx.Value(userIDKey).(string)
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
		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "secret.upsert",
			ResourceType: "secret",
			ResourceID:   name,
		})

		return mcp.NewToolResultText("Secret stored successfully"), nil
	})

	listSecretsTool := mcp.NewTool("list_secrets",
		mcp.WithDescription("Lists user's secret names and creation dates"),
	)
	s.AddTool(listSecretsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value(userIDKey).(string)
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
		userID, ok := ctx.Value(userIDKey).(string)
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
		writeAuditLog(ctx, AuditEvent{
			UserID:       userID,
			Action:       "secret.delete",
			ResourceType: "secret",
			ResourceID:   name,
		})

		return mcp.NewToolResultText("Secret deleted successfully"), nil
	})
}
