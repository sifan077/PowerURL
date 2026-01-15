package repository

import (
	"context"
	"time"

	"github.com/sifan077/PowerURL/internal/app/model"
	"gorm.io/gorm"
)

// ClickEventRepository defines the data access contract for click events.
type ClickEventRepository interface {
	Create(ctx context.Context, event *model.ClickEvent) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateExpiredPendingStatus(ctx context.Context, expiredBefore time.Time) (int64, error)
}

type clickEventRepository struct {
	db *gorm.DB
}

// NewClickEventRepository returns a GORM-backed ClickEventRepository.
func NewClickEventRepository(db *gorm.DB) ClickEventRepository {
	return &clickEventRepository{db: db}
}

func (r *clickEventRepository) Create(ctx context.Context, event *model.ClickEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *clickEventRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&model.ClickEvent{}).Where("id = ?", id).Update("status", status).Error
}

func (r *clickEventRepository) UpdateExpiredPendingStatus(ctx context.Context, expiredBefore time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.ClickEvent{}).
		Where("status = ? AND timestamp < ?", model.ClickStatusPending, expiredBefore).
		Update("status", model.ClickStatusFailed)
	return result.RowsAffected, result.Error
}