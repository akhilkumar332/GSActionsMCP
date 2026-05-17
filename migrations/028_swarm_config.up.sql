ALTER TABLE tasks ADD COLUMN swarm_config JSONB;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_task_type_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_task_type_check CHECK (task_type IN ('mcp_sampling', 'native_action', 'decision_router', 'swarm_router'));
