package handlers

import (
	"github.com/gofiber/fiber/v2"

	"lms/backend/pkg/domain/terms"
)

type TermHandler struct{ service terms.Service }

func NewTermHandler(service terms.Service) *TermHandler { return &TermHandler{service: service} }

func (handler *TermHandler) Start(c *fiber.Ctx) error {
	input, err := bodyParser[terms.StartInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.SchoolID == "" {
		input.SchoolID = c.Params("schoolID")
	}
	term, err := handler.service.Start(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, term)
}

func (handler *TermHandler) Close(c *fiber.Ctx) error {
	term, err := handler.service.Close(c.UserContext(), c.Params("termID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, term)
}

func (handler *TermHandler) List(c *fiber.Ctx) error {
	result, err := handler.service.List(c.UserContext(), c.Params("schoolID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}
