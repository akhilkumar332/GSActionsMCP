-- schema.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Phase 2.1: Users Table
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    api_key TEXT UNIQUE NOT NULL,
    email TEXT,
    password_hash TEXT,
    role TEXT DEFAULT 'user' CHECK (role IN ('user', 'staff', 'admin')),
    last_login TIMESTAMP WITH TIME ZONE,
    tier TEXT DEFAULT 'free' CHECK (tier IN ('free', 'plus', 'pro')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Management Portal Sessions
CREATE TABLE web_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Phase 2.3 & 3.1 & 3.2: Tasks Table with advanced fields
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    trigger_type TEXT CHECK (trigger_type IN ('cron', 'interval', 'date')),
    trigger_config JSONB NOT NULL, -- Stores {"cron": "* * * * *"} or {"minutes": 5}
    agent_prompt TEXT NOT NULL,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'processing', 'completed', 'error')),
    locked_by TEXT, -- Tracks which worker instance is processing this task
    next_run TIMESTAMP WITH TIME ZONE NOT NULL,
    last_run TIMESTAMP WITH TIME ZONE,
    failure_count INT DEFAULT 0, -- Phase 2.3: Dead Letter Queue
    missed_task_policy TEXT DEFAULT 'skip' CHECK (missed_task_policy IN ('skip', 'run_immediately')), -- Phase 3.1
    depends_on_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL, -- Phase 3.2
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    requires_approval BOOLEAN DEFAULT FALSE,
    encrypted_secrets BYTEA,
    last_approval_status VARCHAR(20), -- 'pending', 'approved', 'denied'
    trigger_on_completion BOOLEAN DEFAULT FALSE
);

-- Index for high-speed polling
CREATE INDEX idx_tasks_next_run_status ON tasks (next_run, status) WHERE status = 'active';

-- Phase 3.2: Chaining optimization
CREATE INDEX idx_tasks_depends_on ON tasks (depends_on_task_id) WHERE trigger_on_completion = TRUE;

-- Phase 1.2: Execution Logging
CREATE TABLE task_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    execution_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('success', 'failure', 'missed')),
    llm_response TEXT, -- Phase 1.3: The Response Webhook
    error_message TEXT
);

-- The "Claim" Function
-- This grabs 'batch_size' tasks that are due and locks them so other workers ignore them.
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
        -- Phase 3.2: Ensure dependencies are met (or no dependencies exist)
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

-- The "Post-Execution" Function
-- After the Go worker sends the Sampling request, it calls this to set the next time.
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

-- Phase 3.1: Secret Vault
CREATE TABLE user_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);
