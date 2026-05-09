package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/server"
	"github.com/robfig/cron/v3"
	"schedule-mcp/db"
)

// runScheduler queries the DB every 10 seconds for due tasks
func runScheduler(ctx context.Context, s *server.MCPServer) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 1. Claim batch of tasks using PLpgSQL function with timeout
			claimCtx, claimCancel := context.WithTimeout(ctx, 5*time.Second)
			tasks, err := queries.ClaimDueTasks(claimCtx, db.ClaimDueTasksParams{
				BatchSize: 50,
				WorkerID:  workerID,
			})
			claimCancel()

			if err != nil {
				log.Printf("Error claiming tasks: %v", err)
				continue
			}

			for _, task := range tasks {
				workerWG.Add(1)
				go func(t db.Task) {
					defer workerWG.Done()
					workerCtx := context.Background()
					// 2. Check if user is online via Redis
					isOnline := GlobalSessionManager.IsOnline(workerCtx, t.UserID)

					taskID := formatUUID(t.ID)
					userEmail, _ := queries.GetUserEmail(workerCtx, t.UserID)
					emailStr := ""
					if userEmail.Valid {
						emailStr = userEmail.String
					}

					// Human-in-the-Loop: Check if approval is required
					if t.RequiresApproval.Bool && t.LastApprovalStatus.String != "approved" {
						log.Printf("Task %s requires approval. Pausing.", taskID)
						_ = queries.UpdateTaskApprovalStatus(workerCtx, db.UpdateTaskApprovalStatusParams{
							LastApprovalStatus: pgtype.Text{String: "pending", Valid: true},
							Status:             pgtype.Text{String: StatusPaused, Valid: true},
							ID:                 t.ID,
							UserID:             t.UserID,
						})
						evtPayload, _ := json.Marshal(map[string]interface{}{
							"task_id":   taskID,
							"task_name": t.Name,
						})
						_ = PublishEvent(workerCtx, PubSubEvent{
							UserID:    t.UserID,
							EventType: "approval_required",
							Payload:   string(evtPayload),
						})
						return
					}

					if !isOnline {
						log.Printf("User %s is offline. Task %s missed.", t.UserID, taskID)
						// Phase 1.2: Execution Logging
						logID, _ := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
							TaskID:       t.ID,
							UserID:       t.UserID,
							Status:       "missed",
							ErrorMessage: pgtype.Text{String: "user offline", Valid: true},
						})

						// Emit Redis event
						evtPayload, _ := json.Marshal(map[string]interface{}{
							"id":             formatUUID(logID),
							"task_id":        taskID,
							"status":         "missed",
							"execution_time": time.Now().Format(time.RFC3339),
							"task_name":      t.Name,
							"user_email":     emailStr,
							"error_message":  "user offline",
						})
						_ = PublishEvent(workerCtx, PubSubEvent{
							UserID:    t.UserID,
							EventType: "task_executed",
							Payload:   string(evtPayload),
						})
						
						// Phase 3.1: Missed Task Policy
						if t.MissedTaskPolicy.String == PolicyRunImmediate {
							// Push next_run ahead by 1 minute to prevent rapid polling spam while offline
							_ = queries.UpdateTaskNextRun(workerCtx, db.UpdateTaskNextRunParams{
								Status:  pgtype.Text{String: StatusActive, Valid: true},
								NextRun: pgtype.Timestamptz{Time: time.Now().UTC().Add(1 * time.Minute), Valid: true},
								ID:      t.ID,
							})
							return
						} else {
							// "skip": calculate next run and update
							var config map[string]interface{}
							if err := json.Unmarshal(t.TriggerConfig, &config); err == nil {
								if newNextRun, calcErr := calculateNextRun(t.TriggerType.String, config, time.Now().UTC()); calcErr == nil {
									completeTask(workerCtx, taskID, newNextRun)
									return
								}
							}
							_ = queries.UpdateTaskStatus(workerCtx, db.UpdateTaskStatusParams{
								Status: pgtype.Text{String: StatusPaused, Valid: true},
								ID:     t.ID,
							})
							return
						}
					}

					// Phase 9.2: Cross-Task Context Check
					finalPrompt := t.AgentPrompt
					if t.DependsOnTaskID.Valid {
						parentOutput, err := queries.GetLatestTaskLogResponse(workerCtx, t.DependsOnTaskID)
						if err == nil && parentOutput.Valid && parentOutput.String != "" {
							finalPrompt = fmt.Sprintf("Context from previous task:\n%s\n\nYour Prompt:\n%s", parentOutput.String, t.AgentPrompt)
						}
					}

					// Phase 6.1: Publish to Redis Pub/Sub so the correct node with the SSE connection can trigger it
					executionID := fmt.Sprintf("%s-%d", taskID, time.Now().UTC().UnixNano())
					payloadBytes, _ := json.Marshal(map[string]interface{}{
						"task_id":        taskID,
						"prompt":         finalPrompt,
						"execution_id":   executionID,
						"trigger_type":   t.TriggerType.String,
						"trigger_config": string(t.TriggerConfig),
					})
					subscribers, err := RedisClient.Publish(workerCtx, fmt.Sprintf("trigger_task:%s", t.UserID), string(payloadBytes)).Result()
					if err != nil || subscribers == 0 {
						if err == nil {
							err = fmt.Errorf("no active subscribers received the payload")
						}
						log.Printf("Failed to deliver task %s for user %s: %v", taskID, t.UserID, err)
						// Phase 2.3: Dead Letter Queue
						failureCount := t.FailureCount.Int32 + 1
						logID, _ := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
							TaskID:       t.ID,
							UserID:       t.UserID,
							Status:       "failure",
							ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
						})

						// Emit Redis event
						evtPayload, _ := json.Marshal(map[string]interface{}{
							"id":             formatUUID(logID),
							"task_id":        taskID,
							"status":         "failure",
							"execution_time": time.Now().Format(time.RFC3339),
							"task_name":      t.Name,
							"user_email":     emailStr,
							"error_message":  err.Error(),
						})
						_ = PublishEvent(workerCtx, PubSubEvent{
							UserID:    t.UserID,
							EventType: "task_executed",
							Payload:   string(evtPayload),
						})
						
						if failureCount >= 3 {
							log.Printf("Task %s failed 3 times, setting status to error.", taskID)
							_ = queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
								Status:       pgtype.Text{String: StatusError, Valid: true},
								FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
								ID:           t.ID,
							})
							
							// Phase 9.1: Real Dead Letter Email Alert
							sendFailureEmail(workerCtx, t.UserID, taskID, t.Name)
						} else {
							_ = queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
								Status:       pgtype.Text{String: StatusActive, Valid: true},
								FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
								ID:           t.ID,
							})
						}
						return
					}

					// Log Success delivery to node (session.go will log the actual LLM response)
					logID, _ := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
						TaskID:      t.ID,
						UserID:      t.UserID,
						Status:      "success",
						LlmResponse: pgtype.Text{String: "Task delivered to node via Redis", Valid: true},
					})

					// Emit Redis event
					evtPayload, _ := json.Marshal(map[string]interface{}{
						"id":             formatUUID(logID),
						"task_id":        taskID,
						"status":         "success",
						"execution_time": time.Now().Format(time.RFC3339),
						"task_name":      t.Name,
						"user_email":     emailStr,
						"llm_response":   "Task delivered to node via Redis",
					})
					_ = PublishEvent(workerCtx, PubSubEvent{
						UserID:    t.UserID,
						EventType: "task_executed",
						Payload:   string(evtPayload),
					})

					// Iteration 2: We no longer update the task status or call completeTask here.
					// The execution node (session.go) is now responsible for advancing the task.
					log.Printf("Task %s delivered to node. Remaining in 'processing' status.", taskID)
				}(task)
			}
		case <-ctx.Done():
			return
		}
	}
}

// completeTask calls the PLpgSQL function to set the task back to active and update next_run
func completeTask(ctx context.Context, taskID string, nextRun time.Time, status ...string) {
	finalStatus := StatusActive
	if len(status) > 0 {
		finalStatus = status[0]
	}
	
	var tid pgtype.UUID
	if err := parseUUID(taskID, &tid); err != nil {
		log.Printf("Invalid task ID in completeTask: %s", taskID)
		return
	}

	err := queries.CompleteTask(ctx, db.CompleteTaskParams{
		TaskID:     tid,
		NewNextRun: pgtype.Timestamptz{Time: nextRun, Valid: true},
		NewStatus:  finalStatus,
	})
	if err != nil {
		log.Printf("Error completing task %s: %v", taskID, err)
	}
}

// calculateNextRun determines the next run time based on trigger_type and trigger_config
func calculateNextRun(triggerType string, config map[string]interface{}, now time.Time) (time.Time, error) {
	switch triggerType {
	case TriggerCron:
		cronExpr, ok := config["cron"].(string)
		if !ok {
			return now, fmt.Errorf("missing cron expression")
		}
		
		// Handle optional timezone
		if tz, ok := config["timezone"].(string); ok && tz != "" {
			cronExpr = fmt.Sprintf("CRON_TZ=%s %s", tz, cronExpr)
		}

		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err := parser.Parse(cronExpr)
		if err != nil {
			return now, err
		}
		return schedule.Next(now).UTC(), nil

	case TriggerInterval:
		minutes, ok := config["minutes"].(float64)
		if !ok {
			return now, fmt.Errorf("missing minutes for interval")
		}
		return now.Add(time.Duration(minutes) * time.Minute), nil

	case TriggerDate:
		dateStr, ok := config["date"].(string)
		if !ok {
			return now, fmt.Errorf("missing date string")
		}
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return now, err
		}
		return t.UTC(), nil

	default:
		return now, fmt.Errorf("unknown trigger type: %s", triggerType)
	}
}

// runReaper recovers tasks that were locked by a worker that crashed.
// It runs every minute and resets tasks in 'processing' that have been stuck for more than 5 minutes.
func runReaper(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rows, err := queries.ReapStuckTasks(ctx)
			if err != nil {
				log.Printf("Reaper error: %v", err)
			} else {
				if rows > 0 {
					log.Printf("Reaper: recovered %d stuck tasks", rows)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
