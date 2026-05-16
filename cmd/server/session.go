package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"schedule-mcp/db"
)

// GlobalSessionManager tracks which users have active SSE connections via Redis
var GlobalSessionManager = &SessionManager{}

type SessionManager struct {
	redisClient *redis.Client
}

func (sm *SessionManager) Init(client *redis.Client) {
	sm.redisClient = client
}

// AddUser sets a heartbeat in Redis that expires after 30 seconds
func (sm *SessionManager) AddUser(ctx context.Context, userID string) {
	if sm.redisClient == nil {
		return
	}
	ctx, span := otel.Tracer("session").Start(ctx, "AddUser")
	defer span.End()

	err := sm.redisClient.Set(ctx, fmt.Sprintf("session:%s", userID), "active", 30*time.Second).Err()
	if err != nil {
		log.Printf("Failed to set session for user %s: %v", userID, err)
		span.RecordError(err)
	}
}

// RemoveUser removes the heartbeat from Redis
func (sm *SessionManager) RemoveUser(ctx context.Context, userID string) {
	if sm.redisClient == nil {
		return
	}
	ctx, span := otel.Tracer("session").Start(ctx, "RemoveUser")
	defer span.End()

	sm.redisClient.Del(ctx, fmt.Sprintf("session:%s", userID))
}

// IsOnline checks if a user has an active heartbeat in Redis
func (sm *SessionManager) IsOnline(ctx context.Context, userID string) bool {
	if sm.redisClient == nil {
		return false
	}
	ctx, span := otel.Tracer("session").Start(ctx, "IsOnline")
	defer span.End()

	val, err := sm.redisClient.Get(ctx, fmt.Sprintf("session:%s", userID)).Result()
	if err == redis.Nil {
		return false
	} else if err != nil {
		log.Printf("Failed to check session for user %s: %v", userID, err)
		span.RecordError(err)
		return false
	}
	return val == "active"
}

// Heartbeat Loop - Keeps the session active in Redis while the SSE connection is open
// Also subscribes to Pub/Sub to listen for remote task triggers
func (sm *SessionManager) MaintainHeartbeat(ctx context.Context, userID string, mcpServer *server.MCPServer) {
	// Check per-user connection limit (max 5 connections)
	connCountKey := fmt.Sprintf("conn_count:%s", userID)
	count, _ := sm.redisClient.Incr(ctx, connCountKey).Result()
	sm.redisClient.Expire(ctx, connCountKey, 1*time.Minute)
	
	defer func() {
		sm.redisClient.Decr(context.Background(), connCountKey)
	}()

	if count > 10 {
		log.Printf("User %s exceeded connection limit (%d). Rejecting SSE.", userID, count)
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	activeSSEConnections.Inc()
	defer activeSSEConnections.Dec()

	sm.AddUser(ctx, userID)

	var backoff time.Duration = 1 * time.Second
	for {
		// Subscribe to tasks meant for this user
		pubsub := sm.redisClient.Subscribe(ctx, fmt.Sprintf("trigger_task:%s", userID))

		// Wait for subscription confirmation
		_, err := pubsub.Receive(ctx)
		if err != nil {
			log.Printf("Failed to subscribe to Redis for user %s: %v. Retrying in %v...", userID, err, backoff)
			pubsub.Close()
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
		}

		// Reset backoff on successful subscription
		backoff = 1 * time.Second

		ch := pubsub.Channel()

		// Inner loop for processing messages
		func() {
			defer pubsub.Close()
			for {
				select {
				case <-ctx.Done():
					// The HTTP request was cancelled (connection closed)
					sm.RemoveUser(context.Background(), userID)
					return
				case <-ticker.C:
					sm.AddUser(ctx, userID)
					// Refresh connection count expiry
					sm.redisClient.Expire(ctx, connCountKey, 1*time.Minute)
				case msg, ok := <-ch:
					if !ok {
						log.Printf("Redis channel closed for user %s. Re-subscribing...", userID)
						return
					}
					// Received a task trigger from another node
					log.Printf("Received Pub/Sub task trigger for user %s", userID)

					// Fire the sampling request asynchronously
					workerWG.Add(1)
					go func(payload string) {
						defer workerWG.Done()
						executionStart := time.Now()

						var taskData map[string]interface{}
						if err := json.Unmarshal([]byte(payload), &taskData); err != nil {
							log.Printf("Failed to unmarshal pubsub payload: %v", err)
							return
						}

						// Extract trace context
						traceMap, _ := taskData["trace_context"].(map[string]interface{})
						carrier := propagation.MapCarrier{}
						for k, v := range traceMap {
							if s, ok := v.(string); ok {
								carrier[k] = s
							}
						}
						parentCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
						ctx, span := otel.Tracer("scheduler-mcp").Start(parentCtx, "Redis Task Trigger")
						defer span.End()

						taskID, _ := taskData["task_id"].(string)
						prompt, _ := taskData["prompt"].(string)
						executionID, _ := taskData["execution_id"].(string)
						
						span.SetAttributes(
							attribute.String("task_id", taskID),
							attribute.String("execution_id", executionID),
							attribute.String("user_id", userID),
						)
						triggerType, _ := taskData["trigger_type"].(string)
						triggerConfigStr, _ := taskData["trigger_config"].(string)
						triggerPayload, _ := taskData["trigger_payload"].(map[string]interface{})

						if taskID == "" || prompt == "" || executionID == "" || triggerType == "" || triggerConfigStr == "" {
							err := fmt.Errorf("incomplete Pub/Sub payload")
							log.Printf("Incomplete Pub/Sub payload for user %s: %+v", userID, taskData)
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return
						}

						var tid pgtype.UUID
						if err := parseUUID(taskID, &tid); err != nil {
							log.Printf("Invalid task ID received via Pub/Sub for user %s: %s", userID, taskID)
							span.RecordError(err)
							span.SetStatus(codes.Error, "invalid task uuid")
							return
						}

						// Keep DB operations alive across prompt resolution, sampling, and status updates.
						dbCtx, dbCancel := context.WithTimeout(ctx, 45*time.Second)
						defer dbCancel()

						t, err := queries.GetTaskByID(dbCtx, db.GetTaskByIDParams{
							ID:     tid,
							UserID: userID,
						})
						if err != nil {
							log.Printf("Failed to fetch task %s: %v", taskID, err)
							return
						}

						userEmail, err := queries.GetUserEmail(dbCtx, userID)
						emailStr := ""
						if err != nil {
							log.Printf("Error fetching user email for %s: %v", userID, err)
						} else if userEmail.Valid {
							emailStr = userEmail.String
						}

						// 2. Resolve Prompt (Secrets + Chaining + Webhook Body)
						queries.CreateExecutionTrace(dbCtx, db.CreateExecutionTraceParams{
							TaskID:      tid,
							ExecutionID: executionID,
							WorkerID:    workerID,
							StepName:    "Prompt Resolution",
							InputData:   []byte(prompt),
						})
						finalPrompt, secretCount, chained, err := resolvePrompt(dbCtx, userID, tid, executionID, prompt, t.DependsOnTaskID, triggerPayload)
						if err != nil {
							log.Printf("Prompt resolution failed for task %s: %v", taskID, err)
							queries.CreateExecutionTrace(dbCtx, db.CreateExecutionTraceParams{
								TaskID:       tid,
								ExecutionID:  executionID,
								WorkerID:     workerID,
								StepName:     "Prompt Resolution Failed",
								IsError:      pgtype.Bool{Bool: true, Valid: true},
								ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
							})
						} else {
							queries.CreateExecutionTrace(dbCtx, db.CreateExecutionTraceParams{
								TaskID:      tid,
								ExecutionID: executionID,
								WorkerID:    workerID,
								StepName:    "Prompt Resolution Success",
								OutputData:  []byte(finalPrompt),
							})
						}

						sampleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						defer cancel()

						// Phase 10.1: Prevent Double Execution if user is connected from multiple terminals
						locked, err := sm.redisClient.SetNX(sampleCtx, fmt.Sprintf("lock:exec:%s", executionID), "locked", 5*time.Minute).Result()
						if err != nil || !locked {
							log.Printf("Task %s already executed by another connection for user %s", taskID, userID)
							return
						}

						queries.CreateExecutionTrace(dbCtx, db.CreateExecutionTraceParams{
							TaskID:      tid,
							ExecutionID: executionID,
							WorkerID:    workerID,
							StepName:    "LLM Sampling",
							InputData:   []byte(finalPrompt),
						})

						req := mcp.CreateMessageRequest{
							CreateMessageParams: mcp.CreateMessageParams{
								Messages: []mcp.SamplingMessage{
									{Role: "user", Content: mcp.TextContent{Type: "text", Text: finalPrompt}},
								},
								MaxTokens: 1000,
							},
						}

						// Phase 7.2: Real LLM Response Handling
						res, err := mcpServer.RequestSampling(sampleCtx, req)

						if err != nil {
							observeTaskOutcome("execution_failure")
							observeTaskExecutionDuration(executionStart, "failure")
							log.Printf("Pub/Sub Sampling failed for user %s: %v", userID, err)

							queries.CreateExecutionTrace(dbCtx, db.CreateExecutionTraceParams{
								TaskID:       tid,
								ExecutionID:  executionID,
								WorkerID:     workerID,
								StepName:     "LLM Sampling Failed",
								IsError:      pgtype.Bool{Bool: true, Valid: true},
								ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
							})

							// Phase 10.2: Properly log failure back to DB instead of failing silently
							logID, logErr := queries.CreateTaskLog(dbCtx, db.CreateTaskLogParams{
								TaskID:       tid,
								UserID:       userID,
								Status:       "failure",
								ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
							})
							if logErr != nil {
								log.Printf("Error creating failure log for task %s: %v", taskID, logErr)
							}

							// Emit Redis event
							evtPayload, _ := json.Marshal(map[string]interface{}{
								"id":               formatUUID(logID),
								"task_id":          taskID,
								"status":           "failure",
								"execution_time":   time.Now().Format(time.RFC3339),
								"task_name":        t.Name,
								"user_email":       emailStr,
								"error_message":    err.Error(),
								"secrets_injected": secretCount,
								"chained":          chained,
							})
							if err := PublishEvent(dbCtx, PubSubEvent{
								UserID:    userID,
								EventType: "task_executed",
								Payload:   string(evtPayload),
							}); err != nil {
								log.Printf("Error publishing task_executed (failure) for %s: %v", taskID, err)
							}

							// Increment failure count securely
							currentFailures, err := queries.IncrementTaskFailureCount(dbCtx, db.IncrementTaskFailureCountParams{
								ID:     tid,
								UserID: userID,
							})
							if err != nil {
								log.Printf("Error incrementing failure count for task %s: %v", taskID, err)
							}

							if currentFailures.Int32 >= 3 {
								if err := queries.UpdateTaskStatus(dbCtx, db.UpdateTaskStatusParams{
									Status: pgtype.Text{String: StatusError, Valid: true},
									ID:     tid,
									UserID: userID,
								}); err != nil {
									log.Printf("Error updating status to error for task %s: %v", taskID, err)
								}
								sendFailureEmail(dbCtx, userID, taskID, t.Name)
							} else {
								// Unlock so it can be retried by the scheduler
								if err := queries.UpdateTaskStatus(dbCtx, db.UpdateTaskStatusParams{
									Status: pgtype.Text{String: StatusActive, Valid: true},
									ID:     tid,
									UserID: userID,
								}); err != nil {
									log.Printf("Error updating status to active for task %s: %v", taskID, err)
								}
							}
							return
						}

						// Safely extract the LLM's text response
						llmResponse := "No response provided by LLM"
						if res != nil {
							// Convert response to JSON string for the log
							resBytes, _ := json.Marshal(res)
							llmResponse = string(resBytes)
						}
						llmResponse = sanitizeLLMResponseForStorage(llmResponse)

						// Phase 12.2: Handle State Updates if response is JSON and contains state_update
						var respObj map[string]interface{}
						if err := json.Unmarshal([]byte(llmResponse), &respObj); err == nil {
							if stateUpdate, ok := respObj["state_update"].(map[string]interface{}); ok {
								// Fetch current state
								currentState := make(map[string]interface{})
								sBytes, _ := queries.GetWorkflowState(dbCtx, db.GetWorkflowStateParams{
									TaskID:      tid,
									ExecutionID: executionID,
								})
								if len(sBytes) > 0 {
									json.Unmarshal(sBytes, &currentState)
								}
								// Merge
								for k, v := range stateUpdate {
									currentState[k] = v
								}
								newStateBytes, _ := json.Marshal(currentState)
								queries.UpsertWorkflowState(dbCtx, db.UpsertWorkflowStateParams{
									TaskID:      tid,
									ExecutionID: executionID,
									StateData:   newStateBytes,
								})
								log.Printf("Updated workflow state for task %s, execution %s", taskID, executionID)
							}
						}

						log.Printf("Received LLM Response for user %s: %s", userID, llmResponse)
						observeTaskOutcome("execution_success")
						observeTaskExecutionDuration(executionStart, "success")

						// Save the actual LLM response to the specific task log
						logID, err := queries.CreateTaskLog(dbCtx, db.CreateTaskLogParams{
							TaskID:      tid,
							UserID:      userID,
							Status:      "success",
							LlmResponse: pgtype.Text{String: llmResponse, Valid: true},
						})
						if err != nil {
							log.Printf("Error creating success log for task %s: %v", taskID, err)
						}

						// Emit Redis event
						evtPayload, _ := json.Marshal(map[string]interface{}{
							"id":               formatUUID(logID),
							"task_id":          taskID,
							"status":           "success",
							"execution_time":   time.Now().Format(time.RFC3339),
							"task_name":        t.Name,
							"user_email":       emailStr,
							"llm_response":     llmResponse,
							"secrets_injected": secretCount,
							"chained":          chained,
						})
						if err := PublishEvent(dbCtx, PubSubEvent{
							UserID:    userID,
							EventType: "task_executed",
							Payload:   string(evtPayload),
						}); err != nil {
							log.Printf("Error publishing task_executed (success) for %s: %v", taskID, err)
						}

						// Phase 12.3: Evaluate loop condition
						if len(t.LoopCondition) > 0 {
							// Fetch state for evaluation
							sBytes, _ := queries.GetWorkflowState(dbCtx, db.GetWorkflowStateParams{
								TaskID:      tid,
								ExecutionID: executionID,
							})
							var stateMap map[string]interface{}
							json.Unmarshal(sBytes, &stateMap)

							if evaluateWorkflowLoop(t.LoopCondition, stateMap) {
								log.Printf("Loop condition met for task %s, triggering next iteration.", taskID)
								// Trigger immediate re-run by setting next_run to now and status to active
								completeTask(dbCtx, userID, taskID, time.Now().UTC(), StatusActive)
								return
							}
						}

						// Iteration 2: Advance the task status
						if triggerType == TriggerDate {
							completeTask(dbCtx, userID, taskID, time.Now().UTC(), StatusCompleted)
							return
						}

						var config map[string]interface{}
						if err := json.Unmarshal([]byte(triggerConfigStr), &config); err != nil {
							log.Printf("Error unmarshaling trigger config for task %s: %v", taskID, err)
							if err := queries.UpdateTaskStatus(dbCtx, db.UpdateTaskStatusParams{
								Status: pgtype.Text{String: StatusPaused, Valid: true},
								ID:     tid,
								UserID: userID,
							}); err != nil {
								log.Printf("Error pausing task %s: %v", taskID, err)
							}
							return
						}

						newNextRun, calcErr := calculateNextRun(triggerType, config, time.Now().UTC())
						if calcErr != nil {
							log.Printf("Error calculating next run for task %s: %v", taskID, calcErr)
							if err := queries.UpdateTaskStatus(dbCtx, db.UpdateTaskStatusParams{
								Status: pgtype.Text{String: StatusPaused, Valid: true},
								ID:     tid,
								UserID: userID,
							}); err != nil {
								log.Printf("Error pausing task %s: %v", taskID, err)
							}
							return
						}

						completeTask(dbCtx, userID, taskID, newNextRun)
					}(msg.Payload)
				}
			}
		}()

		// Check if we exited the inner loop because of ctx.Done()
		select {
		case <-ctx.Done():
			return
		default:
			// Continue to outer loop to re-subscribe
		}
	}
}
