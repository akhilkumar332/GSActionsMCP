package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"schedule-mcp/db"
)

type CreateTaskRequest struct {
	Name             string          `json:"name"`
	WorkspaceID      string          `json:"workspace_id"`
	TaskType         string          `json:"task_type"`
	AgentPrompt      string          `json:"agent_prompt"`
	NativeCode       string          `json:"native_code"`
	TriggerType      string          `json:"trigger_type"`
	TriggerConfig    json.RawMessage `json:"trigger_config"`
	RequiresApproval bool            `json:"requires_approval"`
	MissedTaskPolicy string          `json:"missed_task_policy"`
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
		if err := parseUUID(req.WorkspaceID, &workspaceID); err != nil {
			return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid workspace ID"})
		}
	}

	// Default missed task policy if not provided
	policy := req.MissedTaskPolicy
	if policy == "" {
		policy = "run_immediately"
	}

	params := db.CreateTaskParams{
		UserID:           userID,
		Name:             req.Name,
		TriggerType:      pgtype.Text{String: req.TriggerType, Valid: true},
		TriggerConfig:    req.TriggerConfig,
		AgentPrompt:      req.AgentPrompt,
		WorkspaceID:      workspaceID,
		TaskType:         pgtype.Text{String: req.TaskType, Valid: true},
		NativeCode:       pgtype.Text{String: req.NativeCode, Valid: true},
		MissedTaskPolicy: pgtype.Text{String: policy, Valid: true},
		RequiresApproval: pgtype.Bool{Bool: req.RequiresApproval, Valid: true},
		NextRun:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	task, err := queries.CreateTask(c.Request().Context(), params)
	if err != nil {
		log.Printf("Failed to create task: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create task"})
	}

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

	var taskID pgtype.UUID
	err := parseUUID(taskIDStr, &taskID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}

	err = queries.UpdateTaskStatusByUserID(c.Request().Context(), db.UpdateTaskStatusByUserIDParams{
		Status: pgtype.Text{String: "paused", Valid: true},
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to pause task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task paused"})
}

func apiResumeTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	var taskID pgtype.UUID
	err := parseUUID(taskIDStr, &taskID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}

	err = queries.UpdateTaskStatusByUserID(c.Request().Context(), db.UpdateTaskStatusByUserIDParams{
		Status: pgtype.Text{String: "active", Valid: true},
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to resume task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task resumed"})
}

func apiDeleteTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	var taskID pgtype.UUID
	err := parseUUID(taskIDStr, &taskID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}

	err = queries.DeleteTask(c.Request().Context(), db.DeleteTaskParams{
		ID:     taskID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task deleted"})
}

type UpdateTaskRequest struct {
	AgentPrompt      string          `json:"agent_prompt"`
	MissedTaskPolicy string          `json:"missed_task_policy"`
	UICoordinates    json.RawMessage `json:"ui_coordinates"`
}

func apiUpdateTaskHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	var taskID pgtype.UUID
	err := parseUUID(taskIDStr, &taskID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
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

	_, err = queries.UpdateTaskAgentPromptAndPolicy(c.Request().Context(), db.UpdateTaskAgentPromptAndPolicyParams{
		AgentPrompt:      req.AgentPrompt,
		MissedTaskPolicy: pgtype.Text{String: req.MissedTaskPolicy, Valid: true},
		UiCoordinates:    req.UICoordinates,
		ID:               taskID,
		UserID:           userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update task"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task updated"})
}

func apiListTaskVersionsHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	taskIDStr := c.Param("id")
	var taskID pgtype.UUID
	if err := parseUUID(taskIDStr, &taskID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
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

	var taskID, versionID pgtype.UUID
	if err := parseUUID(taskIDStr, &taskID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid task ID"})
	}
	if err := parseUUID(versionIDStr, &versionID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid version ID"})
	}

	// 1. Create a snapshot of CURRENT state before rolling back
	if _, err := queries.CreateTaskVersion(c.Request().Context(), db.CreateTaskVersionParams{
		ID:     taskID,
		UserID: userID,
	}); err != nil {
		log.Printf("Warning: Failed to create current state snapshot before restore for %s: %v", taskIDStr, err)
	}

	// 2. Restore
	err := queries.RestoreTaskFromVersion(c.Request().Context(), db.RestoreTaskFromVersionParams{
		ID:     taskID,
		UserID: userID,
		ID_2:   versionID, // ID_2 is the version ID in RestoreTaskFromVersionParams
	})
	if err != nil {
		log.Printf("Restore failed: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Restore failed"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Task restored successfully"})
}
