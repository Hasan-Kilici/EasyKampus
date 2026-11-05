package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"lms/backend/internal/api/middleware"
	"lms/backend/internal/workflows"
	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/sessions"
	"lms/backend/pkg/domain/users"
)

type AuthHandler struct {
	users      users.Service
	sessions   sessions.Service
	registrar  workflows.StudentRegistrar
	jwtSecret  *string
	expiration time.Duration
	stateless  bool
}

func NewAuthHandler(
	userService users.Service,
	sessionService sessions.Service,
	registrar workflows.StudentRegistrar,
	jwtSecret *string,
	expiration time.Duration,
	stateless bool,
) *AuthHandler {
	return &AuthHandler{
		users:      userService,
		sessions:   sessionService,
		registrar:  registrar,
		jwtSecret:  jwtSecret,
		expiration: expiration,
		stateless:  stateless,
	}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerStudentRequest struct {
	credentialsRequest
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Age      int    `json:"age"`
	SchoolID string `json:"school_id"` // isteğe bağlı
	CampusID string `json:"campus_id"` // isteğe bağlı
}

// Register creates a student account and enrolls it into a school/campus when one is available.
//
//	@Summary	Register a student account (auto-enroll)
//	@Tags		Auth
//	@Produce	json
//	@Accept		json
//	@Param		request	body		registerStudentRequest	true	"İstek gövdesi"
//	@Success	201		{object}	map[string]interface{}	"successful response"
//	@Failure	400		{object}	handlers.ErrorResponse	"geçersiz istek"
//	@Failure	409		{object}	handlers.ErrorResponse	"e-posta zaten kayıtlı"
//	@Failure	422		{object}	handlers.ErrorResponse	"doğrulama hatası"
//	@Router		/api/auth/register [POST]
func (handler *AuthHandler) Register(c *fiber.Ctx) error {
	var request registerStudentRequest
	if err := c.BodyParser(&request); err != nil {
		return writeError(c, fiber.ErrBadRequest)
	}
	result, err := handler.registrar.Register(c.UserContext(), workflows.RegisterStudentInput{
		Name: request.Name, Surname: request.Surname, Email: request.Email, Password: request.Password,
		Age: request.Age, SchoolID: request.SchoolID, CampusID: request.CampusID,
	})
	if err != nil {
		return writeDomainError(c, err)
	}

	// Eski cevabın alanları (id, name, surname, role) korunur; üstüne kayıt bilgisi eklenir.
	data := fiber.Map{
		"id": result.User.ID, "name": result.User.Name, "surname": result.User.Surname, "role": result.User.Role,
		"enrolled":   result.Enrollment != nil,
		"enrollment": result.Enrollment,
		"placement":  result.Placement,
	}
	if result.SkippedReason != "" {
		data["enrollment_skipped_reason"] = result.SkippedReason
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data, "error": nil})
}

// Login creates a session and returns a JWT.
//
//	@Summary	Login and obtain a JWT token
//	@Tags		Auth
//	@Produce	json
//	@Accept		json
//	@Param		request	body		credentialsRequest		true	"İstek gövdesi"
//	@Success	200		{object}	map[string]interface{}	"successful response"
//	@Failure	400		{object}	handlers.ErrorResponse	"geçersiz istek"
//	@Failure	422		{object}	handlers.ErrorResponse	"doğrulama hatası"
//	@Router		/api/auth/login [POST]
func (handler *AuthHandler) Login(c *fiber.Ctx) error {
	var request credentialsRequest
	if err := c.BodyParser(&request); err != nil {
		return writeError(c, fiber.ErrBadRequest)
	}

	user, err := handler.users.Authenticate(
		c.UserContext(),
		request.Email,
		request.Password,
	)
	if err != nil {
		return writeDomainError(c, err)
	}

	var sessionID uuid.UUID
	if handler.stateless {
		sessionID = uuid.New()
	} else {
		session, err := handler.sessions.Create(
			c.UserContext(),
			user.ID,
			c.Get(fiber.HeaderUserAgent),
			handler.expiration,
		)
		if err != nil {
			return writeDomainError(c, err)
		}
		sessionID = session.ID
	}

	role := string(user.Role)

	token, err := core.CreateToken(
		*handler.jwtSecret,
		user.ID,
		sessionID,
		role,
		handler.expiration,
	)
	if err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"token": token,
			"user":  publicUser(user),
		},
		"error": nil,
	})
}

// Logout revokes the current session.
//
//	@Summary	Logout and clear the session token
//	@Tags		Auth
//	@Produce	json
//	@Success	204	"no content"
//	@Failure	401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
//	@Failure	403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
//	@Failure	422	{object}	handlers.ErrorResponse	"doğrulama hatası"
//	@Security	BearerAuth
//	@Router		/api/auth/logout [POST]
func (handler *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID, ok := c.Locals(middleware.SessionIDLocal).(interface{ String() string })
	if !ok {
		return writeError(c, fiber.ErrUnauthorized)
	}
	if err := handler.sessions.Revoke(c.UserContext(), sessionID.String()); err != nil && !errors.Is(err, core.ErrNotFound) {
		return writeDomainError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func publicUser(user *users.User) fiber.Map {
	return fiber.Map{"id": user.ID, "name": user.Name, "surname": user.Surname, "role": user.Role}
}

func writeDomainError(c *fiber.Ctx, err error) error {
	fiberError := fiber.ErrInternalServerError
	switch {
	case errors.Is(err, core.ErrValidation):
		fiberError = fiber.ErrUnprocessableEntity
	case errors.Is(err, core.ErrAlreadyExists):
		fiberError = fiber.ErrConflict
	case errors.Is(err, core.ErrUnauthorized):
		fiberError = fiber.ErrUnauthorized
	case errors.Is(err, core.ErrForbidden):
		fiberError = fiber.ErrForbidden
	case errors.Is(err, core.ErrNotFound):
		fiberError = fiber.ErrNotFound
	case errors.Is(err, core.ErrUnavailable):
		fiberError = fiber.ErrServiceUnavailable
	}
	return c.Status(fiberError.Code).JSON(fiber.Map{
		"data": nil,
		"error": fiber.Map{
			"code":    fiberError.Message,
			"message": domainMessage(err, fiberError),
		},
	})
}

func domainMessage(err error, fiberError *fiber.Error) string {
	switch {
	case errors.Is(err, core.ErrValidation):
		return "validation failed"
	case errors.Is(err, core.ErrAlreadyExists):
		return "already exists"
	case errors.Is(err, core.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, core.ErrForbidden):
		return "forbidden"
	case errors.Is(err, core.ErrNotFound):
		return "not found"
	case errors.Is(err, core.ErrUnavailable):
		return "dependency unavailable"
	}
	return fiberError.Message
}

func writeError(c *fiber.Ctx, fiberError *fiber.Error) error {
	return c.Status(fiberError.Code).JSON(fiber.Map{"data": nil, "error": fiber.Map{"code": fiberError.Message, "message": fiberError.Message}})
}
