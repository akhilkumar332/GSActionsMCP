# Scheduled Actions MCP Server Project Instructions

## Architecture Rules
- **Strict Concurrency**: Use the `FOR UPDATE SKIP LOCKED` pattern in PLpgSQL for all task fetching to handle high concurrency across workers.
- **Stateless Application**: The Go server must remain stateless regarding tasks; PostgreSQL is the single source of truth.
- **SSE Stability**: Ensure robust SSE connections. Utilize `http.Flush()` and set `Keep-Alive` headers to prevent Gemini CLI connection timeouts.
- **Session Management**: Implement a thread-safe in-memory map (`Map[UserID]Connection`) in Go to track active user SSE sessions for routing "Sampling" requests.
- **Offline Handling**: The background scheduler must handle "Offline" users gracefully (e.g., revert status to active or mark as "Missed").

## Conventions
- **Tool Naming Port Consistency**: Tool names MUST exactly match the `liao1fan/schedule-task-mcp` reference:
  - `create_task`
  - `list_tasks`
  - `delete_task`
  - `pause_task`
  - `resume_task`
- **Data Types**: `trigger_config` is stored as JSONB.

## Build and Deployment
- Containerized deployment using Docker (multi-stage build aiming for under 20MB image size).
- Requires infrastructure tuning for 10k connections (e.g., Caddy proxy configuration, increased Linux ulimits).
