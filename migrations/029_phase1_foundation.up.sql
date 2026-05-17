-- migrations/029_phase1_foundation.up.sql
ALTER TABLE system_settings 
ADD COLUMN js_timeout_ms INT NOT NULL DEFAULT 5000,
ADD COLUMN reaper_stuck_threshold_minutes INT NOT NULL DEFAULT 5,
ADD COLUMN scheduler_poll_interval_seconds INT NOT NULL DEFAULT 30;

-- Update existing row
UPDATE system_settings SET js_timeout_ms = 5000, reaper_stuck_threshold_minutes = 5, scheduler_poll_interval_seconds = 30 WHERE id = 1;

-- Trigger to notify Go when a task is immediately runnable
CREATE OR REPLACE FUNCTION fn_notify_task_queued() RETURNS TRIGGER AS $$
BEGIN
    -- Only notify if it's active and due now or in the past
    IF NEW.status = 'active' AND NEW.next_run <= NOW() THEN
        PERFORM pg_notify('task_queued', json_build_object('task_id', NEW.id::text)::text);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_task_queued
AFTER INSERT OR UPDATE OF status, next_run ON tasks
FOR EACH ROW
EXECUTE FUNCTION fn_notify_task_queued();