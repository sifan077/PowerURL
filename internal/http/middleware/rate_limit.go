package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
	KeyPrefix   string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: 100,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit",
	}
}

// RateLimit creates a rate limiting middleware using Redis
func RateLimit(redisClient *redis.Client, config RateLimitConfig, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		ip := c.IP()
		key := config.KeyPrefix + ":" + ip

		// Increment the counter
		result, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			logger.Error("rate limit redis error", zap.Error(err))
			// Fail open: allow request if Redis is unavailable
			return c.Next()
		}

		// Set expiration on first request
		if result == 1 {
			redisClient.Expire(ctx, key, config.Window)
		}

		remaining := config.MaxRequests - int(result)
		c.Set("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(max(0, remaining)))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		if result > int64(config.MaxRequests) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		}

		return c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}