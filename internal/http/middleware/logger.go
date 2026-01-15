package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Logger creates a logging middleware using zap
func Logger(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		requestID := c.Locals("request_id")

		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", duration),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if requestID != nil {
			fields = append(fields, zap.String("request_id", requestID.(string)))
		}

		if err != nil {
			logger.Error("request error", append(fields, zap.Error(err))...)
		} else {
			logger.Info("request", fields...)
		}

		return err
	}
}