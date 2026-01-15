package model

import "time"

// ClickEvent represents a click event on a short link
type ClickEvent struct {
	ID        string    `json:"id"`
	LinkCode  string    `json:"link_code"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	ClickStreamName     = "CLICKS"
	ClickStreamSubject  = "clicks.events"
	ClickConsumerName   = "click-logger"
	ClickConsumerGroup  = "click-loggers"
	ClickStreamMaxBytes = 1024 * 1024 * 100 // 100MB
)