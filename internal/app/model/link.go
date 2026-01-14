package model

import "time"

// Link describes the core short-link entity stored in Postgres.
type Link struct {
	Code         string     `db:"code" gorm:"primaryKey;size:32"`
	URL          string     `db:"url" gorm:"type:text;not null"`
	Mode         string     `db:"mode" gorm:"size:16;not null;default:direct"`
	TimerSeconds int        `db:"timer_seconds" gorm:"not null;default:0"`
	Disabled     bool       `db:"disabled" gorm:"not null;default:false"`
	ExpiresAt    *time.Time `db:"expires_at" gorm:"index"`
	CreatedAt    time.Time  `db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `db:"updated_at" gorm:"autoUpdateTime"`
}
