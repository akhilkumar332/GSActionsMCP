# Advanced Analytics and Worker Heartbeats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace mock analytics with real PostgreSQL aggregate data and implement a worker heartbeat system for cluster observability.

**Architecture:** 
1. Database migration for `worker_heartbeats`.
2. SQLC aggregate queries for P99 latency and daily execution trends.
3. Background heartbeat loop in worker nodes.
4. Refactored analytics API handler.

**Tech Stack:** Go (Echo), PostgreSQL (sqlc), Redis.

---

### Task 1: Database Migration for Worker Heartbeats

**Files:**
- Create: `migrations/013_worker_heartbeats.up.sql`
- Create: `migrations/013_worker_heartbeats.down.sql`
- Modify: `db/queries.sql`

- [ ] **Step 1: Write migration up script**

```sql
CREATE TABLE worker_heartbeats (
    worker_id TEXT PRIMARY KEY,
    hostname TEXT,
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    task_count INT DEFAULT 0 -- currently processing
);
```

- [ ] **Step 2: Write migration down script**

```sql
DROP TABLE IF EXISTS worker_heartbeats;
```

- [ ] **Step 3: Add heartbeat and analytics queries to db/queries.sql**

```sql
-- name: UpsertWorkerHeartbeat :exec
INSERT INTO worker_heartbeats (worker_id, hostname, last_heartbeat, task_count)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (worker_id) DO UPDATE SET
    last_heartbeat = EXCLUDED.last_heartbeat,
    task_count = EXCLUDED.task_count;

-- name: GetActiveWorkerCount :one
SELECT COUNT(*) FROM worker_heartbeats WHERE last_heartbeat > NOW() - INTERVAL '2 minutes';

-- name: GetP99ExecutionLatency :one
SELECT COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms), 0)::float
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '24 hours';

-- name: GetDailyExecutionTrends :many
SELECT 
    DATE(start_time)::text as date,
    COUNT(*)::int as count
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '7 days'
GROUP BY DATE(start_time)
ORDER BY date ASC;

-- name: GetGlobalSuccessRate :one
SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN 100.0
        ELSE (COUNT(*) FILTER (WHERE is_error = FALSE)::float / COUNT(*)::float) * 100.0
    END as success_rate
FROM execution_traces
WHERE start_time > NOW() - INTERVAL '24 hours';
```

- [ ] **Step 4: Run sqlc generate**

Run: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

- [ ] **Step 5: Commit**

```bash
git add migrations/ db/
git commit -m "feat(db): add worker heartbeats and analytics queries"
```

---

### Task 2: Worker Heartbeat Implementation

**Files:**
- Modify: `cmd/server/scheduler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add heartbeat loop to scheduler.go**

```go
func runWorkerHeartbeat(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    hostname, _ := os.Hostname()
    
    for {
        select {
        case <-ticker.C:
            err := queries.UpsertWorkerHeartbeat(ctx, db.UpsertWorkerHeartbeatParams{
                WorkerID:  workerID,
                Hostname:  pgtype.Text{String: hostname, Valid: true},
                TaskCount: 0, // In future, track active go-routines
            })
            if err != nil {
                log.Printf("Heartbeat error: %v", err)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

- [ ] **Step 2: Start heartbeat in main.go**

In `main.go`, after `workerID` is initialized:
`go runWorkerHeartbeat(ctx)`

- [ ] **Step 3: Commit**

```bash
git add cmd/server/
git commit -m "feat(backend): implement worker heartbeat loop"
```

---

### Task 3: Refactor Analytics API to use Real Data

**Files:**
- Modify: `cmd/server/api_analytics.go`

- [ ] **Step 1: Replace mock data with DB queries**

```go
func handleGetSystemInsights(c echo.Context) error {
    ctx := c.Request().Context()
    
    p99, _ := queries.GetP99ExecutionLatency(ctx)
    successRate, _ := queries.GetGlobalSuccessRate(ctx)
    workerCount, _ := queries.GetActiveWorkerCount(ctx)
    trends, _ := queries.GetDailyExecutionTrends(ctx)
    
    // Map trends to expected format
    dailyTasks := []map[string]interface{}{}
    for _, t := range trends {
        dailyTasks = append(dailyTasks, map[string]interface{}{
            "date": t.Date,
            "count": t.Count,
        })
    }

    data := map[string]interface{}{
        "p99_latency":    int64(p99),
        "success_rate":   successRate,
        "active_workers": workerCount,
        "daily_tasks":    dailyTasks,
    }
    
    return c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}
```

- [ ] **Step 2: Verify with build**

Run: `go build ./cmd/server`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add cmd/server/api_analytics.go
git commit -m "feat(backend): hook up Insights dashboard to real system data"
```

---

### Task 4: Final Verification and Build

- [ ] **Step 1: Run Full Lint and Build**

Run: `go vet ./... && cd frontend && npm run lint && npm run build`
Expected: Success

- [ ] **Step 2: Commit**

```bash
git commit -m "chore: verify advanced analytics implementation"
```
