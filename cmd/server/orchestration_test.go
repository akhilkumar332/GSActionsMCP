package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"actionfy/db"
	"testing"
)

type flexibleMockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *flexibleMockRow) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

type flexibleMockDB struct {
	queryRowFunc func(ctx context.Context, query string, args ...interface{}) *flexibleMockRow
}

func (m *flexibleMockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *flexibleMockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (m *flexibleMockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return m.queryRowFunc(ctx, query, args...)
}

// Implement DBTX for flexibleMockDB
// We need to match the signature of DBTX in db.go exactly.
// type DBTX interface {
// 	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
// 	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
// 	QueryRow(context.Context, string, ...interface{}) pgx.Row
// }

func TestResolvePromptVariables(t *testing.T) {
	// Backup original queries
	oldQueries := queries
	defer func() { queries = oldQueries }()

	mock := &flexibleMockDB{}
	queries = db.New(&dbTXWrapper{mock})

	ctx := context.Background()
	userID := "user-123"

	t.Run("Valid variable replacement", func(t *testing.T) {
		taskID := "550e8400-e29b-41d4-a716-446655440000"
		expectedOutput := "The final result"

		mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) *flexibleMockRow {
			// Verify that userID is passed as the second argument (args[1])
			if len(args) < 2 {
				t.Errorf("Expected at least 2 arguments to GetTaskOutput, got %d", len(args))
			} else if args[1] != userID {
				t.Errorf("Expected userID %q, got %q", userID, args[1])
			}

			return &flexibleMockRow{
				scanFunc: func(dest ...interface{}) error {
					if len(dest) > 0 {
						if d, ok := dest[0].(*pgtype.Text); ok {
							*d = pgtype.Text{String: expectedOutput, Valid: true}
						}
					}
					return nil
				},
			}
		}

		prompt := "Result was: {{task." + taskID + ".output}}"
		resolved := resolvePromptVariables(ctx, userID, prompt, nil, nil)
		expected := "Result was: " + expectedOutput

		if resolved != expected {
			t.Errorf("Expected %q, got %q", expected, resolved)
		}
	})

	t.Run("Invalid UUID format", func(t *testing.T) {
		prompt := "Result was: {{task.invalid-uuid.output}}"
		resolved := resolvePromptVariables(ctx, userID, prompt, nil, nil)
		if resolved != prompt {
			t.Errorf("Expected no change for invalid UUID, got %q", resolved)
		}
	})

	t.Run("Multiple variables", func(t *testing.T) {
		task1ID := "550e8400-e29b-41d4-a716-446655440001"
		task2ID := "550e8400-e29b-41d4-a716-446655440002"

		outputs := map[string]string{
			task1ID: "Output 1",
			task2ID: "Output 2",
		}

		mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) *flexibleMockRow {
			tid := args[0].(pgtype.UUID)
			tidStr := formatUUID(tid)
			return &flexibleMockRow{
				scanFunc: func(dest ...interface{}) error {
					if d, ok := dest[0].(*pgtype.Text); ok {
						*d = pgtype.Text{String: outputs[tidStr], Valid: true}
					}
					return nil
				},
			}
		}

		prompt := "{{task." + task1ID + ".output}} and {{task." + task2ID + ".output}}"
		resolved := resolvePromptVariables(ctx, userID, prompt, nil, nil)
		expected := "Output 1 and Output 2"

		if resolved != expected {
			t.Errorf("Expected %q, got %q", expected, resolved)
		}
	})
}

func TestEvaluateBranchCondition(t *testing.T) {
	t.Run("Contains match", func(t *testing.T) {
		cond := map[string]string{
			"if":    "contains",
			"value": "SUCCESS",
		}
		condBytes, _ := json.Marshal(cond)
		if !evaluateBranchCondition(condBytes, "The status is SUCCESS") {
			t.Error("Expected true for matching contains condition")
		}
	})

	t.Run("Contains no match", func(t *testing.T) {
		cond := map[string]string{
			"if":    "contains",
			"value": "SUCCESS",
		}
		condBytes, _ := json.Marshal(cond)
		if evaluateBranchCondition(condBytes, "The status is FAILURE") {
			t.Error("Expected false for non-matching contains condition")
		}
	})

	t.Run("Empty condition", func(t *testing.T) {
		if !evaluateBranchCondition(nil, "any output") {
			t.Error("Expected true for nil condition")
		}
	})

	t.Run("Invalid JSON condition", func(t *testing.T) {
		if !evaluateBranchCondition([]byte("invalid json"), "any output") {
			t.Error("Expected true (default) for invalid JSON condition")
		}
	})
}

// Helper to wrap our mock to satisfy DBTX interface exactly
type dbTXWrapper struct {
	*flexibleMockDB
}

func (w *dbTXWrapper) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return w.flexibleMockDB.Exec(ctx, query, args...)
}
func (w *dbTXWrapper) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return w.flexibleMockDB.Query(ctx, query, args...)
}
func (w *dbTXWrapper) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return w.flexibleMockDB.QueryRow(ctx, query, args...)
}
