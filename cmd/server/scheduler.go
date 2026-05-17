package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"aktionfy/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var promptVarRegex = regexp.MustCompile(`\{\{task\.([0-9a-fA-F-]{36})\.output\}\}`)
var stateVarRegex = regexp.MustCompile(`\{\{state\.([a-zA-Z0-9._-]+)\}\}`)

const (
	PersonaArchitect = "Architect (Logic): You focus on structural integrity, technical consistency, and ensuring the output follows a logical flow. Critique the output against the available branches."
	PersonaSecurity  = "Security Officer (Safety): You scan for high-risk outcomes, malicious intent, or potential safety violations. Critique the output from a risk perspective."
	PersonaAdvocate  = "User Advocate (Intent): You prioritize the end-user's goal and helpfulness. Critique whether the output truly serves the user's original request."
)

func listenForTaskQueued(ctx context.Context, dbURL string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in listenForTaskQueued: %v", r)
		}
	}()
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		err := processTaskQueue(ctx, dbURL)
		if err != nil {
			log.Printf("Task queue listener error: %v. Retrying in %v...", err, backoff)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		} else {
			backoff = time.Second
		}
	}
}

func processTaskQueue(ctx context.Context, dbURL string) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(ctx, "LISTEN task_queued")
	if err != nil {
		return fmt.Errorf("failed to LISTEN for task_queued: %w", err)
	}

	for {
		_, err := conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("wait for notification failed: %w", err)
		}
		
		// A task is instantly ready. Attempt a claim immediately.
		claimCtx, claimCancel := context.WithTimeout(ctx, 5*time.Second)
		tasks, err := queries.ClaimDueTasks(claimCtx, db.ClaimDueTasksParams{
			BatchSize: 10, // Small batch for instant events
			WorkerID:  workerID,
		})
		claimCancel()
		if err == nil && len(tasks) > 0 {
			schedulerClaimsTotal.Add(float64(len(tasks)))
			log.Printf("Event-driven claim: %d task(s)", len(tasks))
		}
	}
}

// runScheduler queries the DB every 10 seconds for due tasks
func runScheduler(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in runScheduler: %v", r)
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

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
		} else if len(tasks) > 0 {
			schedulerClaimsTotal.Add(float64(len(tasks)))
			log.Printf("Claimed %d due task(s); waiting for NOTIFY dispatch", len(tasks))
		}

		pollInterval := CurrentSystemSettings.GetSchedulerPollInterval()
		timer := time.NewTimer(pollInterval)
		select {
		case <-timer.C:
			// Loop again
		case <-ctx.Done():
			timer.Stop()
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in listenForTaskClaims: %v", r)
		}
	}()
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := pgx.Connect(ctx, dbURL)
		if err != nil {
			log.Printf("Failed to connect task claim listener: %v. Retrying in %v...", err, backoff)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
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
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
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
			handleTaskClaimNotification(ctx, notification.Payload)
		}
	}
}

func handleTaskClaimNotification(ctx context.Context, payload string) {
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
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in task claim worker: %v", r)
			}
		}()
		workerCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
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
	stateBytes, err := queries.GetLatestWorkflowState(workerCtx, t.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error fetching workflow state for dispatch %s: %v", taskID, err)
	}
	if len(stateBytes) > 0 {
		json.Unmarshal(stateBytes, &state)
	}

	userEmail, err := queries.GetUserEmail(workerCtx, t.UserID)
	emailStr := ""
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error fetching user email for dispatch %s: %v", taskID, err)
	} else if userEmail.Valid {
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

		evtPayload, mErr := json.Marshal(map[string]interface{}{
			"task_id":      taskID,
			"task_name":    t.Name,
			"execution_id": executionID,
		})
		if mErr != nil {
			log.Printf("Error marshaling approval_required event for %s: %v", taskID, mErr)
		} else {
			if pErr := PublishEvent(workerCtx, PubSubEvent{
				UserID:    t.UserID,
				EventType: "approval_required",
				Payload:   string(evtPayload),
			}); pErr != nil {
				log.Printf("Error publishing approval_required event for %s: %v", taskID, pErr)
			}
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

		evtPayload, mErr := json.Marshal(map[string]interface{}{
			"id":             formatUUID(logID),
			"task_id":        taskID,
			"status":         "missed",
			"execution_time": time.Now().Format(time.RFC3339),
			"task_name":      t.Name,
			"user_email":     emailStr,
			"error_message":  "user offline",
			"execution_id":   executionID,
		})
		if mErr != nil {
			log.Printf("Error marshaling missed task event for %s: %v", taskID, mErr)
		} else {
			if pErr := PublishEvent(workerCtx, PubSubEvent{
				UserID:    t.UserID,
				EventType: "task_executed",
				Payload:   string(evtPayload),
			}); pErr != nil {
				log.Printf("Error publishing task_executed event for %s (missed): %v", taskID, pErr)
			}
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

		inputJSON, err := json.Marshal(inputMap)
		if err != nil {
			log.Printf("Error marshaling native input for %s: %v", taskID, err)
		}
		if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
			TaskID:      t.ID,
			ExecutionID: executionID,
			WorkerID:    workerID,
			StepName:    "Native Execution Started",
			InputData:   pgtype.Text{String: string(inputJSON), Valid: true},
		}); err != nil {
			log.Printf("Trace error for task %s: %v", taskID, err)
		}

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

			evtPayload, mErr := json.Marshal(map[string]interface{}{
				"id":             formatUUID(logID),
				"task_id":        taskID,
				"status":         "failure",
				"execution_time": time.Now().Format(time.RFC3339),
				"task_name":      t.Name,
				"user_email":     emailStr,
				"error_message":  err.Error(),
				"execution_id":   executionID,
			})
			if mErr != nil {
				log.Printf("Error marshaling native failure event for %s: %v", taskID, mErr)
			} else {
				if pErr := PublishEvent(workerCtx, PubSubEvent{
					UserID:    t.UserID,
					EventType: "task_executed",
					Payload:   string(evtPayload),
				}); pErr != nil {
					log.Printf("Error publishing task_executed event for %s (failure): %v", taskID, pErr)
				}
			}

			if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
				TaskID:       t.ID,
				ExecutionID:  executionID,
				WorkerID:     workerID,
				StepName:     "Native Execution Failed",
				IsError:      pgtype.Bool{Bool: true, Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}); err != nil {
				log.Printf("Trace error for task %s: %v", taskID, err)
			}

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

			evtPayload, mErr := json.Marshal(map[string]interface{}{
				"id":             formatUUID(logID),
				"task_id":        taskID,
				"status":         "success",
				"execution_time": time.Now().Format(time.RFC3339),
				"task_name":      t.Name,
				"user_email":     emailStr,
				"llm_response":   result,
				"execution_id":   executionID,
			})
			if mErr != nil {
				log.Printf("Error marshaling native success event for %s: %v", taskID, mErr)
			} else {
				if pErr := PublishEvent(workerCtx, PubSubEvent{
					UserID:    t.UserID,
					EventType: "task_executed",
					Payload:   string(evtPayload),
				}); pErr != nil {
					log.Printf("Error publishing task_executed event for %s (success): %v", taskID, pErr)
				}
			}

			if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
				TaskID:      t.ID,
				ExecutionID: executionID,
				WorkerID:    workerID,
				StepName:    "Native Execution Success",
				OutputData:  pgtype.Text{String: result, Valid: true},
			}); err != nil {
				log.Printf("Trace error for task %s: %v", taskID, err)
			}

			// Evaluate loop condition for native actions
			if len(t.LoopCondition) > 0 {
				// Fetch state for evaluation (state was updated by executeNativeJS)
				sBytes, err := queries.GetWorkflowState(workerCtx, db.GetWorkflowStateParams{
					TaskID:      t.ID,
					ExecutionID: executionID,
				})
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error fetching workflow state for native loop eval %s: %v", taskID, err)
				}
				
				var stateMap map[string]interface{}
				if len(sBytes) > 0 {
					json.Unmarshal(sBytes, &stateMap)
				}

				if evaluateWorkflowLoop(t.LoopCondition, stateMap) {
					log.Printf("Loop condition met for native task %s, triggering next iteration.", taskID)
					completeTask(workerCtx, t.UserID, taskID, time.Now().UTC(), StatusActive)
					return
				}
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
		}
		return
	}

	if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Executor Started",
		InputData:   pgtype.Text{String: t.AgentPrompt, Valid: true},
	}); err != nil {
		log.Printf("Trace error for task %s: %v", taskID, err)
	}

	resolvedPrompt := resolvePromptVariables(workerCtx, t.UserID, t.AgentPrompt, triggerPayload, state)
	if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Prompt Variables Resolved",
		OutputData:  pgtype.Text{String: maskSensitiveData(resolvedPrompt), Valid: true},
	}); err != nil {
		log.Printf("Trace error: %v", err)
	}

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

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("Error marshaling task trigger payload for %s: %v", taskID, err)
		return
	}
	subscribers, err := RedisClient.Publish(workerCtx, fmt.Sprintf("trigger_task:%s", t.UserID), string(payloadBytes)).Result()
	if err != nil || subscribers == 0 {
		observeTaskOutcome("delivery_failure")
		if err == nil {
			err = fmt.Errorf("no active subscribers received the payload")
		}
		log.Printf("Failed to deliver task %s for user %s: %v", taskID, t.UserID, err)

		if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
			TaskID:       t.ID,
			ExecutionID:  executionID,
			WorkerID:     workerID,
			StepName:     "Task Delivery Failed",
			IsError:      pgtype.Bool{Bool: true, Valid: true},
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		}); err != nil {
			log.Printf("Trace error: %v", err)
		}

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

		evtPayload, mErr := json.Marshal(map[string]interface{}{
			"id":             formatUUID(logID),
			"task_id":        taskID,
			"status":         "failure",
			"execution_time": time.Now().Format(time.RFC3339),
			"task_name":      t.Name,
			"user_email":     emailStr,
			"error_message":  err.Error(),
		})
		if mErr != nil {
			log.Printf("Error marshaling delivery failure event for %s: %v", taskID, mErr)
		} else {
			if pErr := PublishEvent(workerCtx, PubSubEvent{
				UserID:    t.UserID,
				EventType: "task_executed",
				Payload:   string(evtPayload),
			}); pErr != nil {
				log.Printf("Error publishing task_executed event for %s (failure): %v", taskID, pErr)
			}
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

	evtPayload, err := json.Marshal(map[string]interface{}{
		"id":             formatUUID(logID),
		"task_id":        taskID,
		"status":         "success",
		"execution_time": time.Now().Format(time.RFC3339),
		"task_name":      t.Name,
		"user_email":     emailStr,
		"llm_response":   "Task delivered to node via Redis",
	})
	if err != nil {
		log.Printf("Error marshaling delivery success event for %s: %v", taskID, err)
	} else {
		if err := PublishEvent(workerCtx, PubSubEvent{
			UserID:    t.UserID,
			EventType: "task_executed",
			Payload:   string(evtPayload),
		}); err != nil {
			log.Printf("Error publishing task_executed event for %s (delivered): %v", taskID, err)
		}
	}

	if _, err := queries.CreateExecutionTrace(workerCtx, db.CreateExecutionTraceParams{Metadata: nil, 
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Task Delivered",
		OutputData:  pgtype.Text{String: "Task delivered to node via Redis", Valid: true},
	}); err != nil {
		log.Printf("Trace error for task %s: %v", taskID, err)
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
		return output.String
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

func evaluateWorkflowLoop(loopConfig []byte, state map[string]interface{}) bool {
	if len(loopConfig) == 0 || state == nil {
		return false
	}
	var loopCond map[string]interface{}
	if err := json.Unmarshal(loopConfig, &loopCond); err != nil {
		log.Printf("Error unmarshaling loop config: %v", err)
		return false
	}

	enabled, ok := loopCond["enabled"].(bool)
	if !ok || !enabled {
		return false
	}

	variable, _ := loopCond["variable"].(string)
	operator, _ := loopCond["operator"].(string)
	targetValue := loopCond["value"]

	stateValue, ok := state[variable]
	if !ok {
		return false
	}

	return compareValues(stateValue, targetValue, operator)
}

// compareValues handles type-agnostic comparison for workflow conditions
func compareValues(actual interface{}, target interface{}, operator string) bool {
	sActual := fmt.Sprintf("%v", actual)
	sTarget := fmt.Sprintf("%v", target)

	switch operator {
	case "equals", "==":
		return sActual == sTarget
	case "not_equals", "!=":
		return sActual != sTarget
	case "contains":
		return strings.Contains(sActual, sTarget)
	case "greater_than", ">":
		fActual, errA := strconv.ParseFloat(sActual, 64)
		fTarget, errT := strconv.ParseFloat(sTarget, 64)
		if errA == nil && errT == nil {
			return fActual > fTarget
		}
		return sActual > sTarget
	case "less_than", "<":
		fActual, errA := strconv.ParseFloat(sActual, 64)
		fTarget, errT := strconv.ParseFloat(sTarget, 64)
		if errA == nil && errT == nil {
			return fActual < fTarget
		}
		return sActual < sTarget
	default:
		log.Printf("Unknown comparison operator: %s", operator)
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

func executeDecisionRouter(ctx context.Context, mcpServer *server.MCPServer, t db.Task, prevOutput string) {
	taskID := formatUUID(t.ID)
	executionID := fmt.Sprintf("%s-%d", taskID, time.Now().UTC().UnixNano())

	if _, err := queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{Metadata: nil, 
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Decision Router Start",
		InputData:   pgtype.Text{String: maskSensitiveData(prevOutput), Valid: true},
	}); err != nil {
		log.Printf("Trace error: %v", err)
	}

	// 1. Fetch dependent tasks
	dependents, err := queries.GetDependentTasks(ctx, t.ID)
	if err != nil {
		log.Printf("Error fetching dependent tasks for router %s: %v", taskID, err)
		return
	}

	// Build options string for the prompt
	optionsStr := ""
	for _, dept := range dependents {
		var cond map[string]string
		if err := json.Unmarshal(dept.BranchCondition, &cond); err == nil {
			if key, ok := cond["key"]; ok {
				optionsStr += fmt.Sprintf("- %s\n", key)
			}
		}
	}

	// 2. Parallel Fan-out Debate
	personas := []string{PersonaArchitect, PersonaSecurity, PersonaAdvocate}
	type debateResult struct {
		persona  string
		critique string
	}
	resultsChan := make(chan debateResult, len(personas))
	var wg sync.WaitGroup

	for _, p := range personas {
		wg.Add(1)
		go func(persona string) {
			defer wg.Done()
			// Call Sampling with specific persona prepended to prompt
			personaPrompt := fmt.Sprintf("%s\n\nTask Input: %s\nOptions:\n%s\n\nProvide your critique.", persona, prevOutput, optionsStr)
			req := mcp.CreateMessageRequest{
				CreateMessageParams: mcp.CreateMessageParams{
					Messages: []mcp.SamplingMessage{
						{Role: "user", Content: mcp.TextContent{Type: "text", Text: personaPrompt}},
					},
					MaxTokens: 300,
				},
			}
			res, err := mcpServer.RequestSampling(ctx, req)
			if err == nil {
				resultsChan <- debateResult{persona: persona, critique: extractRawText(res)}
			}
		}(p)
	}

	wg.Wait()
	close(resultsChan)

	transcript := ""
	for r := range resultsChan {
		transcript += fmt.Sprintf("### %s\n%s\n\n", r.persona, r.critique)
	}

	judgePrompt := fmt.Sprintf(`You are the Executive Judge. You have received the following critiques from three specialists regarding a workflow decision.
Critiques:
%s

Options:
%s

Final Task Output to Evaluate:
%s

Synthesize the views and pick the final branch. Respond ONLY with JSON: {"choice": "branch_key", "reasoning": "..."}`, transcript, optionsStr, prevOutput)

	req := mcp.CreateMessageRequest{
		CreateMessageParams: mcp.CreateMessageParams{
			Messages: []mcp.SamplingMessage{
				{Role: "user", Content: mcp.TextContent{Type: "text", Text: judgePrompt}},
			},
			MaxTokens: 500,
		},
	}

	res, err := mcpServer.RequestSampling(ctx, req)
	if err != nil {
		log.Printf("Decision router LLM call failed for %s: %v", taskID, err)
		queries.UpdateTaskApprovalStatus(ctx, db.UpdateTaskApprovalStatusParams{
			LastApprovalStatus: pgtype.Text{String: ApprovalStatusNeedsRouting, Valid: true},
			Status:             pgtype.Text{String: StatusHalted, Valid: true},
			ID:                 t.ID,
			UserID:             t.UserID,
		})
		if _, tErr := queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{Metadata: nil, 
			TaskID:       t.ID,
			ExecutionID:  executionID,
			WorkerID:     workerID,
			StepName:     "Decision Router LLM Failed",
			IsError:      pgtype.Bool{Bool: true, Valid: true},
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		}); tErr != nil {
			log.Printf("Trace error: %v", tErr)
		}

		return
	}

	// Extract choice
	choice := parseLLMChoice(res)

	if choice == "" {
		log.Printf("Decision router %s failed to get a choice from LLM response", taskID)
		queries.UpdateTaskApprovalStatus(ctx, db.UpdateTaskApprovalStatusParams{
			LastApprovalStatus: pgtype.Text{String: ApprovalStatusNeedsRouting, Valid: true},
			Status:             pgtype.Text{String: StatusHalted, Valid: true},
			ID:                 t.ID,
			UserID:             t.UserID,
		})
		if _, err := queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{Metadata: nil, 
			TaskID:       t.ID,
			ExecutionID:  executionID,
			WorkerID:     workerID,
			StepName:     "Decision Router No Choice",
			ErrorMessage: pgtype.Text{String: "LLM did not provide a valid choice JSON", Valid: true},
			IsError:      pgtype.Bool{Bool: true, Valid: true},
		}); err != nil {
			log.Printf("Trace error: %v", err)
		}
		return
	}

	transcriptJSON, err := json.Marshal(map[string]string{"transcript": maskSensitiveData(transcript)})
	if err != nil {
		log.Printf("Warning: failed to marshal debate transcript: %v", err)
	}
	if _, err := queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Swarm Debate Conclusion",
		OutputData:  pgtype.Text{String: fmt.Sprintf("Choice: %s", choice), Valid: true},
		Metadata:    transcriptJSON,
	}); err != nil {
		log.Printf("Trace error: %v", err)
	}

	// 3. Match choice and activate task
	found := false
	for _, dept := range dependents {
		var cond map[string]string
		if len(dept.BranchCondition) == 0 {
			continue
		}
		if err := json.Unmarshal(dept.BranchCondition, &cond); err != nil {
			log.Printf("Warning: failed to unmarshal branch condition for task %s: %v", formatUUID(dept.ID), err)
			continue
		}
		if cond["key"] == choice {
			queries.UpdateTaskNextRun(ctx, db.UpdateTaskNextRunParams{
				Status:  pgtype.Text{String: StatusActive, Valid: true},
				NextRun: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				ID:      dept.ID,
			})
			log.Printf("Decision router %s activated task %s (choice: %s)", taskID, formatUUID(dept.ID), choice)
			found = true
			break
		}
	}

	if !found {
		log.Printf("Decision router %s choice '%s' matched no dependent tasks", taskID, choice)
		queries.UpdateTaskApprovalStatusAndLastRun(ctx, db.UpdateTaskApprovalStatusAndLastRunParams{
			LastApprovalStatus: pgtype.Text{String: ApprovalStatusNeedsRouting, Valid: true},
			Status:             pgtype.Text{String: StatusHalted, Valid: true},
			ID:                 t.ID,
			UserID:             t.UserID,
		})
	} else {
		queries.UpdateTaskStatusAndLastRun(ctx, db.UpdateTaskStatusAndLastRunParams{
			Status: pgtype.Text{String: StatusCompleted, Valid: true},
			ID:     t.ID,
			UserID: t.UserID,
		})
	}
}

// extractRawText gets the text content from an LLM sampling result.
func extractRawText(res interface{}) string {
	if res == nil {
		return ""
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		return ""
	}
	var resMap map[string]interface{}
	if err := json.Unmarshal(resBytes, &resMap); err != nil {
		return ""
	}

	if content, ok := resMap["content"].(map[string]interface{}); ok {
		if text, ok := content["text"].(string); ok {
			return text
		}
	} else if contentSlice, ok := resMap["content"].([]interface{}); ok && len(contentSlice) > 0 {
		if first, ok := contentSlice[0].(map[string]interface{}); ok {
			if text, ok := first["text"].(string); ok {
				return text
			}
		}
	}

	return ""
}

// parseLLMChoice extracts the "choice" field from an LLM sampling result.
func parseLLMChoice(res interface{}) string {
	responseText := extractRawText(res)
	if responseText == "" {
		return ""
	}

	var respObj struct {
		Choice string `json:"choice"`
	}
	// Basic JSON extraction
	if err := json.Unmarshal([]byte(responseText), &respObj); err == nil {
		return respObj.Choice
	}

	// Try fuzzy matching if JSON is wrapped in markdown
	re := regexp.MustCompile(`\{.*"choice".*\}`)
	match := re.FindString(responseText)
	if match != "" {
		if err := json.Unmarshal([]byte(match), &respObj); err == nil {
			return respObj.Choice
		}
	}

	return ""
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
	parentOutputBytes, err := queries.GetTaskOutput(ctx, db.GetTaskOutputParams{
		TaskID: tid,
		UserID: userID,
	})
	parentOutput := ""
	if err != nil {
		log.Printf("Error fetching output for task %s in completeTask: %v", taskID, err)
	} else {
		parentOutput = parentOutputBytes.String
	}

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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in runReaper: %v", r)
		}
	}()
	reapTicker := time.NewTicker(1 * time.Minute)
	pruneTicker := time.NewTicker(1 * time.Hour)
	defer reapTicker.Stop()
	defer pruneTicker.Stop()

	for {
		select {
		case <-reapTicker.C:
			threshold := time.Now().Add(-CurrentSystemSettings.GetReaperThreshold())
			rows, err := queries.ReapStuckTasks(ctx, pgtype.Timestamp{Time: threshold, Valid: true})
			if err != nil {
				log.Printf("Reaper: error reaping stuck tasks: %v", err)
			} else if rows > 0 {
				log.Printf("Reaper: recovered %d stuck tasks", rows)
			}
		case <-pruneTicker.C:
			// Prune zombie workers
			pruneDays, err := queries.GetSystemSettings(ctx)
			if err != nil {
				log.Printf("Reaper: error fetching prune days: %v", err)
			} else if pruneDays > 0 {
				if err := queries.PruneZombieWorkers(ctx, pruneDays); err != nil {
					log.Printf("Reaper: error pruning zombie workers: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func runWorkerHeartbeat(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in runWorkerHeartbeat: %v", r)
		}
	}()
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

type SwarmAgent struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type SwarmConfig struct {
	ConsensusMode    string       `json:"consensus_mode"` // "voting" or "supervisor"
	SupervisorPrompt string       `json:"supervisor_prompt"`
	Council          []SwarmAgent `json:"council"`
}

func executeSwarmRouter(ctx context.Context, mcpServer *server.MCPServer, t db.Task, prevOutput string) {
	taskID := formatUUID(t.ID)
	executionID := fmt.Sprintf("%s-%d", taskID, time.Now().UTC().UnixNano())

	if _, err := queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{Metadata: nil, 
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Swarm Router Start",
		InputData:   pgtype.Text{String: maskSensitiveData(prevOutput), Valid: true},
	}); err != nil {
		log.Printf("Trace error: %v", err)
	}

	var swarmCfg SwarmConfig
	if err := json.Unmarshal(t.SwarmConfig, &swarmCfg); err != nil {
		log.Printf("Failed to parse swarm config for task %s: %v", taskID, err)
		return
	}

	if len(swarmCfg.Council) == 0 {
		log.Printf("Swarm council is empty for task %s", taskID)
		queries.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
			Status: pgtype.Text{String: ApprovalStatusNeedsRouting, Valid: true},
			ID:     t.ID,
			UserID: t.UserID,
		})
		return
	}

	dependents, err := queries.GetDependentTasks(ctx, t.ID)
	if err != nil {
		log.Printf("Failed to get dependent tasks for %s: %v", taskID, err)
		return
	}

	// Build options string for the prompt
	optionsStr := ""
	for _, dept := range dependents {
		var cond map[string]string
		if err := json.Unmarshal(dept.BranchCondition, &cond); err == nil {
			if key, ok := cond["key"]; ok {
				optionsStr += fmt.Sprintf("- %s\n", key)
			} else {
				optionsStr += "- (missing key in branch condition)\n"
			}
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	agentResponses := make(map[string]string)

	for _, agent := range swarmCfg.Council {
		wg.Add(1)
		go func(a SwarmAgent) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in swarm agent %s: %v", a.Name, r)
				}
			}()

			agentPrompt := ""
			if swarmCfg.ConsensusMode == "voting" {
				agentPrompt = fmt.Sprintf("You are %s. %s\n\nAnalyze this input:\n%s\n\nAvailable options:\n%s\nOutput JSON with a single key 'choice' containing your selected option.", a.Name, a.Prompt, prevOutput, optionsStr)
			} else {
				agentPrompt = fmt.Sprintf("You are %s. %s\n\nProvide a detailed analysis of this input:\n%s\n\nAvailable options are:\n%s\nState which option you recommend and why.", a.Name, a.Prompt, prevOutput, optionsStr)
			}

			// Request Sampling
			sampleCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			req := mcp.CreateMessageRequest{
				CreateMessageParams: mcp.CreateMessageParams{
					Messages: []mcp.SamplingMessage{
						{Role: "user", Content: mcp.TextContent{Type: "text", Text: agentPrompt}},
					},
					MaxTokens: 500,
				},
			}

			res, err := mcpServer.RequestSampling(sampleCtx, req)
			if err != nil {
				log.Printf("Agent %s sampling request failed: %v", a.Name, err)
				return
			}

			// Extract raw text for transcript/traces
			rawText := extractRawText(res)

			if swarmCfg.ConsensusMode == "voting" {
				choice := parseLLMChoice(res)
				if choice != "" {
					mu.Lock()
					agentResponses[a.Name] = choice
					mu.Unlock()
				}
			} else {
				mu.Lock()
				agentResponses[a.Name] = rawText
				mu.Unlock()
			}

			queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{Metadata: nil, 
				TaskID:      t.ID,
				ExecutionID: executionID,
				WorkerID:    workerID,
				StepName:    fmt.Sprintf("Swarm Agent: %s", a.Name),
				OutputData:  pgtype.Text{String: rawText, Valid: true},
			})
		}(agent)
	}

	wg.Wait()

	// Consensus Resolution
	finalChoice := ""
	consensusDetails := ""
	
	if swarmCfg.ConsensusMode == "voting" {
		counts := make(map[string]int)
		maxCount := 0
		for _, c := range agentResponses {
			counts[c]++
			if counts[c] > maxCount {
				maxCount = counts[c]
				finalChoice = c
			}
		}
		
		tallyStr := "Vote Tally: "
		first := true
		for opt, count := range counts {
			if !first {
				tallyStr += ", "
			}
			tallyStr += fmt.Sprintf("%s=%d", opt, count)
			first = false
		}
		consensusDetails = tallyStr

		tieCount := 0
		for _, c := range counts {
			if c == maxCount {
				tieCount++
			}
		}
		if tieCount > 1 {
			finalChoice = "" // Tie, force fallback
			consensusDetails += " (Tie Detected)"
		}
	} else if swarmCfg.ConsensusMode == "supervisor" {
		// Construct transcript
		transcript := "Debate Transcript:\n"
		for name, response := range agentResponses {
			transcript += fmt.Sprintf("Agent %s analysis:\n%s\n---\n", name, response)
		}

		supervisorPrompt := fmt.Sprintf("%s\n\n%s\n\nAvailable options:\n%s\nOutput JSON with a single key 'choice' containing your selected option.", swarmCfg.SupervisorPrompt, transcript, optionsStr)

		// Run supervisor LLM call
		sampleCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		req := mcp.CreateMessageRequest{
			CreateMessageParams: mcp.CreateMessageParams{
				Messages: []mcp.SamplingMessage{
					{Role: "user", Content: mcp.TextContent{Type: "text", Text: supervisorPrompt}},
				},
				MaxTokens: 500,
			},
		}
		res, err := mcpServer.RequestSampling(sampleCtx, req)
		if err == nil && res != nil {
			finalChoice = parseLLMChoice(res)
			consensusDetails = fmt.Sprintf("Supervisor chose: %s", finalChoice)
		} else {
			consensusDetails = "Supervisor failed to respond"
		}
	}

	consensusMetadata, _ := json.Marshal(map[string]string{"details": consensusDetails})
	queries.CreateExecutionTrace(ctx, db.CreateExecutionTraceParams{
		TaskID:      t.ID,
		ExecutionID: executionID,
		WorkerID:    workerID,
		StepName:    "Swarm Consensus Reached",
		OutputData:  pgtype.Text{String: consensusDetails, Valid: true},
		Metadata:    consensusMetadata,
	})

	// Match choice and activate task
	found := false
	if finalChoice != "" {
		for _, dept := range dependents {
			var cond map[string]string
			if len(dept.BranchCondition) == 0 {
				continue
			}
			if err := json.Unmarshal(dept.BranchCondition, &cond); err == nil {
				if cond["key"] == finalChoice {
					queries.UpdateTaskNextRun(ctx, db.UpdateTaskNextRunParams{
						Status:  pgtype.Text{String: StatusActive, Valid: true},
						NextRun: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
						ID:      dept.ID,
					})
					found = true
					break
				}
			}
		}
	}

	if !found {
		queries.UpdateTaskApprovalStatusAndLastRun(ctx, db.UpdateTaskApprovalStatusAndLastRunParams{
			LastApprovalStatus: pgtype.Text{String: ApprovalStatusNeedsRouting, Valid: true},
			Status:             pgtype.Text{String: StatusHalted, Valid: true},
			ID:                 t.ID,
			UserID:             t.UserID,
		})
	} else {
		queries.UpdateTaskStatusAndLastRun(ctx, db.UpdateTaskStatusAndLastRunParams{
			Status: pgtype.Text{String: StatusCompleted, Valid: true},
			ID:     t.ID,
			UserID: t.UserID,
		})
	}
}
