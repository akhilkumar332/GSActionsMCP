package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/robfig/cron/v3"
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
			tasks := claimDueTasks(claimCtx, 50, workerID)
			claimCancel()

			for _, task := range tasks {
				workerWG.Add(1)
				go func(t Task) {
					defer workerWG.Done()
					workerCtx := context.Background()
					// 2. Check if user is online via Redis
					isOnline := GlobalSessionManager.IsOnline(workerCtx, t.UserID)

					if !isOnline {
						log.Printf("User %s is offline. Task %s missed.", t.UserID, t.ID)
						// Phase 1.2: Execution Logging
						_, _ = dbPool.Exec(workerCtx, "INSERT INTO task_logs (task_id, user_id, status, error_message) VALUES ($1, $2, 'missed', 'user offline')", t.ID, t.UserID)
						
						// Phase 3.1: Missed Task Policy
						if t.MissedTaskPolicy == PolicyRunImmediate {
							// Push next_run ahead by 1 minute to prevent rapid polling spam while offline
							_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL, next_run = NOW() + INTERVAL '1 minute' WHERE id = $2", StatusActive, t.ID)
							return
						} else {
							// "skip": calculate next run and update
							var config map[string]interface{}
							if err := json.Unmarshal(t.TriggerConfig, &config); err == nil {
								if newNextRun, calcErr := calculateNextRun(t.TriggerType, config, time.Now().UTC()); calcErr == nil {
									completeTask(workerCtx, t.ID, newNextRun)
									return
								}
							}
							_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2", StatusPaused, t.ID)
							return
						}
					}

					// Phase 9.2: Cross-Task Context Check
					finalPrompt := t.AgentPrompt
					if t.DependsOnTaskID != nil {
						var parentOutput sql.NullString
						err := dbPool.QueryRow(workerCtx, "SELECT llm_response FROM task_logs WHERE task_id = $1 ORDER BY execution_time DESC LIMIT 1", *t.DependsOnTaskID).Scan(&parentOutput)
						if err == nil && parentOutput.Valid && parentOutput.String != "" {
							finalPrompt = fmt.Sprintf("Context from previous task:\n%s\n\nYour Prompt:\n%s", parentOutput.String, t.AgentPrompt)
						}
					}

					// Phase 6.1: Publish to Redis Pub/Sub so the correct node with the SSE connection can trigger it
					executionID := fmt.Sprintf("%s-%d", t.ID, time.Now().UTC().UnixNano())
					payloadBytes, _ := json.Marshal(map[string]interface{}{
						"task_id":        t.ID,
						"prompt":         finalPrompt,
						"execution_id":   executionID,
						"trigger_type":   t.TriggerType,
						"trigger_config": string(t.TriggerConfig),
					})
					subscribers, err := redisClient.Publish(workerCtx, fmt.Sprintf("trigger_task:%s", t.UserID), string(payloadBytes)).Result()
					if err != nil || subscribers == 0 {
						if err == nil {
							err = fmt.Errorf("no active subscribers received the payload")
						}
						log.Printf("Failed to deliver task %s for user %s: %v", t.ID, t.UserID, err)
						// Phase 2.3: Dead Letter Queue
						t.FailureCount++
						_, _ = dbPool.Exec(workerCtx, "INSERT INTO task_logs (task_id, user_id, status, error_message) VALUES ($1, $2, 'failure', $3)", t.ID, t.UserID, err.Error())
						
						if t.FailureCount >= 3 {
							log.Printf("Task %s failed 3 times, setting status to error.", t.ID)
							_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL, failure_count = $2 WHERE id = $3", StatusError, t.FailureCount, t.ID)
							
							// Phase 9.1: Real Dead Letter Email Alert
							sendFailureEmail(workerCtx, t.UserID, t.ID, t.Name)
						} else {
							_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL, failure_count = $2 WHERE id = $3", StatusActive, t.FailureCount, t.ID)
						}
						return
					}

					// Log Success delivery to node (session.go will log the actual LLM response)
					_, _ = dbPool.Exec(workerCtx, "INSERT INTO task_logs (task_id, user_id, status, llm_response) VALUES ($1, $2, 'success', 'Task delivered to node via Redis')", t.ID, t.UserID)

					// Iteration 2: We no longer update the task status or call completeTask here.
					// The execution node (session.go) is now responsible for advancing the task.
					log.Printf("Task %s delivered to node. Remaining in 'processing' status.", t.ID)
				}(task)
			}
		case <-ctx.Done():
			return
		}
	}
}

// claimDueTasks calls the PLpgSQL function to atomically get and lock tasks
func claimDueTasks(ctx context.Context, batchSize int, wID string) []Task {
	rows, err := dbPool.Query(ctx, "SELECT * FROM fn_claim_due_tasks($1, $2)", batchSize, wID)
	if err != nil {
		log.Printf("Error claiming tasks: %v", err)
		return nil
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var lockedBy *string
		var lastRun *time.Time
		var createdAt time.Time
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.TriggerType, &t.TriggerConfig,
			&t.AgentPrompt, &t.Status, &lockedBy, &t.NextRun, &lastRun,
			&t.FailureCount, &t.MissedTaskPolicy, &t.DependsOnTaskID, &createdAt,
		)
		if err == nil {
			tasks = append(tasks, t)
		} else {
			log.Printf("Error scanning task: %v", err)
		}
	}
	return tasks
}

// completeTask calls the PLpgSQL function to set the task back to active and update next_run
func completeTask(ctx context.Context, taskID string, nextRun time.Time, status ...string) {
	finalStatus := StatusActive
	if len(status) > 0 {
		finalStatus = status[0]
	}
	_, err := dbPool.Exec(ctx, "SELECT fn_complete_task($1, $2, $3)", taskID, nextRun, finalStatus)
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
			res, err := dbPool.Exec(ctx, "UPDATE tasks SET status = 'active', locked_by = NULL WHERE status = 'processing' AND next_run < NOW() - INTERVAL '5 minutes'")
			if err != nil {
				log.Printf("Reaper error: %v", err)
			} else {
				if rows := res.RowsAffected(); rows > 0 {
					log.Printf("Reaper: recovered %d stuck tasks", rows)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
