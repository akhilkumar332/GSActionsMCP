# Scheduled Actions MCP Server - Version 2.0 Roadmap

## Phase 5: Codebase Architecture Refactoring
**Goal:** Clean up `main.go` and ensure the project structure is modular, maintainable, and scalable for enterprise development.
- **Action 5.1 - Centralize Constants:** Extract all hardcoded strings, tier limits, status strings, and defaults into a single `constants.go` or `config/constants.go` file.
- **Action 5.2 - Centralize Global Variables:** Move global variables (like `dbPool`, `workerID`, `workerWG`, and `globalRateLimiter`) out of `main.go` into dedicated state management files (e.g., `globals.go` or `state/state.go`).
- **Action 5.3 - Modular Routing:** Extract the HTTP mux and middleware setup into its own dedicated package or file.

## Phase 6: Distributed Scaling (The Redis Rewrite)
**Goal:** Allow infinite horizontal scaling by decoupling state from the Go binaries.
- **Action 6.1 - Redis Pub/Sub Integration:** Introduce Redis. When a task triggers on Server 2, it publishes a message to Redis: `TRIGGER_TASK:<user_id>`. Server 1 (holding the user's connection) hears it and sends the SSE payload.
- **Action 6.2 - Distributed Session State:** Move `GlobalSessionManager` to Redis using temporary Keys with TTLs (Time-To-Live) that act as "Heartbeats".

## Phase 7: True SDK Interception
**Goal:** Bridge the gap between the Go SDK and the physical SSE stream.
- **Action 7.1 - Custom SSE Transport:** Override the default `mcp.NewSSEHandler` to capture the physical `mcp.Session` interface upon connection.
- **Action 7.2 - Real LLM Response Handling:** Implement a callback listener so when Gemini finishes executing the Scheduled Prompt, the output text is asynchronously routed back to the server and cleanly saved into the `task_logs.llm_response` column.

## Phase 8: The Monetization API (Billing)
**Goal:** Automate the business operations.
- **Action 8.1 - Stripe Webhooks:** Create an unprotected (but signed) HTTP endpoint `/webhooks/stripe`. When a user pays, it automatically executes `UPDATE users SET tier = 'pro' WHERE email = $1`.
- **Action 8.2 - API Key Generation Portal:** Build a lightweight HTML/Template route in Go (`/dashboard`) where users can log in via GitHub/Google, view their tasks, and copy their `X-API-Key`.

## Phase 9: Advanced User Alerts & AI Context
**Goal:** Provide premium features to keep retention high.
- **Action 9.1 - Dead Letter Email Alerts:** When a task hits `failure_count >= 3` and is quarantined, trigger an email via SendGrid: *"Your scheduled action 'Morning Summary' has failed 3 times."*
- **Action 9.2 - Cross-Task Context:** Allow "Chained Actions" to pass data. If Task A runs `curl api.weather.com` and saves the output, Task B (running 5 mins later) injects Task A's output into its Prompt.
