package model

import "time"

// ClickEvent represents a click event on a short link
type ClickEvent struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	LinkCode  string    `json:"link_code" gorm:"size:32;not null;index"`
	IP        string    `json:"ip" gorm:"size:64;not null"`
	UserAgent string    `json:"user_agent" gorm:"type:text"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`
}

const (
	ClickStreamName     = "CLICKS"
	ClickStreamSubject  = "clicks.events"
	ClickConsumerName   = "click-logger"
	ClickConsumerGroup  = "click-loggers"
	ClickStreamMaxBytes = 1024 * 1024 * 100 // 100MB
)