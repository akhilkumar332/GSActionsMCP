package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"actionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func handleInboundWebhook(c echo.Context) error {
	token := c.Param("token")

	// Get task by token
	task, err := queries.GetTaskByWebhookToken(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Invalid webhook token"})
	}

	taskID := formatUUID(task.ID)

	// Claim the task for this worker
	claimedTask, err := queries.ClaimTaskByID(c.Request().Context(), db.ClaimTaskByIDParams{
		LockedBy: pgtype.Text{String: workerID, Valid: true},
		ID:       task.ID,
	})
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Task is already being processed or is not active"})
	}

	// Parse webhook body if it exists
	payload := make(map[string]interface{})
	if c.Request().Header.Get("Content-Type") == "application/json" {
		if err := c.Bind(&payload); err != nil {
			log.Printf("Error binding webhook payload for task %s: %v", taskID, err)
		}
	}

	// Trigger the task immediately in a background goroutine to not block the webhook response
	workerWG.Add(1)
	go func(p map[string]interface{}) {
		defer workerWG.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in inbound webhook worker: %v", r)
			}
		}()
		workerCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		handleDispatchTask(workerCtx, claimedTask, p)
	}(payload)

	return c.JSON(http.StatusOK, map[string]string{"status": "task triggered", "task_id": taskID})
}
