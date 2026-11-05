package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func Recovery() fiber.Handler { return recover.New() }

func RequestID() fiber.Handler { return requestid.New() }

func SecurityHeaders() fiber.Handler { return helmet.New() }
