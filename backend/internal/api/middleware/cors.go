package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CORS(origins []string) fiber.Handler {
	allowed := strings.Join(origins, ",")
	return cors.New(cors.Config{
		AllowOrigins:     allowed,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowCredentials: true,
	})
}
