package service

import (
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
func (p *ClickPublisher) Publish(linkCode, ip, userAgent string) error {
	event := model.ClickEvent{
		ID:        uuid.New().String(),
		LinkCode:  linkCode,
		IP:        ip,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(model.ClickStreamSubject, data)
	return err
}