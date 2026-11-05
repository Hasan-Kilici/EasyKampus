package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apiMiddleware "lms/backend/internal/api/middleware"
	"lms/backend/internal/workflows"
	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/relevance"
	"lms/backend/pkg/domain/schedules"
	"lms/backend/pkg/domain/schools"
	"lms/backend/pkg/domain/topics"
)

type SchoolHandler struct {
	service schools.Service
	starter workflows.SchoolStarter
}

func NewSchoolHandler(service schools.Service, starters ...workflows.SchoolStarter) *SchoolHandler {
	handler := &SchoolHandler{service: service}
	if len(starters) > 0 {
		handler.starter = starters[0]
	}
	return handler
}

// @Summary		Start a school
// @Description	Creates a school and all submitted campuses in one transaction.
// @Tags			School
// @Accept			json
// @Produce		json
// @Param			request	body		workflows.StartSchoolInput	true	"School and campuses"
// @Success		201		{object}	workflows.StartSchoolResult
// @Failure		400		{object}	handlers.ErrorResponse
// @Failure		409		{object}	handlers.ErrorResponse
// @Failure		422		{object}	handlers.ErrorResponse
// @Security		BearerAuth
// @Router			/api/schools/start [POST]
func (handler *SchoolHandler) Start(c *fiber.Ctx) error {
	if handler.starter == nil {
		return writeError(c, fiber.ErrNotImplemented)
	}
	input, err := bodyParser[workflows.StartSchoolInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	result, err := handler.starter.Start(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, result)
}

// @Summary		Create School
// @Description	Kaydı oluşturur.
// @Tags			School
// @Produce		json
// @Accept			json
// @Param			request	body		schools.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	schools.School			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools [POST]
func (handler *SchoolHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[schools.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	school, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, school)
}

// @Summary		List School
// @Description	Kayıtları listeler.
// @Tags			School
// @Produce		json
// @Param			q	query		string					false	"q"
// @Success		200	{array}		schools.School			"successful response"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools [GET]
func (handler *SchoolHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c)
	viewer, err := listViewer(c)
	if err != nil {
		return err
	}
	result, err := handler.service.List(c.UserContext(), schools.ListParams{Page: page, Limit: limit, Query: c.Query("q"), Viewer: viewer})
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Get School
// @Description	Tek kaydı döner.
// @Tags			School
// @Produce		json
// @Param			schoolID	path		string					true	"Okul kimliği (UUID)"
// @Success		200			{object}	schools.School			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools/{schoolID} [GET]
func (handler *SchoolHandler) Get(c *fiber.Ctx) error {
	school, err := handler.service.GetByID(c.UserContext(), c.Params("schoolID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, school)
}

// @Summary		Update School
// @Description	Kaydı günceller.
// @Tags			School
// @Produce		json
// @Accept			json
// @Param			schoolID	path		string					true	"Okul kimliği (UUID)"
// @Param			request		body		schools.UpdateInput		true	"İstek gövdesi"
// @Success		200			{object}	schools.School			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools/{schoolID} [PUT]
func (handler *SchoolHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[schools.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	school, err := handler.service.Update(c.UserContext(), c.Params("schoolID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, school)
}

// @Summary		Delete School
// @Description	Kaydı siler.
// @Tags			School
// @Produce		json
// @Param			schoolID	path	string	true	"Okul kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools/{schoolID} [DELETE]
func (handler *SchoolHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("schoolID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary	List school staff
// @Tags		School
// @Produce	json
// @Param		schoolID	path		string					true	"Okul kimliği (UUID)"
// @Success	200			{array}		schools.Staff			"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure	422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/schools/{schoolID}/staff [GET]
func (handler *SchoolHandler) ListStaff(c *fiber.Ctx) error {
	staff, err := handler.service.ListStaff(c.UserContext(), c.Params("schoolID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, staff)
}

// @Summary		Assign a staff member to a school
// @Description	Personeli okula atar.
// @Tags			School
// @Produce		json
// @Accept			json
// @Param			schoolID	path		string					true	"Okul kimliği (UUID)"
// @Param			request		body		schools.StaffInput		true	"İstek gövdesi"
// @Success		201			{object}	map[string]interface{}	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools/{schoolID}/staff [POST]
func (handler *SchoolHandler) AssignStaff(c *fiber.Ctx) error {
	input, err := bodyParser[schools.StaffInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if err := handler.service.AssignStaff(c.UserContext(), c.Params("schoolID"), input); err != nil {
		return writeDomainError(c, err)
	}
	return created(c, fiber.Map{"school_id": c.Params("schoolID"), "user_id": input.UserID, "role": input.Role})
}

// @Summary		Remove a staff member from a school
// @Description	Personelin okul atamasını kaldırır.
// @Tags			School
// @Produce		json
// @Param			schoolID	path	string	true	"Okul kimliği (UUID)"
// @Param			userID		path	string	true	"Kullanıcı kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schools/{schoolID}/staff/{userID} [DELETE]
func (handler *SchoolHandler) RemoveStaff(c *fiber.Ctx) error {
	if err := handler.service.RemoveStaff(c.UserContext(), c.Params("schoolID"), c.Params("userID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type CampusHandler struct{ service campuses.Service }

func NewCampusHandler(service campuses.Service) *CampusHandler {
	return &CampusHandler{service: service}
}

// @Summary		Create Campus
// @Description	Kaydı oluşturur.
// @Tags			Campus
// @Produce		json
// @Accept			json
// @Param			request	body		campuses.CreateInput	true	"İstek gövdesi"
// @Success		201		{object}	campuses.Campus			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/campuses [POST]
func (handler *CampusHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[campuses.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	campus, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, campus)
}

// @Summary		List Campus
// @Description	Kayıtları listeler.
// @Tags			Campus
// @Produce		json
// @Param			q	query		string					false	"q"
// @Success		200	{array}		campuses.Campus			"successful response"
// @Failure		400	{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/campuses [GET]
func (handler *CampusHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c)
	schoolID, err := queryUUID(c, "school_id")
	if err != nil {
		return badRequest(c, err)
	}
	viewer, err := listViewer(c)
	if err != nil {
		return err
	}
	result, err := handler.service.List(c.UserContext(), campuses.ListParams{Page: page, Limit: limit, SchoolID: schoolID, Query: c.Query("q"), Viewer: viewer})
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Get Campus
// @Description	Tek kaydı döner.
// @Tags			Campus
// @Produce		json
// @Param			campusID	path		string					true	"Kampüs kimliği (UUID)"
// @Success		200			{object}	campuses.Campus			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/campuses/{campusID} [GET]
func (handler *CampusHandler) Get(c *fiber.Ctx) error {
	campus, err := handler.service.GetByID(c.UserContext(), c.Params("campusID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, campus)
}

// @Summary		Update Campus
// @Description	Kaydı günceller.
// @Tags			Campus
// @Produce		json
// @Accept			json
// @Param			campusID	path		string					true	"Kampüs kimliği (UUID)"
// @Param			request		body		campuses.UpdateInput	true	"İstek gövdesi"
// @Success		200			{object}	campuses.Campus			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/campuses/{campusID} [PUT]
func (handler *CampusHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[campuses.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	campus, err := handler.service.Update(c.UserContext(), c.Params("campusID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, campus)
}

// @Summary		Delete Campus
// @Description	Kaydı siler.
// @Tags			Campus
// @Produce		json
// @Param			campusID	path	string	true	"Kampüs kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/campuses/{campusID} [DELETE]
func (handler *CampusHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("campusID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type CourseHandler struct{ service courses.Service }

func NewCourseHandler(service courses.Service) *CourseHandler {
	return &CourseHandler{service: service}
}

// @Summary		Create Course
// @Description	Kaydı oluşturur.
// @Tags			Course
// @Produce		json
// @Accept			json
// @Param			request	body		courses.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	courses.Course			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses [POST]
func (handler *CourseHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[courses.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	course, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, course)
}

// @Summary		List Course
// @Description	Kayıtları listeler.
// @Tags			Course
// @Produce		json
// @Param			subject_area	query		string					false	"subject_area"
// @Param			q				query		string					false	"q"
// @Success		200				{array}		courses.Course			"successful response"
// @Failure		400				{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401				{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403				{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422				{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses [GET]
func (handler *CourseHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c)
	schoolID, err := queryUUID(c, "school_id")
	if err != nil {
		return badRequest(c, err)
	}
	viewer, err := listViewer(c)
	if err != nil {
		return err
	}
	result, err := handler.service.List(c.UserContext(), courses.ListParams{
		Page:        page,
		Limit:       limit,
		SchoolID:    schoolID,
		SubjectArea: c.Query("subject_area"),
		ActiveOnly:  queryBool(c, "active"),
		Query:       c.Query("q"),
		Viewer:      viewer,
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Get Course
// @Description	Tek kaydı döner.
// @Tags			Course
// @Produce		json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Success		200			{object}	courses.Course			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID} [GET]
func (handler *CourseHandler) Get(c *fiber.Ctx) error {
	course, err := handler.service.GetByID(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, course)
}

// @Summary		Update Course
// @Description	Kaydı günceller.
// @Tags			Course
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			request		body		courses.UpdateInput		true	"İstek gövdesi"
// @Success		200			{object}	courses.Course			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID} [PUT]
func (handler *CourseHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[courses.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	course, err := handler.service.Update(c.UserContext(), c.Params("courseID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, course)
}

// @Summary		Delete Course
// @Description	Kaydı siler.
// @Tags			Course
// @Produce		json
// @Param			courseID	path	string	true	"Ders kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID} [DELETE]
func (handler *CourseHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("courseID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type TopicHandler struct {
	service topics.Service
	courses courses.Service
}

func NewTopicHandler(service topics.Service, courseService courses.Service) *TopicHandler {
	return &TopicHandler{service: service, courses: courseService}
}

// @Summary		Course workload (EFOR / AKTS)
// @Description	Dersin iş yükünü döner: aktif konu saatleri + ödev + proje saatleri ve buna göre AKTS. hours_per_ects 25–30 arası verilebilir (varsayılan 30).
// @Tags			Topic
// @Produce		json
// @Param			courseID		path		string		true	"Ders kimliği (UUID)"
// @Param			hours_per_ects	query		number		false	"Bir AKTS'nin saat karşılığı (25–30)"
// @Success		200				{object}	courses.Workload	"successful response"
// @Failure		401				{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403				{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404				{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422				{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/workload [GET]
func (handler *TopicHandler) Workload(c *fiber.Ctx) error {
	hoursPerECTS := 0.0
	if raw := strings.TrimSpace(c.Query("hours_per_ects")); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return badRequest(c, err)
		}
		hoursPerECTS = parsed
	}
	course, err := handler.courses.GetByID(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	items, err := handler.service.ListByCourse(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	workload, err := courses.ComputeWorkload(*course, items, hoursPerECTS)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, workload)
}

// @Summary		Course content analysis
// @Description	Dersin aktif konularını, dersin bölümüne göre TF-IDF + kosinüs benzerliğiyle puanlar (0–100). Bölüm veya anahtar kelime yoksa ready=false döner.
// @Tags			Topic
// @Produce		json
// @Param			courseID	path		string	true	"Ders kimliği (UUID)"
// @Success		200			{object}	relevance.Result	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/content-analysis [GET]
func (handler *TopicHandler) ContentAnalysis(c *fiber.Ctx) error {
	course, err := handler.courses.GetByID(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	items, err := handler.service.ListByCourse(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	scored := make([]relevance.Topic, 0, len(items))
	for _, topic := range items {
		if !topic.Active {
			continue
		}
		scored = append(scored, relevance.Topic{ID: topic.ID.String(), Name: topic.Name, Description: topic.Description, Weight: topic.Weight})
	}
	return ok(c, relevance.Analyze(course.Department, course.DepartmentKeywords, scored))
}

// @Summary		Create Topic
// @Description	Kaydı oluşturur.
// @Tags			Topic
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			request		body		topics.CreateInput		true	"İstek gövdesi"
// @Success		201			{object}	topics.Topic			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/topics [POST]
func (handler *TopicHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[topics.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.CourseID == "" {
		input.CourseID = c.Params("courseID")
	}
	topic, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, topic)
}

// @Summary		List Topic
// @Description	Kayıtları listeler.
// @Tags			Topic
// @Produce		json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Success		200			{object}	map[string]interface{}	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/topics [GET]
func (handler *TopicHandler) List(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("courseID"))
	if err != nil {
		return fiber.ErrNotFound
	}
	result, err := handler.service.ListByCourse(c.UserContext(), c.Params("courseID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, filterActiveTopics(result, queryBool(c, "active"), courseID))
}

// @Summary		Get Topic
// @Description	Tek kaydı döner.
// @Tags			Topic
// @Produce		json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			topicID		path		string					true	"Konu kimliği (UUID)"
// @Success		200			{object}	topics.Topic			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/topics/{topicID} [GET]
func (handler *TopicHandler) Get(c *fiber.Ctx) error {
	topic, err := handler.service.GetByID(c.UserContext(), c.Params("topicID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, topic)
}

// @Summary		Update Topic
// @Description	Kaydı günceller.
// @Tags			Topic
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			topicID		path		string					true	"Konu kimliği (UUID)"
// @Param			request		body		topics.UpdateInput		true	"İstek gövdesi"
// @Success		200			{object}	topics.Topic			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/topics/{topicID} [PUT]
func (handler *TopicHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[topics.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	topic, err := handler.service.Update(c.UserContext(), c.Params("topicID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, topic)
}

// @Summary		Delete Topic
// @Description	Kaydı siler.
// @Tags			Topic
// @Produce		json
// @Param			courseID	path	string	true	"Ders kimliği (UUID)"
// @Param			topicID		path	string	true	"Konu kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/topics/{topicID} [DELETE]
func (handler *TopicHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("topicID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type ClassHandler struct{ service classes.Service }

func NewClassHandler(service classes.Service) *ClassHandler { return &ClassHandler{service: service} }

// @Summary		Create Class
// @Description	Kaydı oluşturur.
// @Tags			Class
// @Produce		json
// @Accept			json
// @Param			request	body		classes.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	classes.Class			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes [POST]
func (handler *ClassHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[classes.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary		List Class
// @Description	Kayıtları listeler.
// @Tags			Class
// @Produce		json
// @Param			q	query		string					false	"q"
// @Success		200	{array}		classes.Class			"successful response"
// @Failure		400	{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes [GET]
func (handler *ClassHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c)
	schoolID, err := queryUUID(c, "school_id")
	if err != nil {
		return badRequest(c, err)
	}
	campusID, err := queryUUID(c, "campus_id")
	if err != nil {
		return badRequest(c, err)
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	userID, role, err := identity(c)
	if err != nil {
		return err
	}
	params := classes.ListParams{Page: page, Limit: limit, SchoolID: schoolID, CampusID: campusID, CourseID: courseID, Query: c.Query("q")}
	result, err := handler.service.List(c.UserContext(), classes.Scope{UserID: userID, Role: role}, params)
	if err != nil {
		return writeDomainError(c, err)
	}
	total, err := handler.service.Count(c.UserContext(), classes.Scope{UserID: userID, Role: role}, params)
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, total)
}

// @Summary		Get Class
// @Description	Tek kaydı döner.
// @Tags			Class
// @Produce		json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Success		200		{object}	classes.Class			"successful response"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID} [GET]
func (handler *ClassHandler) Get(c *fiber.Ctx) error {
	entity, err := handler.service.GetByID(c.UserContext(), c.Params("classID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Update Class
// @Description	Kaydı günceller.
// @Tags			Class
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		classes.UpdateInput		true	"İstek gövdesi"
// @Success		200		{object}	classes.Class			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID} [PUT]
func (handler *ClassHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[classes.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Update(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Delete Class
// @Description	Kaydı siler.
// @Tags			Class
// @Produce		json
// @Param			classID	path	string	true	"Sınıf kimliği (UUID)"
// @Success		204		"no content"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID} [DELETE]
func (handler *ClassHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("classID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type AssignmentHandler struct{ service assignments.Service }

func NewAssignmentHandler(service assignments.Service) *AssignmentHandler {
	return &AssignmentHandler{service: service}
}

// @Summary		AssignCourse Assignment
// @Description	Dersi programa atar.
// @Tags			Assignment
// @Produce		json
// @Accept			json
// @Param			classID	path		string							true	"Sınıf kimliği (UUID)"
// @Param			request	body		assignments.AssignCourseInput	true	"İstek gövdesi"
// @Success		201		{object}	assignments.ClassCourse			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse			"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse			"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse			"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse			"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse			"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse			"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/courses [POST]
func (handler *AssignmentHandler) AssignCourse(c *fiber.Ctx) error {
	input, err := bodyParser[assignments.AssignCourseInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.AssignCourse(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary	ListCourses Assignment
// @Tags		Assignment
// @Produce	json
// @Param		classID	path		string					true	"Sınıf kimliği (UUID)"
// @Success	200		{array}		assignments.ClassCourse	"successful response"
// @Failure	401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure	422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/classes/{classID}/courses [GET]
func (handler *AssignmentHandler) ListCourses(c *fiber.Ctx) error {
	result, err := handler.service.ListCourses(c.UserContext(), c.Params("classID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		RemoveCourse Assignment
// @Description	Dersin program atamasını kaldırır.
// @Tags			Assignment
// @Produce		json
// @Param			classID		path	string	true	"Sınıf kimliği (UUID)"
// @Param			courseID	path	string	true	"Ders kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/courses/{courseID} [DELETE]
func (handler *AssignmentHandler) RemoveCourse(c *fiber.Ctx) error {
	if err := handler.service.RemoveCourse(c.UserContext(), c.Params("classID"), c.Params("courseID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		AssignTeacher Assignment
// @Description	Öğretmeni derse atar.
// @Tags			Assignment
// @Produce		json
// @Accept			json
// @Param			classID	path		string							true	"Sınıf kimliği (UUID)"
// @Param			request	body		assignments.AssignTeacherInput	true	"İstek gövdesi"
// @Success		201		{object}	assignments.TeacherAssignment	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse			"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse			"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse			"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse			"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse			"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse			"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/teachers [POST]
func (handler *AssignmentHandler) AssignTeacher(c *fiber.Ctx) error {
	input, err := bodyParser[assignments.AssignTeacherInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.AssignTeacher(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary	ListTeachers Assignment
// @Tags		Assignment
// @Produce	json
// @Param		classID	path		string							true	"Sınıf kimliği (UUID)"
// @Success	200		{array}		assignments.TeacherAssignment	"successful response"
// @Failure	401		{object}	handlers.ErrorResponse			"kimlik doğrulanmadı"
// @Failure	403		{object}	handlers.ErrorResponse			"yetkisiz işlem"
// @Failure	404		{object}	handlers.ErrorResponse			"kayıt bulunamadı"
// @Failure	422		{object}	handlers.ErrorResponse			"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/classes/{classID}/teachers [GET]
func (handler *AssignmentHandler) ListTeachers(c *fiber.Ctx) error {
	result, err := handler.service.ListTeachers(c.UserContext(), c.Params("classID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		RemoveTeacher Assignment
// @Description	Öğretmenin ders atamasını kaldırır.
// @Tags			Assignment
// @Produce		json
// @Param			classID		path	string	true	"Sınıf kimliği (UUID)"
// @Param			teacherID	path	string	true	"Öğretmen kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/teachers/{teacherID} [DELETE]
func (handler *AssignmentHandler) RemoveTeacher(c *fiber.Ctx) error {
	if err := handler.service.RemoveTeacher(c.UserContext(), c.Params("classID"), c.Params("teacherID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type ScheduleHandler struct{ service schedules.Service }

func NewScheduleHandler(service schedules.Service) *ScheduleHandler {
	return &ScheduleHandler{service: service}
}

// @Summary		Create Schedule
// @Description	Kaydı oluşturur.
// @Tags			Schedule
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		schedules.CreateInput	true	"İstek gövdesi"
// @Success		201		{object}	schedules.ScheduleEntry	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/schedules [POST]
func (handler *ScheduleHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[schedules.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Create(c.UserContext(), c.Params("classID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary		List Schedule
// @Description	Kayıtları listeler.
// @Tags			Schedule
// @Produce		json
// @Param			term	query		string					false	"term"
// @Success		200		{array}		schedules.ScheduleEntry	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/schedules [GET]
// @Router			/api/schedules [GET]
func (handler *ScheduleHandler) List(c *fiber.Ctx) error {
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return badRequest(c, err)
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	teacherID, err := queryUUID(c, "teacher_id")
	if err != nil {
		return badRequest(c, err)
	}
	dayOfWeek, err := queryDay(c, "day_of_week")
	if err != nil {
		return badRequest(c, err)
	}
	classIDs, applied := apiMiddleware.ScopedClassIDs(c)
	result, err := handler.service.List(c.UserContext(), schedules.ListParams{
		ClassID:      classID,
		ClassIDs:     classIDs,
		ScopeApplied: applied,
		CourseID:     courseID,
		TeacherID:    teacherID,
		Term:         c.Query("term"),
		DayOfWeek:    dayOfWeek,
		ActiveOnly:   !queryBool(c, "all"),
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Get Schedule
// @Description	Tek kaydı döner.
// @Tags			Schedule
// @Produce		json
// @Param			scheduleID	path		string					true	"Ders programı kimliği (UUID)"
// @Success		200			{object}	schedules.ScheduleEntry	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schedules/{scheduleID} [GET]
func (handler *ScheduleHandler) Get(c *fiber.Ctx) error {
	entity, err := handler.service.GetByID(c.UserContext(), c.Params("scheduleID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Update Schedule
// @Description	Kaydı günceller.
// @Tags			Schedule
// @Produce		json
// @Accept			json
// @Param			scheduleID	path		string					true	"Ders programı kimliği (UUID)"
// @Param			request		body		schedules.UpdateInput	true	"İstek gövdesi"
// @Success		200			{object}	schedules.ScheduleEntry	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schedules/{scheduleID} [PUT]
func (handler *ScheduleHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[schedules.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	entity, err := handler.service.Update(c.UserContext(), c.Params("scheduleID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Delete Schedule
// @Description	Kaydı siler.
// @Tags			Schedule
// @Produce		json
// @Param			scheduleID	path	string	true	"Ders programı kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/schedules/{scheduleID} [DELETE]
func (handler *ScheduleHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("scheduleID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

func filterActiveTopics(result []topics.Topic, activeOnly bool, courseID uuid.UUID) []topics.Topic {
	if !activeOnly {
		return result
	}
	filtered := make([]topics.Topic, 0, len(result))
	for _, topic := range result {
		if topic.Active && topic.CourseID == courseID {
			filtered = append(filtered, topic)
		}
	}
	return filtered
}
