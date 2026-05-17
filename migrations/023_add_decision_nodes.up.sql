ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_task_type_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_task_type_check CHECK (task_type IN ('mcp_sampling', 'native_action', 'decision_router'));

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN ('active', 'paused', 'processing', 'completed', 'error', 'halted'));

-- Index for finding branches out of a decision node
CREATE INDEX IF NOT EXISTS idx_tasks_depends_on_router ON tasks (depends_on_task_id) WHERE status != 'completed';
