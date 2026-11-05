package handlers

import "github.com/gofiber/fiber/v2"

func NotImplemented(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"data":  nil,
		"error": fiber.Map{"code": "NOT_IMPLEMENTED", "message": "endpoint is not implemented"},
	})
}
