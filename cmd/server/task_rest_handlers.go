package main

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"schedule-mcp/db"
)

func apiListTasksHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}
	
	tasks, err := queries.ListUserTasks(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to list tasks"})
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
	AgentPrompt      string `json:"agent_prompt"`
	MissedTaskPolicy string `json:"missed_task_policy"`
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
	_, _ = queries.CreateTaskVersion(c.Request().Context(), db.CreateTaskVersionParams{
		ID:     taskID,
		UserID: userID,
	})

	err = queries.UpdateTaskAgentPromptAndPolicy(c.Request().Context(), db.UpdateTaskAgentPromptAndPolicyParams{
		AgentPrompt:      req.AgentPrompt,
		MissedTaskPolicy: pgtype.Text{String: req.MissedTaskPolicy, Valid: true},
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
	_, _ = queries.CreateTaskVersion(c.Request().Context(), db.CreateTaskVersionParams{
		ID:     taskID,
		UserID: userID,
	})

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
