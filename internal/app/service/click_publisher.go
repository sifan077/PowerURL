package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/sifan077/PowerURL/internal/app/model"
)

// ClickPublisher publishes click events to NATS JetStream
type ClickPublisher struct {
	js nats.JetStreamContext
}

// NewClickPublisher creates a new click event publisher
func NewClickPublisher(js nats.JetStreamContext) *ClickPublisher {
	return &ClickPublisher{js: js}
}

// Publish publishes a click event to the stream
func (p *ClickPublisher) Publish(linkCode, ip, userAgent, status, clickID string) error {
	return p.PublishWithContext(context.Background(), linkCode, ip, userAgent, status, clickID)
}

// PublishWithContext publishes a click event to the stream with context timeout
func (p *ClickPublisher) PublishWithContext(ctx context.Context, linkCode, ip, userAgent, status, clickID string) error {
	eventID := clickID
	if eventID == "" {
		eventID = uuid.New().String()
	}
	event := model.ClickEvent{
		ID:        eventID,
		LinkCode:  linkCode,
		IP:        ip,
		UserAgent: userAgent,
		Status:    status,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Publish is synchronous and waits for ACK
	_, err = p.js.Publish(model.ClickStreamSubject, data)
	if err != nil {
		return err
	}

	return nil
}