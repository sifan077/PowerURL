package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sifan077/PowerURL/internal/app/model"
	apprepository "github.com/sifan077/PowerURL/internal/app/repository"
	"go.uber.org/zap"
)

// ClickConsumer consumes click events from NATS JetStream
type ClickConsumer struct {
	js       nats.JetStreamContext
	logger   *zap.Logger
	repo     apprepository.ClickEventRepository
}

// NewClickConsumer creates a new click event consumer
func NewClickConsumer(js nats.JetStreamContext, logger *zap.Logger, repo apprepository.ClickEventRepository) *ClickConsumer {
	return &ClickConsumer{js: js, logger: logger, repo: repo}
}

// Start begins consuming click events
func (c *ClickConsumer) Start() error {
	// Create stream if not exists
	_, err := c.js.StreamInfo(model.ClickStreamName)
	if err != nil {
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:     model.ClickStreamName,
			Subjects: []string{model.ClickStreamSubject},
			MaxBytes: model.ClickStreamMaxBytes,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
	}

	// Create consumer if not exists
	_, err = c.js.ConsumerInfo(model.ClickStreamName, model.ClickConsumerName)
	if err != nil {
		_, err = c.js.AddConsumer(model.ClickStreamName, &nats.ConsumerConfig{
			Durable:   model.ClickConsumerName,
			AckPolicy: nats.AckExplicitPolicy,
		})
		if err != nil {
			return fmt.Errorf("failed to create consumer: %w", err)
		}
	}

	// Subscribe to consume messages
	sub, err := c.js.PullSubscribe(model.ClickStreamSubject, model.ClickConsumerName)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	go c.consume(sub)
	return nil
}

func (c *ClickConsumer) consume(sub *nats.Subscription) {
	ctx := context.Background()
	for {
		msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
		if err != nil && err != nats.ErrTimeout {
			c.logger.Error("failed to fetch messages", zap.Error(err))
			continue
		}

		for _, msg := range msgs {
			var event model.ClickEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				c.logger.Error("failed to unmarshal click event", zap.Error(err))
				msg.Nak()
				continue
			}

			// Store the click event to database
			if err := c.repo.Create(ctx, &event); err != nil {
				c.logger.Error("failed to store click event",
					zap.String("id", event.ID),
					zap.String("link_code", event.LinkCode),
					zap.Error(err))
				msg.Nak()
				continue
			}

			c.logger.Debug("click event stored",
				zap.String("id", event.ID),
				zap.String("link_code", event.LinkCode),
				zap.String("ip", event.IP),
				zap.Time("timestamp", event.Timestamp),
			)

			msg.Ack()
		}
	}
}