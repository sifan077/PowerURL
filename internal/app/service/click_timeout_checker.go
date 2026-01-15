package service

import (
	"context"
	"time"

	apprepository "github.com/sifan077/PowerURL/internal/app/repository"
	"go.uber.org/zap"
)

// ClickTimeoutChecker periodically checks for expired pending click events and marks them as failed.
type ClickTimeoutChecker struct {
	logger      *zap.Logger
	repo        apprepository.ClickEventRepository
	ttl         time.Duration
	interval    time.Duration
	stopChan    chan struct{}
}

// NewClickTimeoutChecker creates a new click timeout checker.
func NewClickTimeoutChecker(logger *zap.Logger, repo apprepository.ClickEventRepository, ttl time.Duration) *ClickTimeoutChecker {
	return &ClickTimeoutChecker{
		logger:   logger,
		repo:     repo,
		ttl:      ttl,
		interval: 30 * time.Second, // Check every 30 seconds
		stopChan: make(chan struct{}),
	}
}

// Start begins the periodic checking for expired pending click events.
func (c *ClickTimeoutChecker) Start() {
	go c.run()
}

// Stop stops the periodic checking.
func (c *ClickTimeoutChecker) Stop() {
	close(c.stopChan)
}

func (c *ClickTimeoutChecker) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkExpiredPendingEvents()
		case <-c.stopChan:
			c.logger.Info("click timeout checker stopped")
			return
		}
	}
}

func (c *ClickTimeoutChecker) checkExpiredPendingEvents() {
	ctx := context.Background()
	expiredBefore := time.Now().Add(-c.ttl)

	affected, err := c.repo.UpdateExpiredPendingStatus(ctx, expiredBefore)
	if err != nil {
		c.logger.Error("failed to update expired pending click events", zap.Error(err))
		return
	}

	if affected > 0 {
		c.logger.Info("updated expired pending click events to failed",
			zap.Int64("count", affected),
			zap.Time("expired_before", expiredBefore),
		)
	}
}