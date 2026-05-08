# Scheduled Actions MCP Server Roadmap

## Phase 1: Closing the Loop (The Interactive Update)
**Goal:** Move from fire-and-forget triggers to conversational persistence.
- **Action 1.1 - True Session Tracking:** Hook into the SDK's transport layer to map `mcp.Session` pointers to `user_id`s in the `GlobalSessionManager`.
- **Action 1.2 - Execution Logging:** Create a `task_logs` table in Postgres.
- **Action 1.3 - The Response Webhook:** Modify the `SendSamplingRequest` to wait for the LLM's text response. Save that response to `task_logs`. (e.g., The user can later ask, "What did my morning summary say yesterday?").

## Phase 2: SaaS Business Logic (Monetization & Control)
**Goal:** Prevent users from bankrupting your server and enforce billing tiers.
- **Action 2.1 - Concrete Users Table:** Create a `users` table with `api_key`, `email`, and `tier` (e.g., 'free', 'plus', 'pro').
- **Action 2.2 - Tool Quotas:** Update `create_task` so that 'free' users can schedule up to 2 tasks, 'plus' users get 10, and 'pro' users get 20.
- **Action 2.3 - The Dead Letter Queue:** If a task fails to reach the user 3 times in a row (`failure_count >= 3`), automatically set the status to `error`.

## Phase 3: Advanced Scheduling Features
**Goal:** Provide capabilities the native Gemini UI cannot offer.
- **Action 3.1 - "Run on Reconnect" Policies:** Add a `missed_task_policy` column to the DB (e.g. 'skip', 'run_immediately').
- **Action 3.2 - Chained Actions:** Allow a task's `trigger_config` to rely on the completion of another task (e.g., Task B only fires 5 minutes after Task A succeeds).

## Phase 4: Telemetry & Observability
**Goal:** Ensure you have a God-view of the system for 10,000 users.
- **Action 4.1 - Prometheus Integration:** Expose a `/metrics` endpoint in Go that reports active SSE connections, tasks processed, and DB metrics.
- **Action 4.2 - Health Endpoints:** Add a `/healthz` endpoint for Docker/Kubernetes health checks.
