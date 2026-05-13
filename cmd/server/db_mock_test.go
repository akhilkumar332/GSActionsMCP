package main

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockDB struct{}

func (m *mockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return &mockRows{}, nil
}

func (m *mockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return &mockRow{}
}

type mockRows struct {
	pgx.Rows
}

func (m *mockRows) Close()     {}
func (m *mockRows) Next() bool { return false }
func (m *mockRows) Err() error { return nil }

type mockRow struct {
	pgx.Row
}

func (m *mockRow) Scan(dest ...interface{}) error { return nil }
