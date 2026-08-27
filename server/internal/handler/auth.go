package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func BearerAuth(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if token == "" {
			return c.Next()
		}
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or invalid Authorization header",
			})
		}
		if strings.TrimPrefix(auth, "Bearer ") != token {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}
		return c.Next()
	}
}
