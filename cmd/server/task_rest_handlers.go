package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"schedule-mcp/db"
)

type CreateTaskRequest struct {
	Name                string          `json:"name"`
	WorkspaceID         string          `json:"workspace_id"`
	TaskType            string          `json:"task_type"`
	AgentPrompt         string          `json:"agent_prompt"`
	NativeCode          string          `json:"native_code"`
	TriggerType         string          `json:"trigger_type"`
	TriggerConfig       json.RawMessage `json:"trigger_config"`
	RequiresApproval    bool            `json:"requires_approval"`
	MissedTaskPolicy    string          `json:"missed_task_policy"`
	DependsOnTaskID     string          `json:"depends_on_task_id"`
	TriggerOnCompletion bool            `json:"trigger_on_completion"`
	BranchCondition     json.RawMessage `json:"branch_condition"`
	LoopCondition       json.RawMessage `json:"loop_condition"`
	IsBundleRoot        bool            `json:"is_bundle_root"`
}

func apiCreateTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	var workspaceID pgtype.UUID
	if req.WorkspaceID != "" {
		var err error
		workspaceID, err = mustParseUUID(c, req.WorkspaceID)
		if err != nil {
			return err
		}
	}

	var dependsOnTaskID pgtype.UUID
	if req.DependsOnTaskID != "" {
		var err error
		dependsOnTaskID, err = mustParseUUID(c, req.DependsOnTaskID)
		if err != nil {
			return err
		}
	}

	// Default missed task policy if not provided
	policy := req.MissedTaskPolicy
	if policy == "" {
		policy = "run_immediately"
	}

	params := db.CreateTaskParams{
		UserID:              userID,
		Name:                req.Name,
		TriggerType:         pgtype.Text{String: req.TriggerType, Valid: true},
		TriggerConfig:       req.TriggerConfig,
		AgentPrompt:         req.AgentPrompt,
		WorkspaceID:         workspaceID,
		TaskType:            pgtype.Text{String: req.TaskType, Valid: true},
		NativeCode:          pgtype.Text{String: req.NativeCode, Valid: true},
		MissedTaskPolicy:    pgtype.Text{String: policy, Valid: true},
		RequiresApproval:    pgtype.Bool{Bool: req.RequiresApproval, Valid: true},
		DependsOnTaskID:     dependsOnTaskID,
		TriggerOnCompletion: pgtype.Bool{Bool: req.TriggerOnCompletion, Valid: true},
		BranchCondition:     req.BranchCondition,
		LoopCondition:       req.LoopCondition,
		IsBundleRoot:        pgtype.Bool{Bool: req.IsBundleRoot, Valid: true},
		NextRun:             pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	task, err := queries.CreateTask(c.Request().Context(), params)
	if err != nil {
		log.Printf("Failed to create task: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create task"})
	}

	// Audit Log
	taskIDStr := formatUUID(task.ID)
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.create",
		ResourceType: "task",
		ResourceID:   taskIDStr,
		Metadata: map[string]interface{}{
			"name": req.Name,
		},
	})

	return c.JSON(http.StatusCreated, APIResponse{Success: true, Data: task})
}

func apiListTasksHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	tasks, err := queries.ListUserTasks(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to list tasks"})
	}
	if tasks == nil {
		tasks = []db.ListUserTasksRow{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: tasks})
}

func apiPauseTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	err = queries.UpdateTaskStatusByUserID(c.Request().Context(), db.UpdateTaskStatusByUserIDParams{
		Status: pgtype.Text{String: "paused", Valid: true},
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to pause task"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.pause",
		ResourceType: "task",
		ResourceID:   taskIDStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task paused"})
}

func apiResumeTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	err = queries.UpdateTaskStatusByUserID(c.Request().Context(), db.UpdateTaskStatusByUserIDParams{
		Status: pgtype.Text{String: "active", Valid: true},
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to resume task"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.resume",
		ResourceType: "task",
		ResourceID:   taskIDStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task resumed"})
}

func apiDeleteTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	err = queries.DeleteTask(c.Request().Context(), db.DeleteTaskParams{
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete task"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.delete",
		ResourceType: "task",
		ResourceID:   taskIDStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task deleted"})
}

type UpdateTaskRequest struct {
	AgentPrompt         string          `json:"agent_prompt"`
	MissedTaskPolicy    string          `json:"missed_task_policy"`
	UICoordinates       json.RawMessage `json:"ui_coordinates"`
	DependsOnTaskID     string          `json:"depends_on_task_id"`
	TriggerOnCompletion bool            `json:"trigger_on_completion"`
	BranchCondition     json.RawMessage `json:"branch_condition"`
	LoopCondition       json.RawMessage `json:"loop_condition"`
}

func apiUpdateTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	var req UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	// Auto-snapshot before update
	if _, err := queries.CreateTaskVersion(c.Request().Context(), db.CreateTaskVersionParams{
		ID:     taskID,
		UserID: userID,
	}); err != nil {
		log.Printf("Warning: Failed to create task version snapshot for %s: %v", taskIDStr, err)
	}

	var dependsOnTaskID pgtype.UUID
	if req.DependsOnTaskID != "" {
		var err error
		dependsOnTaskID, err = mustParseUUID(c, req.DependsOnTaskID)
		if err != nil {
			return err
		}
	}

	_, err = queries.UpdateTaskAgentPromptAndPolicy(c.Request().Context(), db.UpdateTaskAgentPromptAndPolicyParams{
		AgentPrompt:         req.AgentPrompt,
		MissedTaskPolicy:    pgtype.Text{String: req.MissedTaskPolicy, Valid: true},
		UiCoordinates:       req.UICoordinates,
		DependsOnTaskID:     dependsOnTaskID,
		TriggerOnCompletion: pgtype.Bool{Bool: req.TriggerOnCompletion, Valid: true},
		BranchCondition:     req.BranchCondition,
		LoopCondition:       req.LoopCondition,
		ID:                  taskID,
		UserID:              userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
		}
		log.Printf("Failed to update task: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update task"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.update",
		ResourceType: "task",
		ResourceID:   taskIDStr,
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task updated"})
}

func apiListTaskVersionsHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	// Check ownership first
	exists, _ := queries.CheckTaskOwnership(c.Request().Context(), db.CheckTaskOwnershipParams{ID: taskID, UserID: userID})
	if !exists {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	versions, err := queries.ListTaskVersions(c.Request().Context(), taskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to list versions"})
	}
	if versions == nil {
		versions = []db.TaskVersion{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: versions})
}

func apiRestoreTaskVersionHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	versionIDStr := c.Param("version_id")

	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}
	versionID, err := mustParseUUID(c, versionIDStr)
	if err != nil {
		return err
	}

	// 1. Create a snapshot of CURRENT state before rolling back
	if _, err := queries.CreateTaskVersion(c.Request().Context(), db.CreateTaskVersionParams{
		ID:     taskID,
		UserID: userID,
	}); err != nil {
		log.Printf("Warning: Failed to create current state snapshot before restore for %s: %v", taskIDStr, err)
	}

	// 2. Restore
	err = queries.RestoreTaskFromVersion(c.Request().Context(), db.RestoreTaskFromVersionParams{
		ID:     taskID,
		UserID: userID,
		ID_2:   versionID, // ID_2 is the version ID in RestoreTaskFromVersionParams
	})
	if err != nil {
		log.Printf("Restore failed: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Restore failed"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.restore_version",
		ResourceType: "task",
		ResourceID:   taskIDStr,
		Metadata: map[string]interface{}{
			"version_id": versionIDStr,
		},
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task restored successfully"})
}

type LinkTaskRequest struct {
	DependsOnTaskID     string `json:"depends_on_task_id"`
	TriggerOnCompletion bool   `json:"trigger_on_completion"`
}

func apiLinkTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	var req LinkTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	var dependsOnTaskID pgtype.UUID
	if req.DependsOnTaskID != "" {
		var err error
		dependsOnTaskID, err = mustParseUUID(c, req.DependsOnTaskID)
		if err != nil {
			return err
		}
	}

	err = queries.LinkTaskDependency(c.Request().Context(), db.LinkTaskDependencyParams{
		DependsOnTaskID:     dependsOnTaskID,
		TriggerOnCompletion: pgtype.Bool{Bool: req.TriggerOnCompletion, Valid: true},
		ID:                  taskID,
		UserID:              userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to link tasks"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.link",
		ResourceType: "task",
		ResourceID:   taskIDStr,
		Metadata: map[string]interface{}{
			"depends_on": req.DependsOnTaskID,
		},
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Tasks linked successfully"})
}

func apiGetExecutionTracesHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	executionID := c.Param("execution_id")

	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	// Check ownership
	exists, err := queries.CheckTaskOwnership(c.Request().Context(), db.CheckTaskOwnershipParams{
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to verify task ownership"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	traces, err := queries.ListExecutionTracesByExecutionID(c.Request().Context(), db.ListExecutionTracesByExecutionIDParams{
		TaskID:      taskID,
		ExecutionID: executionID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch execution traces"})
	}
	if traces == nil {
		traces = []db.ExecutionTrace{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: traces})
}

func apiListTaskExecutionsHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")

	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	// Check ownership
	exists, err := queries.CheckTaskOwnership(c.Request().Context(), db.CheckTaskOwnershipParams{
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to verify task ownership"})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	executions, err := queries.ListTaskExecutionIDs(c.Request().Context(), taskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch task executions"})
	}
	if executions == nil {
		executions = []db.ListTaskExecutionIDsRow{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: executions})
}

type ManualRouteRequest struct {
	TargetTaskID string `json:"target_task_id"`
}

func apiManualRouteHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	taskID, err := mustParseUUID(c, taskIDStr)
	if err != nil {
		return err
	}

	var req ManualRouteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	targetTaskID, err := mustParseUUID(c, req.TargetTaskID)
	if err != nil {
		return err
	}

	// 1. Verify source task ownership and state
	task, err := queries.GetTaskByID(c.Request().Context(), db.GetTaskByIDParams{
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Task not found"})
	}

	if task.LastApprovalStatus.String != "needs_routing" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Task is not in needs_routing state"})
	}

	// 2. Verify target task ownership and dependency
	targetTask, err := queries.GetTaskByID(c.Request().Context(), db.GetTaskByIDParams{
		ID:     targetTaskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Target task not found"})
	}

	if !targetTask.DependsOnTaskID.Valid || formatUUID(targetTask.DependsOnTaskID) != taskIDStr {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Target task does not depend on source task"})
	}

	// 3. Update target task to active and next_run = NOW()
	err = queries.UpdateTaskNextRun(c.Request().Context(), db.UpdateTaskNextRunParams{
		Status:  pgtype.Text{String: "active", Valid: true},
		NextRun: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ID:      targetTaskID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to trigger target task"})
	}

	// 4. Mark source task as completed (or approved)
	err = queries.UpdateTaskApprovalStatus(c.Request().Context(), db.UpdateTaskApprovalStatusParams{
		LastApprovalStatus: pgtype.Text{String: "manual_routed", Valid: true},
		Status:             pgtype.Text{String: "completed", Valid: true},
		ID:                 taskID,
		UserID:             userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update source task"})
	}

	// Audit Log
	writeAuditLog(c.Request().Context(), AuditEvent{
		UserID:       userID,
		Action:       "task.manual_route",
		ResourceType: "task",
		ResourceID:   taskIDStr,
		Metadata: map[string]interface{}{
			"target_task_id": req.TargetTaskID,
		},
	})

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task manually routed"})
}
