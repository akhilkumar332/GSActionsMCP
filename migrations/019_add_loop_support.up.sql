ALTER TABLE tasks ADD COLUMN IF NOT EXISTS loop_condition JSONB; -- e.g. {"state_path": "count", "op": "lt", "value": 5}
