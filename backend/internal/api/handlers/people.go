package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/users"
)

type EnrollmentHandler struct{ service enrollments.Service }

func NewEnrollmentHandler(service enrollments.Service) *EnrollmentHandler {
	return &EnrollmentHandler{service: service}
}

// @Summary		Enroll a student into a class
// @Description	Öğrenciyi sınıfa kaydeder.
// @Tags			Enrollment
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		enrollments.EnrollInput	true	"İstek gövdesi"
// @Success		201		{object}	enrollments.Enrollment	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/enrollments [POST]
func (handler *EnrollmentHandler) Enroll(c *fiber.Ctx) error {
	input, err := bodyParser[enrollments.EnrollInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Enroll(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary	Enroll students in bulk
// @Tags		Enrollment
// @Accept		json
// @Produce	json
// @Param		classID	path	string						true	"Sınıf kimliği (UUID)"
// @Param		request	body	[]enrollments.EnrollInput	true	"Enrollment records"
// @Success	201		{array}	enrollments.Enrollment
// @Security	BearerAuth
// @Router		/api/classes/{classID}/enrollments/bulk [POST]
func (handler *EnrollmentHandler) EnrollBulk(c *fiber.Ctx) error {
	inputs, err := batchParser[enrollments.EnrollInput](c, "enrollments")
	if err != nil {
		return badRequest(c, err)
	}
	result, err := handler.service.EnrollBulk(c.UserContext(), c.Params("classID"), inputs)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, result)
}

// @Summary	Transfer a student between classes
// @Tags		Enrollment
// @Accept		json
// @Produce	json
// @Param		classID	path		string						true	"Source class UUID"
// @Param		request	body		enrollments.TransferInput	true	"Transfer payload"
// @Success	201		{object}	enrollments.Enrollment
// @Security	BearerAuth
// @Router		/api/classes/{classID}/transfers [POST]
func (handler *EnrollmentHandler) Transfer(c *fiber.Ctx) error {
	input, err := bodyParser[enrollments.TransferInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Transfer(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary	List class enrollments
// @Tags		Enrollment
// @Produce	json
// @Param		classID	path		string					true	"Sınıf kimliği (UUID)"
// @Success	200		{array}		enrollments.Enrollment	"successful response"
// @Failure	401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure	422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/classes/{classID}/enrollments [GET]
func (handler *EnrollmentHandler) ListByClass(c *fiber.Ctx) error {
	result, err := handler.service.ListByClass(c.UserContext(), c.Params("classID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		List student enrollments
// @Description	/me/ öneki ile çağrıldığında token içindeki öğrenci kullanılır.
// @Tags			Enrollment
// @Produce		json
// @Success		200	{array}		enrollments.Enrollment	"successful response"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/students/me/enrollments [GET]
// @Router			/api/students/{studentID}/enrollments [GET]
func (handler *EnrollmentHandler) ListByStudent(c *fiber.Ctx) error {
	studentID := c.Params("studentID")
	if studentID == "" {
		userID, _, err := identity(c)
		if err != nil {
			return err
		}
		studentID = userID.String()
	}
	result, err := handler.service.ListByStudent(c.UserContext(), studentID)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Withdraw a student from a class
// @Description	Öğrencinin kaydını siler.
// @Tags			Enrollment
// @Produce		json
// @Param			classID		path	string	true	"Sınıf kimliği (UUID)"
// @Param			studentID	path	string	true	"Öğrenci kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/enrollments/{studentID} [DELETE]
func (handler *EnrollmentHandler) Withdraw(c *fiber.Ctx) error {
	if err := handler.service.Withdraw(c.UserContext(), c.Params("classID"), c.Params("studentID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type StudentHandler struct {
	completions lessons.Service
	accounts    users.Service
}

func NewStudentHandler(completionService lessons.Service, accountService users.Service) *StudentHandler {
	return &StudentHandler{completions: completionService, accounts: accountService}
}

// @Summary		List lesson completions
// @Description	Ders tamamlama kayıtlarını listeler.
// @Tags			Student
// @Produce		json
// @Param			course_id	query		string						false	"Sonuçları derse göre filtrele"
// @Success		200			{array}		lessons.LessonCompletion	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		422			{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/students/me/lessons [GET]
// @Router			/api/students/{studentID}/lessons [GET]
func (handler *StudentHandler) Lessons(c *fiber.Ctx) error {
	studentID, err := studentIdentity(c)
	if err != nil {
		return err
	}
	result, err := handler.completions.ListCompletions(c.UserContext(), studentID.String(), c.Query("course_id"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary	Get a student profile
// @Tags		Student
// @Produce	json
// @Param		studentID	path		string					true	"Öğrenci kimliği (UUID)"
// @Success	200			{object}	map[string]interface{}	"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure	422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/students/{studentID}/profile [GET]
func (handler *StudentHandler) Profile(c *fiber.Ctx) error {
	studentID, err := studentIdentity(c)
	if err != nil {
		return err
	}
	user, err := handler.accounts.GetByID(c.UserContext(), studentID.String())
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, publicUser(user))
}

type UserHandler struct{ service users.Service }

func NewUserHandler(service users.Service) *UserHandler { return &UserHandler{service: service} }

// @Summary		Create User
// @Description	Kaydı oluşturur.
// @Tags			User
// @Produce		json
// @Accept			json
// @Param			request	body		users.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	map[string]interface{}	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users [POST]
func (handler *UserHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[users.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	user, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, publicUser(user))
}

// @Summary		Get User
// @Description	Tek kaydı döner.
// @Tags			User
// @Produce		json
// @Param			userID	path		string					true	"Kullanıcı kimliği (UUID)"
// @Success		200		{object}	map[string]interface{}	"successful response"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users/{userID} [GET]
func (handler *UserHandler) Get(c *fiber.Ctx) error {
	user, err := handler.service.GetByID(c.UserContext(), c.Params("userID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, publicUser(user))
}

// @Summary		List User
// @Description	Kayıtları listeler.
// @Tags			User
// @Produce		json
// @Success		200	{array}		users.User				"successful response"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users [GET]
func (handler *UserHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c)
	result, err := handler.service.List(c.UserContext(), users.ListParams{Page: page, Limit: limit})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Update User
// @Description	Kaydı günceller.
// @Tags			User
// @Produce		json
// @Accept			json
// @Param			userID	path		string					true	"Kullanıcı kimliği (UUID)"
// @Param			request	body		users.UpdateInput		true	"İstek gövdesi"
// @Success		200		{object}	map[string]interface{}	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users/{userID} [PUT]
func (handler *UserHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[users.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	user, err := handler.service.Update(c.UserContext(), c.Params("userID"), input)
	if err != nil {
		fmt.Println("Error updating user:", err) // Debugging line
		return writeDomainError(c, err)
	}
	return ok(c, publicUser(user))
}

// @Summary		Delete User
// @Description	Kaydı siler.
// @Tags			User
// @Produce		json
// @Param			userID	path	string	true	"Kullanıcı kimliği (UUID)"
// @Success		204		"no content"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users/{userID} [DELETE]
func (handler *UserHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("userID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		Get the authenticated user
// @Description	Token içindeki kullanıcı bilgisini döner.
// @Tags			User
// @Produce		json
// @Success		200	{object}	map[string]interface{}	"successful response"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/users/me [GET]
func (handler *UserHandler) Me(c *fiber.Ctx) error {
	userID, _, err := identity(c)
	if err != nil {
		return err
	}
	user, err := handler.service.GetByID(c.UserContext(), userID.String())
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, publicUser(user))
}
