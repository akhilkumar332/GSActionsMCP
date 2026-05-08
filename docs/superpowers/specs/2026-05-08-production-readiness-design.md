# Production Readiness Design

**Goal**: Prepare the `schedule-mcp` project for production deployment by fixing the build, introducing Redis-based rate limiting, setting up testcontainers for integration tests, and adding observability endpoints.

## 1. Build and SDK Fixes
- **Problem**: The current `go.mod` relies on a broken commit of `github.com/modelcontextprotocol/go-sdk`.
- **Solution**: 
  - Update `go.mod` to use `github.com/mark3labs/mcp-go@latest`.
  - Replace all imports of `github.com/modelcontextprotocol/go-sdk` across `cmd/server/` files (`main.go`, `tools.go`, `middleware.go`, `scheduler.go`) with `github.com/mark3labs/mcp-go/mcp` or `github.com/mark3labs/mcp-go/server` as appropriate.

## 2. Redis Rate Limiting (Token Bucket)
- **Problem**: Rate limiting in `ratelimit.go` is currently in-memory, which doesn't scale across multiple Docker containers.
- **Solution**:
  - Introduce a Lua script-based Token Bucket rate limiter in `ratelimit.go`.
  - Use the existing Redis connection pool from `go-redis/v9`.
  - The Lua script will ensure atomic token deduction and refilling based on client IP or user ID.

## 3. Automated Testing (Testcontainers)
- **Problem**: The system lacks test coverage, especially for concurrent database operations.
- **Solution**:
  - Introduce `github.com/testcontainers/testcontainers-go` for spawning real PostgreSQL and Redis instances during testing.
  - **Unit Tests**: Add tests for `calculateNextRun` in `scheduler_test.go` to ensure cron trigger logic is sound.
  - **Integration Tests**: Add tests in `scheduler_db_test.go` to verify the `FOR UPDATE SKIP LOCKED` functionality of `claimDueTasks` across simulated concurrent workers.

## 4. Observability
- **Problem**: Lack of insight into system health and metrics.
- **Solution**:
  - Add a `/healthz` endpoint returning HTTP 200 after validating PostgreSQL and Redis connectivity (`dbPool.Ping` and `redisClient.Ping`).
  - Add a `/metrics` endpoint using `github.com/prometheus/client_golang/prometheus/promhttp`.
  - Register basic metrics: Total scheduled tasks, active SSE connections, and HTTP request rates.
