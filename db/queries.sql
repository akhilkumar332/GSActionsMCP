-- name: GetUserBySessionID :one
SELECT u.id, u.email, u.api_key, u.role, u.tier, u.created_at 
FROM web_sessions s 
JOIN users u ON s.user_id = u.id 
WHERE s.id = $1 AND s.expires_at > $2;

-- name: DeleteWebSession :exec
DELETE FROM web_sessions WHERE id = $1;

-- name: CountUserTasks :one
SELECT COUNT(*) FROM tasks WHERE user_id = $1;

-- name: GetTaskLogs :many
SELECT l.id, l.task_id, l.user_id, l.execution_time, l.status, l.llm_response, l.error_message, t.name as task_name, u.email as user_email
FROM task_logs l
JOIN tasks t ON l.task_id = t.id
JOIN users u ON l.user_id = u.id
ORDER BY l.execution_time DESC
LIMIT 100;

-- name: ListUsers :many
SELECT id, email, api_key, role, tier, created_at 
FROM users 
WHERE email ILIKE $1 OR role ILIKE $1 OR tier ILIKE $1
ORDER BY created_at DESC;

-- name: GetUser :one
SELECT id, email, api_key, role, tier, created_at FROM users WHERE id = $1;

-- name: UpdateUserRole :exec
UPDATE users SET role = $1 WHERE id = $2;

-- name: UpdateUserTier :exec
UPDATE users SET tier = $1 WHERE id = $2;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, api_key) 
VALUES ($1, $2, $3) 
RETURNING id, email, api_key, role, tier, created_at;

-- name: GetAuthInfoByEmail :one
SELECT id, password_hash FROM users WHERE email = $1;

-- name: CreateWebSession :one
INSERT INTO web_sessions (user_id, expires_at) 
VALUES ($1, $2) 
RETURNING id;

-- name: UpdateUserAPIKey :exec
UPDATE users SET api_key = $1 WHERE id = $2;

-- name: GetUserEmail :one
SELECT email FROM users WHERE id = $1;

-- name: RevertProcessingTasks :exec
UPDATE tasks SET status = 'active', locked_by = NULL WHERE locked_by = $1;

-- name: GetUserByAPIKey :one
SELECT id, tier FROM users WHERE api_key = $1;

-- name: CreateTaskLog :one
INSERT INTO task_logs (task_id, user_id, status, error_message, llm_response) 
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, metadata)
VALUES ($1, $2, $3, $4, $5);

-- name: SetUserRoleByEmail :exec
UPDATE users SET role = $1 WHERE email = $2;

-- name: ExportUserTasks :many
SELECT * FROM tasks WHERE user_id = $1;

-- name: GetDependentTasks :many
SELECT * FROM tasks WHERE depends_on_task_id = $1;

-- name: LinkTaskDependency :exec
UPDATE tasks
SET depends_on_task_id = $1, trigger_on_completion = $2
WHERE id = $3 AND user_id = $4;

-- name: UpdateTaskNextRun :exec
UPDATE tasks SET status = $1, locked_by = NULL, next_run = $2 WHERE id = $3;

-- name: GetLatestTaskLogResponse :one
SELECT l.llm_response FROM task_logs l
INNER JOIN tasks t ON l.task_id = t.id
WHERE l.task_id = $1 AND t.user_id = $2
ORDER BY l.execution_time DESC 
LIMIT 1;

-- name: IncrementTaskFailureCount :one
UPDATE tasks SET failure_count = failure_count + 1 
WHERE id = $1 AND user_id = $2
RETURNING failure_count;

-- name: UpdateTaskStatus :exec
UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2 AND user_id = $3;

-- name: UpdateTaskStatusAndFailureCount :exec
UPDATE tasks SET status = $1, locked_by = NULL, failure_count = $2, retry_count = $3 WHERE id = $4 AND user_id = $5;

-- name: ClaimTaskByID :one
UPDATE tasks
SET status = 'processing',
    locked_by = $1,
    locked_at = NOW()
WHERE id = $2 AND status IN ('active', 'paused')
RETURNING *;

-- name: UpdateTaskRetryCount :exec
UPDATE tasks SET retry_count = $1 WHERE id = $2 AND user_id = $3;

-- name: ClaimDueTasks :many
SELECT * FROM fn_claim_due_tasks($1, $2);

-- name: CompleteTask :exec
SELECT fn_complete_task($1, $2, $3);

-- name: ReapStuckTasks :execrows
UPDATE tasks
SET status = 'active', locked_by = NULL
WHERE status = 'processing' AND next_run < $1;
-- name: CheckTaskOwnership :one
SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2);

-- name: CreateTask :one
INSERT INTO tasks (user_id, name, trigger_type, trigger_config, agent_prompt, missed_task_policy, depends_on_task_id, next_run, requires_approval, encrypted_secrets, trigger_on_completion, workspace_id, task_type, native_code, branch_condition, is_bundle_root, loop_condition, swarm_config) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) 
RETURNING *;

-- name: ListUserTasks :many
SELECT 
    t.*,
    (SELECT COUNT(*) FROM task_versions tv WHERE tv.task_id = t.id) as version_count
FROM tasks t
WHERE t.user_id = $1
ORDER BY t.created_at DESC;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = $1 AND user_id = $2;

-- name: GetDispatchableTaskByID :one
SELECT * FROM tasks
WHERE id = $1
  AND user_id = $2
  AND status = 'processing'
  AND locked_by = $3;

-- name: GetDependentTasksToTrigger :many
SELECT t.* FROM tasks t
INNER JOIN tasks parent ON t.depends_on_task_id = parent.id
WHERE t.depends_on_task_id = $1 
  AND t.trigger_on_completion = TRUE 
  AND t.status = 'active'
  AND t.user_id = parent.user_id;

-- name: UpdateTaskApprovalStatus :exec
UPDATE tasks SET last_approval_status = $1, status = $2, locked_by = NULL WHERE id = $3 AND user_id = $4;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1 AND user_id = $2;

-- name: UpdateTaskStatusAndLastRun :exec
UPDATE tasks SET status = $1, locked_by = NULL, last_run = NOW() WHERE id = $2 AND user_id = $3;

-- name: UpdateTaskApprovalStatusAndLastRun :exec
UPDATE tasks SET last_approval_status = $1, status = $2, locked_by = NULL, last_run = NOW() WHERE id = $3 AND user_id = $4;

-- name: ResetTaskFailureCount :exec
UPDATE tasks SET status = $1, failure_count = 0 WHERE id = $2 AND user_id = $3;

-- name: UpdateTaskStatusByUserID :exec
UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2 AND user_id = $3;

-- name: UpsertUserSecret :one
INSERT INTO user_secrets (user_id, name, encrypted_value)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, name) DO UPDATE SET encrypted_value = $3
RETURNING id;

-- name: GetUserSecret :one
SELECT encrypted_value FROM user_secrets WHERE user_id = $1 AND name = $2;

-- name: ListUserSecrets :many
SELECT id, name, created_at FROM user_secrets WHERE user_id = $1 ORDER BY name ASC;

-- name: DeleteUserSecret :exec
DELETE FROM user_secrets WHERE user_id = $1 AND name = $2;

-- name: GetSEOSettings :one
SELECT * FROM seo_settings WHERE id = 1;

-- name: UpdateSEOSettings :exec
UPDATE seo_settings 
SET title = $1, description = $2, keywords = $3, og_image = $4, updated_at = NOW()
WHERE id = 1;

-- name: UpdateTaskAgentPromptAndPolicy :one
UPDATE tasks
SET agent_prompt = $1, 
    missed_task_policy = $2,
    ui_coordinates = $3,
    depends_on_task_id = $4,
    trigger_on_completion = $5,
    branch_condition = $6,
    loop_condition = $7,
    swarm_config = $8
WHERE id = $9 AND user_id = $10
RETURNING *;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, owner_id) VALUES ($1, $2) RETURNING *;

-- name: GetUserWorkspaces :many
SELECT w.* FROM workspaces w 
LEFT JOIN workspace_members wm ON w.id = wm.workspace_id 
WHERE w.owner_id = $1 OR wm.user_id = $1;

-- name: CreateWebhookTrigger :one
INSERT INTO webhook_triggers (task_id, token) VALUES ($1, $2) RETURNING *;

-- name: GetTaskByWebhookToken :one
SELECT t.* FROM tasks t JOIN webhook_triggers w ON t.id = w.task_id WHERE w.token = $1;

-- name: MoveToDLQ :one
INSERT INTO dlq_tasks (task_id, error_message) VALUES ($1, $2) RETURNING *;

-- name: CreateTemplate :one
INSERT INTO templates (name, description, config, is_public, workspace_id) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListOutboundWebhooks :many
SELECT id, endpoint_url, event_types, is_active, created_at
FROM outbound_webhooks
WHERE user_id = $1;

-- name: CreateOutboundWebhook :one
INSERT INTO outbound_webhooks (user_id, endpoint_url, event_types, encrypted_signing_secret)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at;

-- name: DeleteOutboundWebhook :exec
DELETE FROM outbound_webhooks WHERE id = $1 AND user_id = $2;

-- name: ListActiveOutboundWebhooks :many
SELECT id, endpoint_url, event_types, encrypted_signing_secret
FROM outbound_webhooks
WHERE user_id = $1 AND is_active = TRUE;

-- name: CreateWebhookDelivery :exec
INSERT INTO webhook_deliveries (webhook_id, user_id, event_type, status_code, success, response_body)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListWebhookDeliveries :many
SELECT * FROM webhook_deliveries WHERE webhook_id = $1 AND user_id = $2 ORDER BY created_at DESC LIMIT 50;

-- name: ListAuditLogs :many
SELECT id, user_id, action, resource_type, resource_id, metadata, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1;

-- name: GetSystemUsageMetrics :one
SELECT 
    (SELECT COUNT(*) FROM users) as user_count,
    (SELECT COUNT(*) FROM tasks) as task_count,
    (SELECT COUNT(*) FROM task_logs WHERE status = 'success') as success_count,
    (SELECT COUNT(*) FROM task_logs WHERE status = 'failure') as failure_count,
    (SELECT COUNT(*) FROM task_logs WHERE status = 'missed') as missed_count,
    (SELECT COUNT(*) FROM audit_logs) as audit_count;

-- name: CheckWorkspaceAccess :one
SELECT EXISTS (
    SELECT 1 FROM workspaces w
    LEFT JOIN workspace_members wm ON w.id = wm.workspace_id
    WHERE w.id = $1 AND (w.owner_id = $2 OR wm.user_id = $2)
) AS has_access;

-- name: CreateExecutionTrace :one
INSERT INTO execution_traces (task_id, execution_id, worker_id, step_name, input_data, output_data, is_error, error_message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListExecutionTracesByExecutionID :many
SELECT * FROM execution_traces 
WHERE task_id = $1 AND execution_id = $2 
ORDER BY start_time ASC;

-- name: ListTaskExecutionIDs :many
SELECT DISTINCT execution_id, MAX(start_time) as last_activity
FROM execution_traces
WHERE task_id = $1
GROUP BY execution_id
ORDER BY last_activity DESC
LIMIT 20;

-- name: GetTaskOutput :one
SELECT output_data 
FROM execution_traces e
JOIN tasks t ON e.task_id = t.id
WHERE e.task_id = $1 AND t.user_id = $2
ORDER BY e.start_time DESC 
LIMIT 1;

-- name: GetTemplateWithSubscription :one
SELECT t.*, s.subscribed_at IS NOT NULL as is_subscribed
FROM templates t
LEFT JOIN user_template_subscriptions s ON t.id = s.template_id AND s.user_id = $1
WHERE t.id = $2;

-- name: ListPublicTemplates :many
SELECT * FROM templates 
WHERE is_public = true AND (name ILIKE $1 OR description ILIKE $1)
ORDER BY created_at DESC;

-- name: GetTemplateByIDRaw :one
SELECT * FROM templates WHERE id = $1;

-- name: CreateTemplateSubscription :exec
INSERT INTO user_template_subscriptions (user_id, template_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UpsertWorkerHeartbeat :exec
INSERT INTO worker_heartbeats (worker_id, hostname, last_heartbeat, task_count)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (worker_id) DO UPDATE SET
    last_heartbeat = EXCLUDED.last_heartbeat,
    task_count = EXCLUDED.task_count;

-- name: GetActiveWorkerCount :one
SELECT COUNT(*) FROM worker_heartbeats WHERE last_heartbeat > NOW() - INTERVAL '2 minutes';

-- name: ListWorkerHeartbeats :many
SELECT * FROM worker_heartbeats ORDER BY last_heartbeat DESC;

-- name: GetP99ExecutionLatency :one
SELECT COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms), 0)::float
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '24 hours';

-- name: GetDailyExecutionTrends :many
SELECT 
    DATE(start_time)::text as date,
    COUNT(*)::int as count
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '7 days'
GROUP BY DATE(start_time)
ORDER BY date ASC;

-- name: GetGlobalSuccessRate :one
SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN 100.0
        ELSE (COUNT(*) FILTER (WHERE is_error = FALSE)::float / COUNT(*)::float) * 100.0
    END as success_rate
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '24 hours';

-- name: CreateTaskVersion :one
INSERT INTO task_versions (
    task_id, name, trigger_type, trigger_config, agent_prompt, 
    missed_task_policy, depends_on_task_id, requires_approval, 
    trigger_on_completion, task_type, native_code, branch_condition, is_bundle_root, loop_condition, swarm_config
) 
SELECT 
    t.id, t.name, t.trigger_type, t.trigger_config, t.agent_prompt, 
    t.missed_task_policy, t.depends_on_task_id, t.requires_approval, 
    t.trigger_on_completion, t.task_type, t.native_code, t.branch_condition, t.is_bundle_root, t.loop_condition, t.swarm_config
FROM tasks t WHERE t.id = $1 AND t.user_id = $2
RETURNING *;

-- name: ListTaskVersions :many
SELECT * FROM task_versions WHERE task_id = $1 ORDER BY created_at DESC;

-- name: GetTaskVersionByID :one
SELECT * FROM task_versions WHERE id = $1 AND task_id = $2;

-- name: RestoreTaskFromVersion :exec
UPDATE tasks
SET 
    name = v.name,
    trigger_type = v.trigger_type,
    trigger_config = v.trigger_config,
    agent_prompt = v.agent_prompt,
    missed_task_policy = v.missed_task_policy,
    depends_on_task_id = v.depends_on_task_id,
    requires_approval = v.requires_approval,
    trigger_on_completion = v.trigger_on_completion,
    task_type = v.task_type,
    native_code = v.native_code,
    branch_condition = v.branch_condition,
    loop_condition = v.loop_condition,
    is_bundle_root = v.is_bundle_root,
    swarm_config = v.swarm_config
FROM task_versions v
WHERE tasks.id = $1 AND tasks.user_id = $2 AND v.id = $3 AND v.task_id = $1;

-- name: UpsertWorkspaceEnvVar :one
INSERT INTO workspace_env_vars (workspace_id, name, value)
VALUES ($1, $2, $3)
ON CONFLICT (workspace_id, name) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListWorkspaceEnvVars :many
SELECT * FROM workspace_env_vars WHERE workspace_id = $1 ORDER BY name ASC;

-- name: DeleteWorkspaceEnvVar :exec
DELETE FROM workspace_env_vars WHERE workspace_id = $1 AND name = $2;

-- name: GetTaskWorkspaceEnvVars :many
SELECT e.name, e.value 
FROM workspace_env_vars e
JOIN tasks t ON e.workspace_id = t.workspace_id
WHERE t.id = $1;

-- name: UpsertWorkflowState :exec
INSERT INTO workflow_state (task_id, execution_id, state_data, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (task_id, execution_id) DO UPDATE SET
    state_data = EXCLUDED.state_data,
    updated_at = NOW();

-- name: GetWorkflowState :one
SELECT state_data FROM workflow_state WHERE task_id = $1 AND execution_id = $2;

-- name: GetLatestWorkflowState :one
SELECT state_data FROM workflow_state WHERE task_id = $1 ORDER BY updated_at DESC LIMIT 1;

-- name: GetSystemSettings :one
SELECT worker_prune_days FROM system_settings WHERE id = 1;

-- name: UpdateSystemSettings :exec
UPDATE system_settings 
SET worker_prune_days = $1, 
    updated_at = NOW() 
WHERE id = 1;

-- name: PruneZombieWorkers :exec
DELETE FROM worker_heartbeats 
WHERE last_heartbeat < NOW() - ($1 * INTERVAL '1 day');

-- name: GetCountTracesAfter :one
SELECT COUNT(*) FROM execution_traces WHERE start_time > $1;

-- name: GetCountTracesBetween :one
SELECT COUNT(*) FROM execution_traces WHERE start_time > $1 AND start_time <= $2;

-- name: GetSuccessRateAfter :one
SELECT COALESCE((COUNT(*) FILTER (WHERE is_error = FALSE)::float / NULLIF(COUNT(*), 0)::float) * 100, 100.0)::float
FROM execution_traces WHERE start_time > $1;

-- name: GetSuccessRateBetween :one
SELECT COALESCE((COUNT(*) FILTER (WHERE is_error = FALSE)::float / NULLIF(COUNT(*), 0)::float) * 100, 100.0)::float
FROM execution_traces WHERE start_time > $1 AND start_time <= $2;

-- name: GetCountUsersAfter :one
SELECT COUNT(*) FROM users WHERE created_at > $1;

-- name: GetCountUsersBetween :one
SELECT COUNT(*) FROM users WHERE created_at > $1 AND created_at <= $2;
