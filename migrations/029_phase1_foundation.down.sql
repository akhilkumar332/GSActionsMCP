-- migrations/029_phase1_foundation.down.sql
DROP TRIGGER IF EXISTS trg_task_queued ON tasks;
DROP FUNCTION IF EXISTS fn_notify_task_queued;

ALTER TABLE system_settings 
DROP COLUMN js_timeout_ms,
DROP COLUMN reaper_stuck_threshold_minutes,
DROP COLUMN scheduler_poll_interval_seconds;