ALTER TABLE tasks ADD COLUMN requires_approval BOOLEAN DEFAULT FALSE;
ALTER TABLE tasks ADD COLUMN encrypted_secrets BYTEA;
ALTER TABLE tasks ADD COLUMN last_approval_status VARCHAR(20); -- 'pending', 'approved', 'denied'

-- Redefine fn_complete_task to handle the new approval columns
CREATE OR REPLACE FUNCTION fn_complete_task(task_id UUID, new_next_run TIMESTAMP WITH TIME ZONE, new_status TEXT DEFAULT 'active')
RETURNS VOID AS $$
BEGIN
    UPDATE tasks
    SET status = new_status,
        locked_by = NULL,
        last_run = NOW(),
        failure_count = 0, -- reset on success
        next_run = new_next_run,
        last_approval_status = CASE WHEN requires_approval THEN 'pending' ELSE last_approval_status END
    WHERE id = task_id;
END;
$$ LANGUAGE plpgsql;
