package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"schedule-mcp/db"
)

func handleGetSystemInsights(c echo.Context) error {
	ctx := c.Request().Context()

	p99, err := queries.GetP99ExecutionLatency(ctx)
	if err != nil {
		p99 = 0
	}

	successRateRaw, err := queries.GetGlobalSuccessRate(ctx)
	if err != nil {
		successRateRaw = 100.0
	}

	workerCount, err := queries.GetActiveWorkerCount(ctx)
	if err != nil {
		workerCount = 0
	}

	trends, err := queries.GetDailyExecutionTrends(ctx)
	if err != nil {
		trends = []db.GetDailyExecutionTrendsRow{}
	}

	// Map trends to expected format
	dailyTasks := []map[string]interface{}{}
	for _, t := range trends {
		dailyTasks = append(dailyTasks, map[string]interface{}{
			"date":  t.Date,
			"count": t.Count,
		})
	}

	// Type assertion for successRateRaw which is interface{} from sqlc
	var successRate float64
	switch v := successRateRaw.(type) {
	case float64:
		successRate = v
	case float32:
		successRate = float64(v)
	case int64:
		successRate = float64(v)
	case int32:
		successRate = float64(v)
	default:
		successRate = 100.0
	}

	data := map[string]interface{}{
		"p99_latency":    int64(p99),
		"success_rate":   successRate,
		"active_workers": workerCount,
		"daily_tasks":    dailyTasks,
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}

func handleGetTrends(c echo.Context) error {
	ctx := c.Request().Context()

	// 1. Total Tasks Trends (Current 30d vs Previous 30d)
	var currentTasks, prevTasks int64
	_ = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM execution_traces WHERE start_time > NOW() - INTERVAL '30 days'").Scan(&currentTasks)
	_ = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM execution_traces WHERE start_time > NOW() - INTERVAL '60 days' AND start_time <= NOW() - INTERVAL '30 days'").Scan(&prevTasks)

	// 2. Success Rate Trends
	var currentSuccess, prevSuccess float64
	_ = dbPool.QueryRow(ctx, `
		SELECT COALESCE((COUNT(*) FILTER (WHERE is_error = FALSE)::float / NULLIF(COUNT(*), 0)::float) * 100, 100)
		FROM execution_traces WHERE start_time > NOW() - INTERVAL '30 days'
	`).Scan(&currentSuccess)
	_ = dbPool.QueryRow(ctx, `
		SELECT COALESCE((COUNT(*) FILTER (WHERE is_error = FALSE)::float / NULLIF(COUNT(*), 0)::float) * 100, 100)
		FROM execution_traces WHERE start_time > NOW() - INTERVAL '60 days' AND start_time <= NOW() - INTERVAL '30 days'
	`).Scan(&prevSuccess)

	// 3. User Growth
	var currentUsers, prevUsers int64
	_ = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '30 days'").Scan(&currentUsers)
	_ = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '60 days' AND created_at <= NOW() - INTERVAL '30 days'").Scan(&prevUsers)

	calcGrowth := func(curr, prev float64) string {
		if prev == 0 {
			if curr > 0 {
				return "+100%"
			}
			return "0%"
		}
		growth := ((curr - prev) / prev) * 100
		if growth >= 0 {
			return fmt.Sprintf("+%.1f%%", growth)
		}
		return fmt.Sprintf("%.1f%%", growth)
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"tasks_growth":   calcGrowth(float64(currentTasks), float64(prevTasks)),
			"success_growth": calcGrowth(currentSuccess, prevSuccess),
			"users_growth":   calcGrowth(float64(currentUsers), float64(prevUsers)),
		},
	})
}

func handleGetWorkers(c echo.Context) error {
	ctx := c.Request().Context()
	workers, err := queries.ListWorkerHeartbeats(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch workers"})
	}

	type workerInfo struct {
		WorkerID      string    `json:"worker_id"`
		Hostname      string    `json:"hostname"`
		LastHeartbeat time.Time `json:"last_heartbeat"`
		Status        string    `json:"status"`
		TaskCount     int32     `json:"task_count"`
	}

	var data []workerInfo
	if workers == nil {
		data = []workerInfo{}
	}
	now := time.Now().UTC()
	for _, w := range workers {
		status := "offline"
		if w.LastHeartbeat.Valid && w.LastHeartbeat.Time.After(now.Add(-2*time.Minute)) {
			status = "online"
		}

		data = append(data, workerInfo{
			WorkerID:      w.WorkerID,
			Hostname:      w.Hostname.String,
			LastHeartbeat: w.LastHeartbeat.Time,
			Status:        status,
			TaskCount:     w.TaskCount.Int32,
		})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}
