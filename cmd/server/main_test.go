package main

import (
	"os"
	"testing"

	"schedule-mcp/db"
)

func TestMain(m *testing.M) {
	queries = db.New(&mockDB{})
	os.Exit(m.Run())
}
