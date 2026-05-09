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
SELECT id, email, api_key, role, tier, created_at FROM users ORDER BY created_at DESC;

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

-- name: UpdateTaskNextRun :exec
UPDATE tasks SET status = $1, locked_by = NULL, next_run = $2 WHERE id = $3;

-- name: GetLatestTaskLogResponse :one
SELECT llm_response FROM task_logs 
WHERE task_id = $1 
ORDER BY execution_time DESC 
LIMIT 1;

-- name: IncrementTaskFailureCount :one
UPDATE tasks SET failure_count = failure_count + 1 
WHERE id = $1 
RETURNING failure_count;

-- name: UpdateTaskStatus :exec
UPDATE tasks SET status = $1, locked_by = NULL WHERE id = $2;

-- name: UpdateTaskStatusAndFailureCount :exec
UPDATE tasks SET status = $1, locked_by = NULL, failure_count = $2 WHERE id = $3;

-- name: ClaimDueTasks :many
SELECT * FROM fn_claim_due_tasks($1, $2);

-- name: CompleteTask :exec
SELECT fn_complete_task($1, $2, $3);

-- name: ReapStuckTasks :execrows
UPDATE tasks 
SET status = 'active', locked_by = NULL 
WHERE status = 'processing' AND next_run < NOW() - INTERVAL '5 minutes';

-- name: CheckTaskOwnership :one
SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2);

-- name: CreateTask :one
INSERT INTO tasks (user_id, name, trigger_type, trigger_config, agent_prompt, missed_task_policy, depends_on_task_id, next_run, requires_approval, encrypted_secrets, trigger_on_completion) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
RETURNING *;

-- name: ListUserTasks :many
SELECT id, name, trigger_type, status, next_run, requires_approval, encrypted_secrets, last_approval_status, trigger_on_completion FROM tasks WHERE user_id = $1;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = $1;

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
