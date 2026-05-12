ALTER TABLE tasks DROP COLUMN native_code;
ALTER TABLE tasks DROP COLUMN task_type;

DROP TABLE IF EXISTS user_template_subscriptions;
ALTER TABLE templates DROP COLUMN author_id;
ALTER TABLE templates DROP COLUMN is_premium;
ALTER TABLE templates DROP COLUMN price_id;

DROP TABLE IF EXISTS execution_traces;
