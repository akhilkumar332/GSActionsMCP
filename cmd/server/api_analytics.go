package main

import (
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
