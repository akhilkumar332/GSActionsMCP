package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"actionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type WebhookSubscription struct {
	ID          string    `json:"id"`
	EndpointURL string    `json:"endpoint_url"`
	EventTypes  []string  `json:"event_types"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type WebhookDelivery struct {
	ID           string    `json:"id"`
	WebhookID    string    `json:"webhook_id"`
	EventType    string    `json:"event_type"`
	StatusCode   *int32    `json:"status_code,omitempty"`
	Success      bool      `json:"success"`
	ResponseBody *string   `json:"response_body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type WebhookCreateInput struct {
	EndpointURL   string   `json:"endpoint_url"`
	EventTypes    []string `json:"event_types"`
	SigningSecret string   `json:"signing_secret"`
}

func handleSystemEvent(ctx context.Context, event PubSubEvent) {
	log.Printf("Received event for user %s: %s", event.UserID, event.EventType)
	dispatchOutboundWebhooks(ctx, event)
}

func dispatchOutboundWebhooks(ctx context.Context, event PubSubEvent) {
	if event.UserID == "" || queries == nil {
		return
	}

	rows, err := queries.ListActiveOutboundWebhooks(ctx, event.UserID)
	if err != nil {
		log.Printf("Failed to query outbound webhooks for user %s: %v", event.UserID, err)
		return
	}

	for _, row := range rows {
		var eventTypes []string
		if err := json.Unmarshal(row.EventTypes, &eventTypes); err != nil {
			log.Printf("Failed to decode webhook event types for %s: %v", formatUUID(row.ID), err)
			continue
		}
		if !webhookInterestedInEvent(eventTypes, event.EventType) {
			continue
		}

		secret, err := Decrypt(row.EncryptedSigningSecret)
		if err != nil {
			log.Printf("Failed to decrypt webhook signing secret for %s: %v", formatUUID(row.ID), err)
			continue
		}

		webhookID := formatUUID(row.ID)
		endpointURL := row.EndpointUrl

		workerWG.Add(1)
		go func(webhookID, endpointURL string, secret []byte) {
			defer workerWG.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic recovered in webhook delivery worker: %v", r)
				}
			}()
			deliverCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := deliverWebhookEvent(deliverCtx, webhookID, event, endpointURL, string(secret)); err != nil {
				log.Printf("Webhook delivery failed for %s: %v", webhookID, err)
			}
		}(webhookID, endpointURL, secret)
	}
}

func webhookInterestedInEvent(eventTypes []string, eventType string) bool {
	for _, candidate := range eventTypes {
		if candidate == "*" || candidate == eventType {
			return true
		}
	}
	return false
}

func deliverWebhookEvent(ctx context.Context, webhookID string, event PubSubEvent, endpointURL string, signingSecret string) error {
	payloadBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := signWebhookPayload(timestamp, payloadBytes, signingSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Schedule-Event", event.EventType)
	req.Header.Set("X-Schedule-Timestamp", timestamp)
	req.Header.Set("X-Schedule-Signature", signature)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		recordWebhookDelivery(ctx, webhookID, event.UserID, event.EventType, nil, false, err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	statusCode := int32(resp.StatusCode)
	recordWebhookDelivery(ctx, webhookID, event.UserID, event.EventType, &statusCode, success, string(body))
	if !success {
		return fmt.Errorf("non-2xx status %d", resp.StatusCode)
	}
	return nil
}

func signWebhookPayload(timestamp string, body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func generateSigningSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func recordWebhookDelivery(ctx context.Context, webhookID, userID, eventType string, statusCode *int32, success bool, responseBody string) {
	if queries == nil {
		return
	}
	var wid pgtype.UUID
	if err := parseUUID(webhookID, &wid); err != nil {
		log.Printf("Failed to parse webhook ID for delivery recording: %v", err)
		return
	}

	var sc pgtype.Int4
	if statusCode != nil {
		sc = pgtype.Int4{Int32: *statusCode, Valid: true}
	}

	err := queries.CreateWebhookDelivery(ctx, db.CreateWebhookDeliveryParams{
		WebhookID:    wid,
		UserID:       userID,
		EventType:    eventType,
		StatusCode:   sc,
		Success:      success,
		ResponseBody: pgtype.Text{String: responseBody, Valid: responseBody != ""},
	})
	if err != nil {
		log.Printf("Failed to record webhook delivery for %s: %v", webhookID, err)
	}
}
