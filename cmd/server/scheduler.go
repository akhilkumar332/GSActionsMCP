package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"schedule-mcp/db"
)

var promptVarRegex = regexp.MustCompile(`\{\{task\.([0-9a-fA-F-]{36})\.output\}\}`)
var stateVarRegex = regexp.MustCompile(`\{\{state\.([a-zA-Z0-9._-]+)\}\}`)

// runScheduler queries the DB every 10 seconds for due tasks
func runScheduler(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Claiming now emits a PostgreSQL NOTIFY from the same transaction.
			claimCtx, claimCancel := context.WithTimeout(ctx, 5*time.Second)
			tasks, err := queries.ClaimDueTasks(claimCtx, db.ClaimDueTasksParams{
				BatchSize: 50,
				WorkerID:  workerID,
			})
			claimCancel()

			if err != nil {
				schedulerClaimErrorsTotal.Inc()
				log.Printf("Error claiming tasks: %v", err)
				continue
			}
			if len(tasks) > 0 {
				schedulerClaimsTotal.Add(float64(len(tasks)))
				log.Printf("Claimed %d due task(s); waiting for NOTIFY dispatch", len(tasks))
			}
		case <-ctx.Done():
			return
		}
	}
}

type taskClaimNotification struct {
	TaskID   string `json:"task_id"`
	UserID   string `json:"user_id"`
	WorkerID string `json:"worker_id"`
}

func listenForTaskClaims(ctx context.Context, dbURL string) {
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := pgx.Connect(ctx, dbURL)
		if err != nil {
			log.Printf("Failed to connect task claim listener: %v. Retrying in %v...", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		_, err = conn.Exec(ctx, "LISTEN task_claimed")
		if err != nil {
			log.Printf("Failed to LISTEN for task claims: %v", err)
			conn.Close(context.Background())
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		backoff = time.Second
		log.Printf("Listening for PostgreSQL task claim notifications as worker %s", workerID)

		for {
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				conn.Close(context.Background())
				if ctx.Err() != nil {
					return
				}
				log.Printf("Task claim listener disconnected: %v", err)
				break
			}
			handleTaskClaimNotification(notification.Payload)
		}
	}
}

func handleTaskClaimNotification(payload string) {
	var notice taskClaimNotification
	if err := json.Unmarshal([]byte(payload), &notice); err != nil {
		log.Printf("Invalid task claim notification payload: %v", err)
		return
	}

	if notice.WorkerID != workerID {
		return
	}

	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		workerCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var taskID pgtype.UUID
		if err := parseUUID(notice.TaskID, &taskID); err != nil {
			log.Printf("Invalid task ID in notification: %s", notice.TaskID)
			return
		}

		t, err := queries.GetDispatchableTaskByID(workerCtx, db.GetDispatchableTaskByIDParams{
			ID:       taskID,
			UserID:   notice.UserID,
			LockedBy: pgtype.Text{String: workerID, Valid: true},
		})
		if err != nil {
			log.Printf("Skipping claimed task %s for worker %s: %v", notice.TaskID, workerID, err)
			return
		}

		handleDispatchTask(workerCtx, t, nil)
	}()
}

func handleDispatchTask(workerCtx context.Context, t db.Task, triggerPayload map[string]interface{}) {
	isOnline := GlobalSessionManager.IsOnline(workerCtx, t.UserID)

	taskID := formatUUID(t.ID)
	executionID := fmt.Sprintf("%s-%d", taskID, time.Now().UTC().UnixNano())

	// Load workflow state if it exists (try latest state for this task)
	var state map[string]interface{}
	stateBytes, _ := queries.GetLatestWorkflowState(workerCtx, t.ID)
	if len(stateBytes) > 0 {
		json.Unmarshal(stateBytes, &state)
	}

	userEmail, _ := queries.GetUserEmail(workerCtx, t.UserID)
	emailStr := ""
	if userEmail.Valid {
		emailStr = userEmail.String
	}

	if t.RequiresApproval.Bool && t.LastApprovalStatus.String != "approved" {
		observeTaskOutcome("approval_required")
		log.Printf("Task %s requires approval. Pausing.", taskID)
		if err := queries.UpdateTaskApprovalStatus(workerCtx, db.UpdateTaskApprovalStatusParams{
			LastApprovalStatus: pgtype.Text{String: "pending", Valid: true},
			Status:             pgtype.Text{String: StatusPaused, Valid: true},
			ID:                 t.ID,
			UserID:             t.UserID,
		}); err != nil {
			log.Printf("Error updating task approval status for %s: %v", taskID, err)
		}

		evtPayload, _ := json.Marshal(map[string]interface{}{
			"task_id":      taskID,
			"task_name":    t.Name,
			"execution_id": executionID,
		})
		if err := PublishEvent(workerCtx, PubSubEvent{
			UserID:    t.UserID,
			EventType: "approval_required",
			Payload:   string(evtPayload),
		}); err != nil {
			log.Printf("Error publishing approval_required event for %s: %v", taskID, err)
		}
		return
	}

	if !isOnline {
		observeTaskOutcome("missed")
		log.Printf("User %s is offline. Task %s missed.", t.UserID, taskID)
		logID, err := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
			TaskID:       t.ID,
			UserID:       t.UserID,
			Status:       "missed",
			ErrorMessage: pgtype.Text{String: "user offline", Valid: true},
		})
		if err != nil {
			log.Printf("Error creating task log for %s (missed): %v", taskID, err)
		}

		evtPayload, _ := json.Marshal(map[string]interface{}{
			"id":             formatUUID(logID),
			"task_id":        taskID,
			"status":         "missed",
			"execution_time": time.Now().Format(time.RFC3339),
			"task_name":      t.Name,
			"user_email":     emailStr,
			"error_message":  "user offline",
			"execution_id":   executionID,
		})
		if err := PublishEvent(workerCtx, PubSubEvent{
			UserID:    t.UserID,
			EventType: "task_executed",
			Payload:   string(evtPayload),
		}); err != nil {
			log.Printf("Error publishing task_executed event for %s (missed): %v", taskID, err)
		}

		if t.MissedTaskPolicy.String == PolicyRunImmediate {
			if err := queries.UpdateTaskNextRun(workerCtx, db.UpdateTaskNextRunParams{
				Status:  pgtype.Text{String: StatusActive, Valid: true},
				NextRun: pgtype.Timestamptz{Time: time.Now().UTC().Add(1 * time.Minute), Valid: true},
				ID:      t.ID,
			}); err != nil {
				log.Printf("Error updating next run for missed task %s: %v", taskID, err)
			}
			return
		}

		var config map[string]interface{}
		if err := json.Unmarshal(t.TriggerConfig, &config); err == nil {
			if newNextRun, calcErr := calculateNextRun(t.TriggerType.String, config, time.Now().UTC()); calcErr == nil {
				completeTask(workerCtx, t.UserID, taskID, newNextRun)
				return
			}
		}
		if err := queries.UpdateTaskStatus(workerCtx, db.UpdateTaskStatusParams{
			Status: pgtype.Text{String: StatusPaused, Valid: true},
			ID:     t.ID,
		}); err != nil {
			log.Printf("Error pausing task %s: %v", taskID, err)
		}
		return
	}

	if t.TaskType.String == "native_action" {
		inputMap := map[string]interface{}{
			"task_id":      taskID,
			"execution_id": executionID,
			"payload":      triggerPayload,
		}
		inputJSON, _ := json.Marshal(inputMap)
		queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{
			TaskID:      t.ID,
			ExecutionID: executionID,
			WorkerID:    workerID,
			StepName:    "Native Execution",
			InputData:   inputJSON,
		})

		result, err := executeNativeJS(workerCtx, t.NativeCode.String, inputMap)
		if err != nil {
			log.Printf("Native execution failed for task %s: %v", taskID, err)
			observeTaskOutcome("execution_failure")
			failureCount := t.FailureCount.Int32 + 1
			logID, logErr := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
				TaskID:       t.ID,
				UserID:       t.UserID,
				Status:       "failure",
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			})
			if logErr != nil {
				log.Printf("Error creating task log for %s (failure): %v", taskID, logErr)
			}

			evtPayload, _ := json.Marshal(map[string]interface{}{
				"id":             formatUUID(logID),
				"task_id":        taskID,
				"status":         "failure",
				"execution_time": time.Now().Format(time.RFC3339),
				"task_name":      t.Name,
				"user_email":     emailStr,
				"error_message":  err.Error(),
				"execution_id":   executionID,
			})
			if err := PublishEvent(workerCtx, PubSubEvent{
				UserID:    t.UserID,
				EventType: "task_executed",
				Payload:   string(evtPayload),
			}); err != nil {
				log.Printf("Error publishing task_executed event for %s (failure): %v", taskID, err)
			}

			queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{
				TaskID:       t.ID,
				ExecutionID:  executionID,
				WorkerID:     workerID,
				StepName:     "Native Execution Failed",
				IsError:      pgtype.Bool{Bool: true, Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			})

			retryCount := t.RetryCount.Int32 + 1
			maxRetries := t.MaxRetries.Int32
			if maxRetries == 0 {
				maxRetries = 3
			}

			if retryCount > maxRetries {
				log.Printf("Task %s exhausted retries (%d), moving to DLQ.", taskID, maxRetries)
				if err := queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
					Status:       pgtype.Text{String: StatusError, Valid: true},
					FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
					RetryCount:   pgtype.Int4{Int32: retryCount, Valid: true},
					ID:           t.ID,
					UserID:       t.UserID,
				}); err != nil {
					log.Printf("Error updating status to error for task %s: %v", taskID, err)
				}
				queries.MoveToDLQ(workerCtx, db.MoveToDLQParams{
					TaskID:       t.ID,
					ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
				})
				sendFailureEmail(workerCtx, t.UserID, taskID, t.Name)
			} else {
				backoffMinutes := int(retryCount) * 2
				if t.BackoffStrategy.String == "exponential" {
					backoffMinutes = 1 << retryCount
				}
				nextRun := time.Now().UTC().Add(time.Duration(backoffMinutes) * time.Minute)
				log.Printf("Task %s retry %d/%d scheduled for %v", taskID, retryCount, maxRetries, nextRun)
				queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
					Status:       pgtype.Text{String: StatusActive, Valid: true},
					FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
					RetryCount:   pgtype.Int4{Int32: retryCount, Valid: true},
					ID:           t.ID,
					UserID:       t.UserID,
				})
				queries.UpdateTaskNextRun(workerCtx, db.UpdateTaskNextRunParams{
					Status:  pgtype.Text{String: StatusActive, Valid: true},
					NextRun: pgtype.Timestamptz{Time: nextRun, Valid: true},
					ID:      t.ID,
				})
			}
		} else {
			observeTaskOutcome("success")
			logID, err := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
				TaskID:      t.ID,
				UserID:      t.UserID,
				Status:      "success",
				LlmResponse: pgtype.Text{String: result, Valid: true},
			})
			if err != nil {
				log.Printf("Error creating task log for %s (success): %v", taskID, err)
			}

			evtPayload, _ := json.Marshal(map[string]interface{}{
				"id":             formatUUID(logID),
				"task_id":        taskID,
				"status":         "success",
				"execution_time": time.Now().Format(time.RFC3339),
				"task_name":      t.Name,
				"user_email":     emailStr,
				"llm_response":   result,
				"execution_id":   executionID,
			})
			if err := PublishEvent(workerCtx, PubSubEvent{
				UserID:    t.UserID,
				EventType: "task_executed",
				Payload:   string(evtPayload),
			}); err != nil {
				log.Printf("Error publishing task_executed event for %s (success): %v", taskID, err)
			}

			queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{
				TaskID:      t.ID,
				ExecutionID: executionID,
				WorkerID:    workerID,
				StepName:    "Native Execution Success",
				OutputData:  []byte(result),
			})

			var config map[string]interface{}
			if err := json.Unmarshal(t.TriggerConfig, &config); err == nil {
				if newNextRun, calcErr := calculateNextRun(t.TriggerType.String, config, time.Now().UTC()); calcErr == nil {
					completeTask(workerCtx, t.UserID, taskID, newNextRun)
					return
				}
			}
			if err := queries.UpdateTaskStatus(workerCtx, db.UpdateTaskStatusParams{
				Status: pgtype.Text{String: StatusPaused, Valid: true},
				ID:     t.ID,
			}); err != nil {
				log.Printf("Error pausing task %s: %v", taskID, err)
			}
		}
		return
	}

	queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Prompt Resolution",
		InputData:   []byte(t.AgentPrompt),
	})

	resolvedPrompt := resolvePromptVariables(workerCtx, t.UserID, t.AgentPrompt, triggerPayload, state)

	traceContext := make(map[string]string)
	otel.GetTextMapPropagator().Inject(workerCtx, propagation.MapCarrier(traceContext))

	payloadMap := map[string]interface{}{
		"task_id":        taskID,
		"prompt":         resolvedPrompt,
		"execution_id":   executionID,
		"trigger_type":   t.TriggerType.String,
		"trigger_config": string(t.TriggerConfig),
		"trace_context":  traceContext,
	}
	if triggerPayload != nil {
		payloadMap["trigger_payload"] = triggerPayload
	}

	payloadBytes, _ := json.Marshal(payloadMap)
	subscribers, err := RedisClient.Publish(workerCtx, fmt.Sprintf("trigger_task:%s", t.UserID), string(payloadBytes)).Result()
	if err != nil || subscribers == 0 {
		observeTaskOutcome("delivery_failure")
		if err == nil {
			err = fmt.Errorf("no active subscribers received the payload")
		}
		log.Printf("Failed to deliver task %s for user %s: %v", taskID, t.UserID, err)

		queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{
			TaskID:       t.ID,
			ExecutionID:  executionID,
			WorkerID:     workerID,
			StepName:     "Redis Delivery Failed",
			IsError:      pgtype.Bool{Bool: true, Valid: true},
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		})

		failureCount := t.FailureCount.Int32 + 1
		logID, logErr := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
			TaskID:       t.ID,
			UserID:       t.UserID,
			Status:       "failure",
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		})
		if logErr != nil {
			log.Printf("Error creating task log for %s (failure): %v", taskID, logErr)
		}

		evtPayload, _ := json.Marshal(map[string]interface{}{
			"id":             formatUUID(logID),
			"task_id":        taskID,
			"status":         "failure",
			"execution_time": time.Now().Format(time.RFC3339),
			"task_name":      t.Name,
			"user_email":     emailStr,
			"error_message":  err.Error(),
		})
		if err := PublishEvent(workerCtx, PubSubEvent{
			UserID:    t.UserID,
			EventType: "task_executed",
			Payload:   string(evtPayload),
		}); err != nil {
			log.Printf("Error publishing task_executed event for %s (failure): %v", taskID, err)
		}

		retryCount := t.RetryCount.Int32 + 1
		maxRetries := t.MaxRetries.Int32
		if maxRetries == 0 {
			maxRetries = 3 // Fallback for old tasks
		}

		if retryCount > maxRetries {
			log.Printf("Task %s exhausted retries (%d), moving to DLQ.", taskID, maxRetries)
			if err := queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
				Status:       pgtype.Text{String: StatusError, Valid: true},
				FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
				RetryCount:   pgtype.Int4{Int32: retryCount, Valid: true},
				ID:           t.ID,
				UserID:       t.UserID,
			}); err != nil {
				log.Printf("Error updating status to error for task %s: %v", taskID, err)
			}

			// Move to DLQ
			_, dlqErr := queries.MoveToDLQ(workerCtx, db.MoveToDLQParams{
				TaskID:       t.ID,
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			})
			if dlqErr != nil {
				log.Printf("Error moving task %s to DLQ: %v", taskID, dlqErr)
			}

			sendFailureEmail(workerCtx, t.UserID, taskID, t.Name)
		} else {
			// Calculate backoff
			backoffMinutes := int(retryCount) * 2
			if t.BackoffStrategy.String == "exponential" {
				backoffMinutes = 1 << retryCount // 2^retryCount
			}
			nextRun := time.Now().UTC().Add(time.Duration(backoffMinutes) * time.Minute)

			log.Printf("Task %s retry %d/%d scheduled for %v", taskID, retryCount, maxRetries, nextRun)

			// Update task with new retry_count and next_run
			// Assuming we should update failure_count as well and keep status Active
			if err := queries.UpdateTaskStatusAndFailureCount(workerCtx, db.UpdateTaskStatusAndFailureCountParams{
				Status:       pgtype.Text{String: StatusActive, Valid: true},
				FailureCount: pgtype.Int4{Int32: failureCount, Valid: true},
				RetryCount:   pgtype.Int4{Int32: retryCount, Valid: true},
				ID:           t.ID,
				UserID:       t.UserID,
			}); err != nil {
				log.Printf("Error updating failure count for task %s: %v", taskID, err)
			}
			// Update next run
			if err := queries.UpdateTaskNextRun(workerCtx, db.UpdateTaskNextRunParams{
				Status:  pgtype.Text{String: StatusActive, Valid: true},
				NextRun: pgtype.Timestamptz{Time: nextRun, Valid: true},
				ID:      t.ID,
			}); err != nil {
				log.Printf("Error updating next run for task %s: %v", taskID, err)
			}
		}
		return
	}

	observeTaskOutcome("delivered")
	logID, err := queries.CreateTaskLog(workerCtx, db.CreateTaskLogParams{
		TaskID:      t.ID,
		UserID:      t.UserID,
		Status:      "success",
		LlmResponse: pgtype.Text{String: "Task delivered to node via Redis", Valid: true},
	})
	if err != nil {
		log.Printf("Error creating task log for %s (delivered): %v", taskID, err)
	}

	evtPayload, _ := json.Marshal(map[string]interface{}{
		"id":             formatUUID(logID),
		"task_id":        taskID,
		"status":         "success",
		"execution_time": time.Now().Format(time.RFC3339),
		"task_name":      t.Name,
		"user_email":     emailStr,
		"llm_response":   "Task delivered to node via Redis",
	})
	if err := PublishEvent(workerCtx, PubSubEvent{
		UserID:    t.UserID,
		EventType: "task_executed",
		Payload:   string(evtPayload),
	}); err != nil {
		log.Printf("Error publishing task_executed event for %s (delivered): %v", taskID, err)
	}

	log.Printf("Task %s delivered to node. Remaining in 'processing' status.", taskID)
}

func resolvePromptVariables(ctx context.Context, userID string, prompt string, triggerPayload map[string]interface{}, state map[string]interface{}) string {
	resolved := promptVarRegex.ReplaceAllStringFunc(prompt, func(match string) string {
		submatch := promptVarRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		taskIDStr := submatch[1]
		var tid pgtype.UUID
		if err := parseUUID(taskIDStr, &tid); err != nil {
			return match
		}

		// Ensure ownership and fetch output
		output, err := queries.GetTaskOutput(ctx, db.GetTaskOutputParams{
			TaskID: tid,
			UserID: userID,
		})
		if err != nil {
			log.Printf("Error fetching output for task %s (user %s): %v", taskIDStr, userID, err)
			return match
		}
		return string(output)
	})

	// Support {{state.FIELD}}
	resolved = stateVarRegex.ReplaceAllStringFunc(resolved, func(match string) string {
		submatches := stateVarRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		key := submatches[1]
		if state != nil {
			if val, ok := state[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
		return match
	})

	// Also support {{webhook.body.FIELD}} in normal scheduler flows if payload exists
	if triggerPayload != nil {
		resolved = webhookBodyRegex.ReplaceAllStringFunc(resolved, func(match string) string {
			submatches := webhookBodyRegex.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			key := submatches[1]
			if val, ok := triggerPayload[key]; ok {
				return fmt.Sprintf("%v", val)
			}
			return match
		})
	}

	return resolved
}

func evaluateLoopCondition(loopConfig []byte, lastOutput string) bool {
	if len(loopConfig) == 0 {
		return false
	}
	var config map[string]interface{}
	if err := json.Unmarshal(loopConfig, &config); err != nil {
		log.Printf("Error unmarshaling loop config: %v", err)
		return false
	}

	enabled, _ := config["enabled"].(bool)
	if !enabled {
		return false
	}

	condition, _ := config["condition"].(string)
	value, _ := config["value"].(string)

	switch condition {
	case "contains":
		return strings.Contains(lastOutput, value)
	case "not_contains":
		return !strings.Contains(lastOutput, value)
	default:
		return false
	}
}

func evaluateBranchCondition(condition []byte, parentOutput string) bool {
	if len(condition) == 0 {
		return true
	}
	var cond map[string]string
	if err := json.Unmarshal(condition, &cond); err != nil {
		log.Printf("Error unmarshaling branch condition: %v", err)
		return true // Default to true if condition is malformed
	}

	op := cond["if"]
	val := cond["value"]

	switch op {
	case "contains":
		return strings.Contains(parentOutput, val)
	default:
		return true
	}
}

// completeTask calls the PLpgSQL function to set the task back to active and update next_run
func completeTask(ctx context.Context, userID string, taskID string, nextRun time.Time, status ...string) {
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
		return
	}

	// Step 1: Trigger dependent tasks immediately
	dependents, err := queries.GetDependentTasksToTrigger(ctx, tid)
	if err != nil {
		log.Printf("Error fetching dependent tasks for %s: %v", taskID, err)
		return
	}

	// Fetch last output of the parent task
	parentOutputBytes, _ := queries.GetTaskOutput(ctx, db.GetTaskOutputParams{
		TaskID: tid,
		UserID: userID,
	})
	parentOutput := string(parentOutputBytes)

	for _, t := range dependents {
		if !evaluateBranchCondition(t.BranchCondition, parentOutput) {
			log.Printf("Skipping dependent task %s: branch condition not met", formatUUID(t.ID))
			continue
		}

		if err := queries.UpdateTaskNextRun(ctx, db.UpdateTaskNextRunParams{
			Status:  pgtype.Text{String: StatusActive, Valid: true},
			NextRun: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ID:      t.ID,
		}); err != nil {
			log.Printf("Error making dependent task %s due immediately for user %s: %v", formatUUID(t.ID), t.UserID, err)
			continue
		}
		log.Printf("Queued dependent task %s for immediate execution", formatUUID(t.ID))
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

func runWorkerHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	hostname, _ := os.Hostname()

	for {
		select {
		case <-ticker.C:
			err := queries.UpsertWorkerHeartbeat(ctx, db.UpsertWorkerHeartbeatParams{
				WorkerID:  workerID,
				Hostname:  pgtype.Text{String: hostname, Valid: true},
				TaskCount: pgtype.Int4{Int32: 0, Valid: true}, // In future, track active go-routines
			})
			if err != nil {
				log.Printf("Heartbeat error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
