package main

import (
	"context"
	"encoding/json"
	"log"

	"actionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuditEvent struct {
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]interface{}
}

type AuditLogEntry struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
}

func writeAuditLog(ctx context.Context, event AuditEvent) {
	if dbPool == nil {
		return
	}
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("Failed to marshal audit log metadata for action %s: %v", event.Action, err)
		return
	}

	err = queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		UserID:       pgtype.Text{String: event.UserID, Valid: event.UserID != ""},
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   pgtype.Text{String: event.ResourceID, Valid: event.ResourceID != ""},
		Metadata:     payload,
	})
	if err != nil {
		log.Printf("Failed to write audit log for action %s: %v", event.Action, err)
	}
}
