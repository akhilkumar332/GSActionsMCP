# Production Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prepare the schedule-mcp application for production deployment by fixing the SDK build, implementing distributed Redis rate limiting, adding automated tests, and setting up observability endpoints.

**Architecture:** We are transitioning from in-memory primitives (like the local rate limiter) to a distributed model using Redis and Lua scripts to ensure stateless operation across multiple worker instances. We will introduce testcontainers for reliable integration testing of our concurrency logic (`FOR UPDATE SKIP LOCKED`), and expose standard Prometheus endpoints for ops.

**Tech Stack:** Go 1.25, mark3labs/mcp-go (SDK), go-redis/v9, pgx/v5, testcontainers-go, prometheus/client_golang.

---

### Task 1: Fix SDK Imports and Build

**Files:**
- Modify: `schedule-mcp/cmd/server/tools.go`
- Modify: `schedule-mcp/cmd/server/middleware.go`
- Modify: `schedule-mcp/cmd/server/main.go`
- Modify: `schedule-mcp/cmd/server/scheduler.go`

- [ ] **Step 1: Replace imports in tools.go**
```bash
sed -i '' 's|"github.com/modelcontextprotocol/go-sdk"|"github.com/mark3labs/mcp-go/mcp"|g' schedule-mcp/cmd/server/tools.go
```

- [ ] **Step 2: Replace imports in middleware.go**
```bash
sed -i '' 's|"github.com/modelcontextprotocol/go-sdk"|"github.com/mark3labs/mcp-go/mcp"|g' schedule-mcp/cmd/server/middleware.go
```

- [ ] **Step 3: Replace imports in main.go**
```bash
sed -i '' 's|"github.com/modelcontextprotocol/go-sdk"|"github.com/mark3labs/mcp-go/mcp"|g' schedule-mcp/cmd/server/main.go
```

- [ ] **Step 4: Replace imports in scheduler.go**
```bash
sed -i '' 's|"github.com/modelcontextprotocol/go-sdk"|"github.com/mark3labs/mcp-go/mcp"|g' schedule-mcp/cmd/server/scheduler.go
```

- [ ] **Step 5: Tidy and build**
Run: `cd schedule-mcp && go mod tidy && go build ./cmd/server`
Expected: Successful compilation (no output or binary generated).

- [ ] **Step 6: Commit**
```bash
cd schedule-mcp
git add go.mod go.sum cmd/server/*.go
git commit -m "fix: update mcp-go sdk imports"
```

---

### Task 2: Implement Redis Token Bucket Rate Limiting

**Files:**
- Modify: `schedule-mcp/cmd/server/ratelimit.go`

- [ ] **Step 1: Overwrite ratelimit.go with Redis implementation**
Write the following to `schedule-mcp/cmd/server/ratelimit.go`:
```go
package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = 1

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
	tokens = capacity
	last_refill = now
end

local elapsed = now - last_refill
local refill = elapsed * rate
tokens = math.min(capacity, tokens + refill)

if tokens >= requested then
	tokens = tokens - requested
	redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
	redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
	return 1
else
	redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
	redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
	return 0
end
`)

type rateLimiter struct {
	client *redis.Client
}

func (rl *rateLimiter) Allow(userID string) bool {
	if rl.client == nil {
		// Fallback if redis is not ready
		return false
	}
	
	now := time.Now().UnixMilli()
	// rate: 5 tokens/sec = 0.005 tokens/ms
	// capacity: 10 burst
	key := "ratelimit:" + userID
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := rateLimitScript.Run(ctx, rl.client, []string{key}, 0.005, 10, now).Result()
	if err != nil {
		return false // Default deny on error
	}
	
	allowed, _ := result.(int64)
	return allowed == 1
}

func (rl *rateLimiter) cleanup() {
	// No-op: Redis EXPIRE handles cleanup
}
```

- [ ] **Step 2: Initialize globalRateLimiter client in main.go**
Since `main.go` initializes `globalRateLimiter` somewhere (currently with in-memory map), we need to ensure it uses the Redis client.
Modify `schedule-mcp/cmd/server/main.go` to set the Redis client on the rate limiter after connecting to Redis:
```bash
sed -i '' 's/globalRateLimiter = &rateLimiter{.*}/globalRateLimiter = \&rateLimiter{client: redisClient}/g' schedule-mcp/cmd/server/main.go
```
*(Note: verify `globalRateLimiter` initialization logic in `main.go` manually if `sed` fails).*

- [ ] **Step 3: Run build to verify**
Run: `cd schedule-mcp && go build ./cmd/server`
Expected: PASS

- [ ] **Step 4: Commit**
```bash
cd schedule-mcp
git add cmd/server/ratelimit.go cmd/server/main.go
git commit -m "feat: implement redis token bucket rate limiting"
```

---

### Task 3: Add Unit Tests for Scheduler

**Files:**
- Create: `schedule-mcp/cmd/server/scheduler_test.go`

- [ ] **Step 1: Write calculateNextRun unit test**
Write the following to `schedule-mcp/cmd/server/scheduler_test.go`:
```go
package main

import (
	"testing"
	"time"
)

func TestCalculateNextRun(t *testing.T) {
	// Assuming a trigger_config parsing logic exists
	// We will test standard cron parsing
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	
	// Example cron: every hour at minute 30
	cronExpr := "30 * * * *"
	
	next, err := calculateNextRun(cronExpr, now)
	if err != nil {
		t.Fatalf("Failed to calculate next run: %v", err)
	}
	
	expected := time.Date(2026, 5, 8, 12, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}
```

- [ ] **Step 2: Run unit test**
Run: `cd schedule-mcp && go test ./cmd/server -run TestCalculateNextRun -v`
Expected: PASS (or minimal code changes to `scheduler.go` to make it pass).

- [ ] **Step 3: Commit**
```bash
cd schedule-mcp
git add cmd/server/scheduler_test.go
git commit -m "test: add calculateNextRun unit test"
```

---

### Task 4: Add Observability (Health & Metrics)

**Files:**
- Modify: `schedule-mcp/cmd/server/main.go`

- [ ] **Step 1: Add prometheus dependency**
Run: `cd schedule-mcp && go get github.com/prometheus/client_golang/prometheus/promhttp`

- [ ] **Step 2: Register /healthz and /metrics handlers in main.go**
In `schedule-mcp/cmd/server/main.go`, inside the HTTP setup before `http.ListenAndServe`, add:
```go
// Add import if missing: "github.com/prometheus/client_golang/prometheus/promhttp"
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
	if dbPool == nil || redisClient == nil {
		http.Error(w, "Dependencies not initialized", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()
	
	if err := dbPool.Ping(ctx); err != nil {
		http.Error(w, "DB down", http.StatusServiceUnavailable)
		return
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		http.Error(w, "Redis down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
})
http.Handle("/metrics", promhttp.Handler())
```

- [ ] **Step 3: Build and test**
Run: `cd schedule-mcp && go build ./cmd/server`
Expected: PASS

- [ ] **Step 4: Commit**
```bash
cd schedule-mcp
git add cmd/server/main.go go.mod go.sum
git commit -m "feat: add healthz and prometheus metrics endpoints"
```
