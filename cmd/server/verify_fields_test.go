package main

import (
	"context"
	"testing"
	"schedule-mcp/db"
)

func TestListUserTasksResultFields(t *testing.T) {
	// This is a compile-time check mostly, but also verifies we can access the fields.
	var row db.ListUserTasksRow
	_ = row.AgentPrompt
	_ = row.VersionCount
}

type dummyDB struct {
	db.DBTX
}

func (d *dummyDB) Query(ctx context.Context, query string, args ...interface{}) (any, error) {
	return nil, nil
}

// We don't really need to run it if it compiles, as sqlc generation is trusted.
// But we want to ensure the API handler is using it correctly.
