package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID ensures every request has an X-Request-ID header and local context value.
// ทำให้ทุก request มี X-Request-ID ทั้งใน header และใน locals ของ Fiber
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set("X-Request-ID", requestID)
		c.Locals("request_id", requestID)

		return c.Next()
	}
}
