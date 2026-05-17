package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"actionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
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
	now := time.Now().UTC()
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sixtyDaysAgo := now.Add(-60 * 24 * time.Hour)

	// 1. Total Tasks Trends
	currentTasks, err := queries.GetCountTracesAfter(ctx, pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true})
	if err != nil {
		log.Printf("Trends: failed to fetch current tasks: %v", err)
	}
	prevTasks, err := queries.GetCountTracesBetween(ctx, db.GetCountTracesBetweenParams{
		StartTime: pgtype.Timestamptz{Time: sixtyDaysAgo, Valid: true},
		EndTime:   pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true},
	})
	if err != nil {
		log.Printf("Trends: failed to fetch prev tasks: %v", err)
	}

	// 2. Success Rate Trends
	currentSuccess, err := queries.GetSuccessRateAfter(ctx, pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true})
	if err != nil {
		log.Printf("Trends: failed to fetch current success: %v", err)
		currentSuccess = 100.0
	}
	prevSuccess, err := queries.GetSuccessRateBetween(ctx, db.GetSuccessRateBetweenParams{
		StartTime: pgtype.Timestamptz{Time: sixtyDaysAgo, Valid: true},
		EndTime:   pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true},
	})
	if err != nil {
		log.Printf("Trends: failed to fetch prev success: %v", err)
		prevSuccess = 100.0
	}

	// 3. User Growth
	currentUsers, err := queries.GetCountUsersAfter(ctx, pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true})
	if err != nil {
		log.Printf("Trends: failed to fetch current users: %v", err)
	}
	prevUsers, err := queries.GetCountUsersBetween(ctx, db.GetCountUsersBetweenParams{
		CreatedAt:   pgtype.Timestamptz{Time: sixtyDaysAgo, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: thirtyDaysAgo, Valid: true},
	})
	if err != nil {
		log.Printf("Trends: failed to fetch prev users: %v", err)
	}

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
