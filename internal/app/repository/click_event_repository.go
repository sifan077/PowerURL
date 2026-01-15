package repository

import (
	"context"

	"github.com/sifan077/PowerURL/internal/app/model"
	"gorm.io/gorm"
)

// ClickEventRepository defines the data access contract for click events.
type ClickEventRepository interface {
	Create(ctx context.Context, event *model.ClickEvent) error
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