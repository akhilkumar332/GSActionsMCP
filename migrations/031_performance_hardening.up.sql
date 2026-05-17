-- migrations/031_performance_hardening.up.sql
-- Add indices for high-frequency filters
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks (user_id);
CREATE INDEX IF NOT EXISTS idx_task_logs_task_id_time ON task_logs (task_id, execution_time DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_user_id ON webhook_deliveries (user_id);

-- Optimize execution traces for time-travel scrubbing
CREATE INDEX IF NOT EXISTS idx_execution_traces_lookup ON execution_traces (task_id, execution_id, start_time ASC);
