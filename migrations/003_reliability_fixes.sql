-- migrations/003_reliability_fixes.sql

-- Redefine fn_claim_due_tasks to ensure it only joins with dependencies belonging to the same user
-- Note: This will be redefined again in later migrations if columns are added.
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
          AND (t.depends_on_task_id IS NULL OR (dep.user_id = t.user_id AND (dep.status = 'completed' OR dep.status = 'active')))
        ORDER BY t.next_run ASC
        LIMIT batch_size
        FOR UPDATE OF t SKIP LOCKED -- CRITICAL: Prevents race conditions
    )
    RETURNING *;
END;
$$ LANGUAGE plpgsql;

-- Redefine fn_complete_task to allow setting status (e.g. for one-off tasks)
CREATE OR REPLACE FUNCTION fn_complete_task(task_id UUID, new_next_run TIMESTAMP WITH TIME ZONE, new_status TEXT DEFAULT 'active')
RETURNS VOID AS $$
BEGIN
    UPDATE tasks
    SET status = new_status,
        locked_by = NULL,
        last_run = NOW(),
        failure_count = 0, -- reset on success
        next_run = new_next_run
    WHERE id = task_id;
END;
$$ LANGUAGE plpgsql;
