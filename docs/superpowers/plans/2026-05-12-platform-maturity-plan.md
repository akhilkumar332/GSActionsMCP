# Platform Maturity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Hybrid Execution, Advanced Tracing, Insights Dashboard, and a Monetized Marketplace.

**Architecture:**
1. **Hybrid Execution:** Extend the scheduler to handle `native_action` tasks using a new `executor.go` and `v8go` (or `otto`) for sandboxed JS.
2. **Advanced Tracing:** Refactor `task_logs` into a more detailed `execution_traces` table and update the `MaintainHeartbeat` loop to capture millisecond-level data.
3. **Insights:** Add aggregate queries in `db/queries.sql` and expose them via a new `api_analytics.go` handler.
4. **Marketplace:** Implement `templates` API with paywall logic linked to Stripe.

**Tech Stack:** Go (Echo), PostgreSQL (sqlc), Redis, React (Recharts/Chart.js), Stripe API, `github.com/robertkrimen/otto` (for JS sandbox).

---

### Task 1: Database Migrations for Tracing and Marketplace

**Files:**
- Create: `migrations/012_maturity_features.up.sql`
- Create: `migrations/012_maturity_features.down.sql`
- Modify: `db/queries.sql`

- [ ] **Step 1: Write migration up script**

```sql
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
```

- [ ] **Step 2: Write migration down script**

```sql
ALTER TABLE tasks DROP COLUMN native_code;
ALTER TABLE tasks DROP COLUMN task_type;

DROP TABLE IF EXISTS user_template_subscriptions;
ALTER TABLE templates DROP COLUMN author_id;
ALTER TABLE templates DROP COLUMN is_premium;
ALTER TABLE templates DROP COLUMN price_id;

DROP TABLE IF EXISTS execution_traces;
```

- [ ] **Step 3: Update db/queries.sql**

```sql
-- name: CreateExecutionTrace :one
INSERT INTO execution_traces (task_id, execution_id, worker_id, step_name, input_data, output_data, is_error, error_message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListTaskTraces :many
SELECT * FROM execution_traces WHERE task_id = $1 ORDER BY start_time DESC;

-- name: GetTemplateWithSubscription :one
SELECT t.*, s.subscribed_at IS NOT NULL as is_subscribed
FROM templates t
LEFT JOIN user_template_subscriptions s ON t.id = s.template_id AND s.user_id = $1
WHERE t.id = $2;
```

- [ ] **Step 4: Run sqlc generate**

Run: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

- [ ] **Step 5: Apply migration**

Run: `./scripts/migrate.sh up`

- [ ] **Step 6: Commit**

```bash
git add migrations/ db/
git commit -m "feat(db): schema for tracing, hybrid tasks and marketplace"
```

---

### Task 2: Core Engine - Hybrid Executor (JS Sandbox)

**Files:**
- Create: `cmd/server/executor.go`
- Modify: `cmd/server/scheduler.go`
- Modify: `go.mod`

- [ ] **Step 1: Install JS Engine**

Run: `go get github.com/robertkrimen/otto`

- [ ] **Step 2: Implement Native Executor in executor.go**

```go
package main

import (
	"context"
	"github.com/robertkrimen/otto"
)

func executeNativeJS(ctx context.Context, code string, input map[string]interface{}) (string, error) {
	vm := otto.New()
	vm.Set("input", input)
	
	// Add native helpers like fetch/log if needed
	
	val, err := vm.Run(code)
	if err != nil {
		return "", err
	}
	return val.String(), nil
}
```

- [ ] **Step 3: Update handleClaimedTask in scheduler.go to route by type**

```go
// In scheduler.go inside handleClaimedTask
if t.TaskType.String == "native_action" {
    result, err := executeNativeJS(workerCtx, t.NativeCode.String, map[string]interface{}{"task_id": taskID})
    // ... handle result and completeTask ...
} else {
    // Current MCP Sampling logic
}
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/ go.mod go.sum
git commit -m "feat(backend): implement native JS executor"
```

---

### Task 3: Advanced Tracing Integration

**Files:**
- Modify: `cmd/server/scheduler.go`
- Modify: `cmd/server/session.go`

- [ ] **Step 1: Refactor logging to include ExecutionTrace calls**

In `scheduler.go` and `session.go`, wherever a major step starts/ends (Prompt Resolution, LLM Sampling, Native Execution), call `queries.CreateExecutionTrace`.

- [ ] **Step 2: Update Trace metadata**

Ensure the `execution_id` (already generated in `scheduler.go`) is passed down and stored in every trace step.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/
git commit -m "feat(backend): integrate execution tracing into task lifecycle"
```

---

### Task 4: Analytics API and Insights Dashboard

**Files:**
- Create: `cmd/server/api_analytics.go`
- Modify: `cmd/server/main.go`
- Create: `frontend/src/pages/Insights.jsx`

- [ ] **Step 1: Implement Analytics handlers**

```go
// api_analytics.go
func handleGetSystemInsights(c echo.Context) error {
    // Query db for aggregate latency, success rates, etc.
    return c.JSON(http.StatusOK, map[string]interface{}{"p99_latency": 1200, "success_rate": 0.98})
}
```

- [ ] **Step 2: Register Admin routes**

In `main.go`, add `admin.GET("/insights", handleGetSystemInsights)`.

- [ ] **Step 3: Build Insights Dashboard with Recharts**

Run: `npm install recharts` in `frontend/`
Implement `Insights.jsx` with Line/Bar charts.

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat(platform): implement insights dashboard and analytics API"
```

---

### Task 5: Monetized Marketplace UI and Paywall

**Files:**
- Create: `frontend/src/pages/Marketplace.jsx`
- Modify: `cmd/server/template_handlers.go`
- Modify: `cmd/server/billing_handlers.go`

- [ ] **Step 1: Implement Marketplace API**

Update `template_handlers.go` to support listing public templates and checking subscription status.

- [ ] **Step 2: Integrate Stripe for Premium Templates**

In `billing_handlers.go`, add logic to create a Checkout Session for a specific `template_id`.

- [ ] **Step 3: Build Marketplace UI**

Create `Marketplace.jsx` with template cards, "Premium" badges, and "Buy/Install" buttons.

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat(platform): implement monetized workflow marketplace"
```

---

### Task 6: Final Verification, Lint and Build

- [ ] **Step 1: Run Go Lint and Build**

Run: `go build ./cmd/server`
Expected: Success

- [ ] **Step 2: Run Frontend Build**

Run: `cd frontend && npm run build`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: final verification of maturity features"
```
