-- migrations/031_performance_hardening.down.sql
DROP INDEX IF EXISTS idx_tasks_user_id;
DROP INDEX IF EXISTS idx_task_logs_task_id_time;
DROP INDEX IF EXISTS idx_webhook_deliveries_user_id;
DROP INDEX IF EXISTS idx_execution_traces_lookup;
