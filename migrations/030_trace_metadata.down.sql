-- migrations/030_trace_metadata.down.sql
ALTER TABLE execution_traces DROP COLUMN metadata;
