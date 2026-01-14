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
	bloomKey         = "bloom:links"
	bloomSize        = 1000000
	bloomHashes      = 7
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
	db              *gorm.DB
	redis           *redis.Client
	bloomSupported  bool
}

// NewLinkRepository returns a GORM-backed LinkRepository with Redis caching and bloom filter.
func NewLinkRepository(db *gorm.DB, redis *redis.Client) LinkRepository {
	repo := &linkRepository{
		db:    db,
		redis: redis,
	}
	repo.initBloomFilter(context.Background())
	return repo
}

func (r *linkRepository) initBloomFilter(ctx context.Context) {
	if r.redis == nil {
		r.bloomSupported = false
		return
	}
	exists, _ := r.redis.Exists(ctx, bloomKey).Result()
	if exists == 0 {
		err := r.redis.Do(ctx, "BF.RESERVE", bloomKey, bloomSize, bloomHashes).Err()
		r.bloomSupported = err == nil
	} else {
		r.bloomSupported = true
	}
}

func (r *linkRepository) bloomAdd(ctx context.Context, code string) {
	if !r.bloomSupported || r.redis == nil {
		return
	}
	r.redis.Do(ctx, "BF.ADD", bloomKey, code)
}

func (r *linkRepository) bloomExists(ctx context.Context, code string) bool {
	if !r.bloomSupported || r.redis == nil {
		return true
	}
	result, err := r.redis.Do(ctx, "BF.EXISTS", bloomKey, code).Int()
	if err != nil {
		return true
	}
	return result == 1
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