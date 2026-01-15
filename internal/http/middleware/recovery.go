package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Recovery recovers from panics and logs the error
func Recovery(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err := fmt.Errorf("panic recovered: %v", r)

				requestID := c.Locals("request_id")
				fields := []zap.Field{
					zap.Error(err),
					zap.ByteString("stack", stack),
					zap.String("method", c.Method()),
					zap.String("path", c.Path()),
				}

				if requestID != nil {
					fields = append(fields, zap.String("request_id", requestID.(string)))
				}

				logger.Error("panic recovered", fields...)

				if c.Response().StatusCode() == 0 {
					c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Internal Server Error",
					})
				}
			}
		}()

		return c.Next()
	}
}