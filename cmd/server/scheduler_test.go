package main

import (
	"testing"
	"time"
)

func TestCalculateNextRun(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	
	// Example 1: every hour at minute 30
	cronExpr := "30 * * * *"
	
	next, err := calculateNextRun("cron", map[string]interface{}{"cron": cronExpr}, now)
	if err != nil {
		t.Fatalf("Failed to calculate next run: %v", err)
	}
	
	expected := time.Date(2026, 5, 8, 12, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
    
    // Example 2: intervals (5 minutes)
    next, err = calculateNextRun("interval", map[string]interface{}{"minutes": float64(5)}, now)
    if err != nil {
		t.Fatalf("Failed to calculate next run: %v", err)
	}
    expected = time.Date(2026, 5, 8, 12, 5, 0, 0, time.UTC)
    if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}
