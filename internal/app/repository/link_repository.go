package repository

import (
	"context"
	"errors"

	"github.com/sifan077/PowerURL/internal/app/model"
	"gorm.io/gorm"
)

var (
	// ErrLinkNotFound signals that the requested short link does not exist.
	ErrLinkNotFound = errors.New("link not found")
)

// LinkRepository defines the data access contract for short links.
type LinkRepository interface {
	Create(ctx context.Context, link *model.Link) error
	GetByCode(ctx context.Context, code string) (*model.Link, error)
	List(ctx context.Context, limit, offset int) ([]model.Link, error)
	Update(ctx context.Context, link *model.Link) error
}

type linkRepository struct {
	db *gorm.DB
}

// NewLinkRepository returns a GORM-backed LinkRepository.
func NewLinkRepository(db *gorm.DB) LinkRepository {
	return &linkRepository{db: db}
}

func (r *linkRepository) Create(ctx context.Context, link *model.Link) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return err
	}
	return nil
}

func (r *linkRepository) GetByCode(ctx context.Context, code string) (*model.Link, error) {
	var link model.Link
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return &link, nil
}

func (r *linkRepository) List(ctx context.Context, limit, offset int) ([]model.Link, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var result []model.Link
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *linkRepository) Update(ctx context.Context, link *model.Link) error {
	result := r.db.WithContext(ctx).
		Model(&model.Link{}).
		Where("code = ?", link.Code).
		Updates(map[string]interface{}{
			"url":           link.URL,
			"mode":          link.Mode,
			"timer_seconds": link.TimerSeconds,
			"disabled":      link.Disabled,
			"expires_at":    link.ExpiresAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLinkNotFound
	}

	return r.db.WithContext(ctx).Where("code = ?", link.Code).First(link).Error
}
