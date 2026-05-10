-- migrations/007_repair_schema.sql
-- This migration ensures that all columns exist and functions are up to date.
-- It's designed to be safe to run even if some changes are already present.

DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tasks' AND column_name='requires_approval') THEN
        ALTER TABLE tasks ADD COLUMN requires_approval BOOLEAN DEFAULT FALSE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tasks' AND column_name='encrypted_secrets') THEN
        ALTER TABLE tasks ADD COLUMN encrypted_secrets BYTEA;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tasks' AND column_name='last_approval_status') THEN
        ALTER TABLE tasks ADD COLUMN last_approval_status VARCHAR(20);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tasks' AND column_name='trigger_on_completion') THEN
        ALTER TABLE tasks ADD COLUMN trigger_on_completion BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Ensure indexes exist
CREATE INDEX IF NOT EXISTS idx_tasks_next_run_status ON tasks (next_run, status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_tasks_depends_on ON tasks (depends_on_task_id) WHERE trigger_on_completion = TRUE;

-- Redefine functions to latest version
CREATE OR REPLACE FUNCTION fn_claim_due_tasks(batch_size INT, worker_id TEXT)
RETURNS SETOF tasks AS $$
BEGIN
    RETURN QUERY
    UPDATE tasks
    SET status = 'processing', -- Temporary state to prevent double-firing
        locked_by = worker_id
    WHERE id IN (
        SELECT t.id 
        FROM tasks t
        -- Ensure dependency belongs to same user AND is in a valid state
        LEFT JOIN tasks dep ON t.depends_on_task_id = dep.id
        WHERE t.next_run <= NOW() 
          AND t.status = 'active'
          -- Ensure dependency belongs to same user AND is in a valid state
          AND (t.depends_on_task_id IS NULL OR (dep.user_id = t.user_id AND (dep.status = 'completed' OR dep.status = 'active')))
        ORDER BY t.next_run ASC
        LIMIT batch_size
        FOR UPDATE OF t SKIP LOCKED -- CRITICAL: Prevents race conditions
    )
    RETURNING *;
END;
$$ LANGUAGE plpgsql;

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
