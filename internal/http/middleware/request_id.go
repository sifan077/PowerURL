package middleware

import (
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
)

const RequestIDHeader = "X-Request-ID"

// RequestID generates a unique request ID for each request
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := c.Get(RequestIDHeader)
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set(RequestIDHeader, rid)
		c.Locals("request_id", rid)
		return c.Next()
	}
}