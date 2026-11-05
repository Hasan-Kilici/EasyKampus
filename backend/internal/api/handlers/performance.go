package handlers

import (
	"github.com/gofiber/fiber/v2"

	"lms/backend/pkg/domain/performance"
)

type PerformanceHandler struct{ service performance.Service }

func NewPerformanceHandler(service performance.Service) *PerformanceHandler {
	return &PerformanceHandler{service: service}
}

// @Summary		Get student performance analysis
// @Description	Sonucu Python analiz servisinden alır; servis erişilemezse 503 döner. Aynı girdi için sonuç önbellekten döner.
// @Tags			Performance
// @Produce		json
// @Success		200	{object}	performance.StudentResponse	"successful response"
// @Failure		401	{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Failure		503	{object}	handlers.ErrorResponse		"analiz servisi erişilemiyor"
// @Security		BearerAuth
// @Router			/api/students/me/performance [GET]
// @Router			/api/students/{studentID}/performance [GET]
func (handler *PerformanceHandler) Student(c *fiber.Ctx) error {
	studentID, err := studentIdentity(c)
	if err != nil {
		return err
	}
	response, err := handler.service.StudentPerformance(c.UserContext(), studentID)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, response)
}

// @Summary		Get teacher performance analysis
// @Description	Sonucu Python analiz servisinden alır; servis erişilemezse 503 döner.
// @Tags			Performance
// @Produce		json
// @Success		200	{object}	performance.TeacherResponse	"successful response"
// @Failure		401	{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Failure		503	{object}	handlers.ErrorResponse		"analiz servisi erişilemiyor"
// @Security		BearerAuth
// @Router			/api/teachers/me/performance [GET]
// @Router			/api/teachers/{teacherID}/performance [GET]
func (handler *PerformanceHandler) Teacher(c *fiber.Ctx) error {
	teacherID, err := teacherIdentity(c)
	if err != nil {
		return err
	}
	response, err := handler.service.TeacherPerformance(c.UserContext(), teacherID)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, response)
}
