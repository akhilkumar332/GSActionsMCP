package main

import (
	"context"
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
								if newNextRun, calcErr := calculateNextRun(t.TriggerType, config, time.Now()); calcErr == nil {
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
						var parentOutput string
						err := dbPool.QueryRow(workerCtx, "SELECT llm_response FROM task_logs WHERE task_id = $1 ORDER BY execution_time DESC LIMIT 1", *t.DependsOnTaskID).Scan(&parentOutput)
						if err == nil && parentOutput != "" {
							finalPrompt = fmt.Sprintf("Context from previous task:\n%s\n\nYour Prompt:\n%s", parentOutput, t.AgentPrompt)
						}
					}

					// Phase 6.1: Publish to Redis Pub/Sub so the correct node with the SSE connection can trigger it
					executionID := fmt.Sprintf("%s-%d", t.ID, time.Now().UnixNano())
					payloadBytes, _ := json.Marshal(map[string]string{
						"task_id":      t.ID,
						"prompt":       finalPrompt,
						"execution_id": executionID,
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

					// 4. Handle one-off vs recurring tasks
					if t.TriggerType == TriggerDate {
						_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL, last_run = NOW(), failure_count = 0 WHERE id = $2", StatusCompleted, t.ID)
						return
					}

					var config map[string]interface{}
					if err := json.Unmarshal(t.TriggerConfig, &config); err != nil {
						log.Printf("Error unmarshaling trigger config for task %s: %v", t.ID, err)
						_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2", StatusPaused, t.ID)
						return
					}

					newNextRun, calcErr := calculateNextRun(t.TriggerType, config, time.Now())
					if calcErr != nil {
						log.Printf("Error calculating next run for task %s: %v", t.ID, calcErr)
						// Fallback: pause the task if we can't calculate next run
						_, _ = dbPool.Exec(workerCtx, "UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2", StatusPaused, t.ID)
						return
					}

					// 5. Update next_run in DB using the PLpgSQL complete function
					completeTask(workerCtx, t.ID, newNextRun)
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
func completeTask(ctx context.Context, taskID string, nextRun time.Time) {
	_, err := dbPool.Exec(ctx, "SELECT fn_complete_task($1, $2)", taskID, nextRun)
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
