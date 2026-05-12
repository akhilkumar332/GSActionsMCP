-- Execution Tracing
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

-- Marketplace Monetization
ALTER TABLE templates ADD COLUMN price_id TEXT; -- Stripe Price ID
ALTER TABLE templates ADD COLUMN is_premium BOOLEAN DEFAULT false;
ALTER TABLE templates ADD COLUMN author_id TEXT REFERENCES users(id);

CREATE TABLE user_template_subscriptions (
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, template_id)
);

-- Hybrid Task Support
ALTER TABLE tasks ADD COLUMN task_type TEXT DEFAULT 'mcp_sampling' CHECK (task_type IN ('mcp_sampling', 'native_action'));
ALTER TABLE tasks ADD COLUMN native_code TEXT;
