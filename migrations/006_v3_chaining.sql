ALTER TABLE tasks ADD COLUMN trigger_on_completion BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_tasks_depends_on ON tasks (depends_on_task_id) WHERE trigger_on_completion = TRUE;
