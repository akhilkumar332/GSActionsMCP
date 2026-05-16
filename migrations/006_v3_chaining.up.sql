ALTER TABLE tasks ADD COLUMN IF NOT EXISTS trigger_on_completion BOOLEAN DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_tasks_depends_on ON tasks (depends_on_task_id) WHERE trigger_on_completion = TRUE;
