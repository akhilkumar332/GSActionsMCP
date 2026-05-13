package main

const (
	StatusActive     = "active"
	StatusPaused     = "paused"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusError      = "error"

	TierFree = "free"
	TierPlus = "plus"
	TierPro  = "pro"

	QuotaFree = 2
	QuotaPlus = 10
	QuotaPro  = 20

	PolicySkip         = "skip"
	PolicyRunImmediate = "run_immediately"

	TriggerCron     = "cron"
	TriggerInterval = "interval"
	TriggerDate     = "date"
)
