package main

import (
	"context"
	"encoding/json"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type PubSubEvent struct {
	UserID       string            `json:"user_id"`
	EventType    string            `json:"event_type"` // e.g., "task_status_changed"
	Payload      string            `json:"payload"`
	TraceContext map[string]string `json:"trace_context,omitempty"`
}

func PublishEvent(ctx context.Context, event PubSubEvent) error {
	if event.TraceContext == nil {
		event.TraceContext = make(map[string]string)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(event.TraceContext))

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return RedisClient.Publish(ctx, "system_events", data).Err()
}

func SubscribeToEvents(ctx context.Context, onEvent func(context.Context, PubSubEvent)) {
	pubsub := RedisClient.Subscribe(ctx, "system_events")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event PubSubEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		// Extract trace context
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.TraceContext))
		_, span := otel.Tracer("scheduler-mcp").Start(parentCtx, "Redis Subscription")
		
		onEvent(parentCtx, event)
		span.End()
	}
}
