CREATE TABLE IF NOT EXISTS task_versions (
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

CREATE INDEX IF NOT EXISTS idx_task_versions_task_id ON task_versions (task_id);
