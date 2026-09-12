package handler

import "github.com/gofiber/fiber/v2"

// Health handles GET /healthz.
func Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
