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
    trigger_on_completion BOOLEAN DEFAULT FALSE,
    task_type TEXT DEFAULT 'mcp_sampling' CHECK (task_type IN ('mcp_sampling', 'native_action')),
    native_code TEXT
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

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);

CREATE TABLE outbound_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint_url TEXT NOT NULL,
    event_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    encrypted_signing_secret BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbound_webhooks_user_id ON outbound_webhooks (user_id);

CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES outbound_webhooks(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    status_code INT,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    response_body TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries (webhook_id, created_at DESC);

-- The "Claim" Function
-- This grabs 'batch_size' tasks that are due and locks them so other workers ignore them.
CREATE OR REPLACE FUNCTION fn_claim_due_tasks(batch_size INT, worker_id TEXT)
RETURNS SETOF tasks AS $$
DECLARE
    claimed_task tasks%ROWTYPE;
BEGIN
    FOR claimed_task IN
        WITH claimed AS (
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
            RETURNING *
        )
        SELECT * FROM claimed
    LOOP
        -- NOTIFY is delivered only if the transaction commits, so the claim and wake-up signal stay atomic.
        PERFORM pg_notify(
            'task_claimed',
            json_build_object(
                'task_id', claimed_task.id::text,
                'user_id', claimed_task.user_id,
                'worker_id', worker_id
            )::text
        );
        RETURN NEXT claimed_task;
    END LOOP;
    RETURN;
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
        retry_count = 0,   -- reset on success
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

CREATE TABLE seo_settings (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- Singleton
    title TEXT NOT NULL DEFAULT 'Schedule MCP',
    description TEXT NOT NULL DEFAULT 'Durable AI Workflows',
    keywords TEXT NOT NULL DEFAULT 'AI, MCP, Scheduler',
    og_image TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed initial settings
INSERT INTO seo_settings (id, title, description, keywords) 
VALUES (1, 'Schedule MCP', 'Durable AI Workflows', 'AI, MCP, Scheduler')
ON CONFLICT DO NOTHING;

-- Workspaces
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE workspace_members (
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member',
    PRIMARY KEY (workspace_id, user_id)
);

-- Templates
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    is_public BOOLEAN DEFAULT false,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    price_id TEXT, -- Stripe Price ID
    is_premium BOOLEAN DEFAULT false,
    author_id TEXT REFERENCES users(id)
);

CREATE TABLE user_template_subscriptions (
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, template_id)
);

CREATE TABLE execution_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    execution_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    step_name TEXT NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_ms INT,
    input_data JSONB,
    output_data JSONB,
    is_error BOOLEAN DEFAULT false,
    error_message TEXT
);

-- Inbound Webhooks
CREATE TABLE webhook_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- DLQ Tasks
CREATE TABLE dlq_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    error_message TEXT,
    failed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Update Tasks
ALTER TABLE tasks ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE tasks ADD COLUMN max_retries INT DEFAULT 0;
ALTER TABLE tasks ADD COLUMN retry_count INT DEFAULT 0;
ALTER TABLE tasks ADD COLUMN backoff_strategy VARCHAR(50) DEFAULT 'linear';
ALTER TABLE tasks ADD COLUMN ui_coordinates JSONB;

CREATE TABLE worker_heartbeats (
    worker_id TEXT PRIMARY KEY,
    hostname TEXT,
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    task_count INT DEFAULT 0 -- currently processing
);

CREATE TABLE task_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    trigger_config JSONB NOT NULL,
    agent_prompt TEXT NOT NULL,
    missed_task_policy TEXT NOT NULL,
    depends_on_task_id UUID,
    requires_approval BOOLEAN NOT NULL,
    trigger_on_completion BOOLEAN NOT NULL,
    task_type TEXT NOT NULL,
    native_code TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_task_versions_task_id ON task_versions (task_id);

-- Workspace Environment Variables
CREATE TABLE workspace_env_vars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, name)
);

CREATE INDEX idx_workspace_env_vars_workspace_id ON workspace_env_vars (workspace_id);
