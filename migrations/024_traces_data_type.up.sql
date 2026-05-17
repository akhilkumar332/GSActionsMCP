-- schedule-mcp/migrations/024_traces_data_type.up.sql
ALTER TABLE execution_traces ALTER COLUMN input_data TYPE TEXT USING input_data::text;
ALTER TABLE execution_traces ALTER COLUMN output_data TYPE TEXT USING output_data::text;
