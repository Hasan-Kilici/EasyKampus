package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"lms/backend/internal/api/middleware"
	"lms/backend/pkg/domain/users"
)

func pageParams(c *fiber.Ctx) (int, int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func queryUUID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	value := c.Query(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, fiber.ErrBadRequest
	}
	return &parsed, nil
}

func paramUUID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	value := c.Params(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, fiber.ErrBadRequest
	}
	return &parsed, nil
}

func queryBool(c *fiber.Ctx, name string) bool {
	value, _ := strconv.ParseBool(c.Query(name))
	return value
}

func queryDate(c *fiber.Ctx, name string) (*string, error) {
	value := c.Query(name)
	if value == "" {
		return nil, nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return nil, fiber.ErrBadRequest
	}
	return &value, nil
}

func queryTime(c *fiber.Ctx, name string) (*time.Time, error) {
	value := c.Query(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fiber.ErrBadRequest
	}
	return &parsed, nil
}

func queryDay(c *fiber.Ctx, name string) (*int, error) {
	value := c.Query(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fiber.ErrBadRequest
	}
	return &parsed, nil
}

func identity(c *fiber.Ctx) (uuid.UUID, users.Role, error) {
	userID, ok := c.Locals(middleware.UserIDLocal).(uuid.UUID)
	if !ok {
		return uuid.Nil, "", fiber.ErrUnauthorized
	}
	role, ok := c.Locals(middleware.RoleLocal).(users.Role)
	if !ok {
		return uuid.Nil, "", fiber.ErrUnauthorized
	}
	return userID, role, nil
}

func studentIdentity(c *fiber.Ctx) (uuid.UUID, error) {
	subject, err := subjectIdentity(c, "studentID")
	if err != nil {
		return uuid.Nil, err
	}
	return subject, nil
}

func teacherIdentity(c *fiber.Ctx) (uuid.UUID, error) {
	return subjectIdentity(c, "teacherID")
}

func subjectIdentity(c *fiber.Ctx, param string) (uuid.UUID, error) {
	raw := c.Params(param)
	if raw == "" {
		userID, _, err := identity(c)
		return userID, err
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fiber.ErrNotFound
	}
	return parsed, nil
}

func bodyParser[T any](c *fiber.Ctx) (T, error) {
	var input T
	if err := c.BodyParser(&input); err != nil {
		return input, fiber.ErrBadRequest
	}
	return input, nil
}

const maxBatchItems = 500

// batchParser tek kayıt gövdesini, {"<key>": [...]} sarmalayıcısını veya
// doğrudan bir dizi kabul eder.
func batchParser[T any](c *fiber.Ctx, key string) ([]T, error) {
	body := c.Body()
	if len(body) == 0 {
		return nil, fiber.ErrBadRequest
	}
	var items []T
	if json.Unmarshal(body, &items) == nil {
		return limitBatch(items)
	}
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(body, &wrapper) == nil {
		if raw, found := wrapper[key]; found {
			if json.Unmarshal(raw, &items) == nil {
				return limitBatch(items)
			}
		}
	}
	var single T
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fiber.ErrBadRequest
	}
	return []T{single}, nil
}

func limitBatch[T any](items []T) ([]T, error) {
	if len(items) > maxBatchItems {
		return nil, fiber.ErrRequestEntityTooLarge
	}
	return items, nil
}

func ok(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{"data": data, "error": nil})
}

func created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data, "error": nil})
}

func list(c *fiber.Ctx, data any, total int64) error {
	return c.JSON(fiber.Map{"data": data, "meta": fiber.Map{"total": total}, "error": nil})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func badRequest(c *fiber.Ctx, err error) error {
	if fiberError, ok := err.(*fiber.Error); ok {
		return fiberError
	}
	return writeError(c, fiber.ErrBadRequest)
}
