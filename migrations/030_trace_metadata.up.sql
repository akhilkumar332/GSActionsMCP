-- migrations/030_trace_metadata.up.sql
ALTER TABLE execution_traces ADD COLUMN metadata JSONB;
