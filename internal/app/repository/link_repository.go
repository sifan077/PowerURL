package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sifan077/PowerURL/internal/app/model"
	"gorm.io/gorm"
)

var (
	// ErrLinkNotFound signals that the requested short link does not exist.
	ErrLinkNotFound = errors.New("link not found")
)

const (
	cacheKeyPrefix   = "link:"
	cacheTTL         = 1 * time.Hour
	cacheNullTTL     = 5 * time.Minute
	linkSetKey       = "links:exists"
	cacheNullValue   = "NULL"
)

// LinkRepository defines the data access contract for short links.
type LinkRepository interface {
	Create(ctx context.Context, link *model.Link) error
	GetByCode(ctx context.Context, code string) (*model.Link, error)
	List(ctx context.Context, limit, offset int) ([]model.Link, error)
	Update(ctx context.Context, link *model.Link) error
}

type linkRepository struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewLinkRepository returns a GORM-backed LinkRepository with Redis caching.
func NewLinkRepository(db *gorm.DB, redis *redis.Client) LinkRepository {
	return &linkRepository{
		db:    db,
		redis: redis,
	}
}

func (r *linkRepository) bloomAdd(ctx context.Context, code string) {
	if r.redis == nil {
		return
	}
	r.redis.SAdd(ctx, linkSetKey, code)
}

func (r *linkRepository) bloomExists(ctx context.Context, code string) bool {
	if r.redis == nil {
		return true
	}
	result, err := r.redis.SIsMember(ctx, linkSetKey, code).Result()
	if err != nil {
		return true
	}
	return result
}

func (r *linkRepository) Create(ctx context.Context, link *model.Link) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return err
	}
	r.bloomAdd(ctx, link.Code)
	return nil
}

func (r *linkRepository) GetByCode(ctx context.Context, code string) (*model.Link, error) {
	if !r.bloomExists(ctx, code) {
		return nil, ErrLinkNotFound
	}

	cacheKey := cacheKeyPrefix + code

	if r.redis != nil {
		cached, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			if cached == cacheNullValue {
				return nil, ErrLinkNotFound
			}
			var link model.Link
			if err := json.Unmarshal([]byte(cached), &link); err == nil {
				return &link, nil
			}
		}
	}

	var link model.Link
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if r.redis != nil {
				r.redis.Set(ctx, cacheKey, cacheNullValue, cacheNullTTL)
			}
			return nil, ErrLinkNotFound
		}
		return nil, err
	}

	if r.redis != nil {
		data, err := json.Marshal(link)
		if err == nil {
			r.redis.Set(ctx, cacheKey, data, cacheTTL)
		}
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

	if err := r.db.WithContext(ctx).Where("code = ?", link.Code).First(link).Error; err != nil {
		return err
	}

	if r.redis != nil {
		cacheKey := cacheKeyPrefix + link.Code
		r.redis.Del(ctx, cacheKey)
	}

	return nil
}