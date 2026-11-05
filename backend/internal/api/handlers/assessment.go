package handlers

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apiMiddleware "lms/backend/internal/api/middleware"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/grades"
	"lms/backend/pkg/domain/notes"
	"lms/backend/pkg/domain/scopeguard"
	"lms/backend/pkg/domain/users"
)

type AttendanceHandler struct{ service attendance.Service }

func NewAttendanceHandler(service attendance.Service) *AttendanceHandler {
	return &AttendanceHandler{service: service}
}

// @Summary		Record attendance for students
// @Description	Tek kayıt, dizi veya {"records": [...]} sarmalayıcısı kabul eder.
// @Tags			Attendance
// @Produce		json
// @Accept			json
// @Param			classID	path		string						true	"Sınıf kimliği (UUID)"
// @Param			request	body		[]attendance.RecordInput	true	"Gönderilecek kayıtlar"
// @Success		201		{array}		attendance.Attendance		"successful response"
// @Failure		400		{object}	handlers.ErrorResponse		"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse		"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse		"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/attendance [POST]
func (handler *AttendanceHandler) Record(c *fiber.Ctx) error {
	input, err := batchParser[attendance.RecordInput](c, "records")
	if err != nil {
		return badRequest(c, err)
	}
	if len(input) == 0 {
		return badRequest(c, core.ErrValidation)
	}
	classID := c.Params("classID")
	for index := range input {
		if input[index].ClassID == "" {
			input[index].ClassID = classID
		}
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	records, err := handler.service.Record(c.UserContext(), actorID, role, input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, records)
}

// @Summary		Update an attendance record
// @Description	Kaydı günceller.
// @Tags			Attendance
// @Produce		json
// @Accept			json
// @Param			attendanceID	path		string					true	"Yoklama kaydı kimliği (UUID)"
// @Param			request			body		attendance.RecordInput	true	"İstek gövdesi"
// @Success		200				{object}	attendance.Attendance	"successful response"
// @Failure		400				{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401				{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403				{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404				{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409				{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422				{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attendance/{attendanceID} [PUT]
func (handler *AttendanceHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[attendance.RecordInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	entity, err := handler.service.Update(c.UserContext(), actorID, role, c.Params("attendanceID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Delete an attendance record
// @Description	Kaydı siler.
// @Tags			Attendance
// @Produce		json
// @Param			attendanceID	path	string	true	"Yoklama kaydı kimliği (UUID)"
// @Success		204				"no content"
// @Failure		401				{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403				{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404				{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422				{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attendance/{attendanceID} [DELETE]
func (handler *AttendanceHandler) Delete(c *fiber.Ctx) error {
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	if err := handler.service.Delete(c.UserContext(), actorID, role, c.Params("attendanceID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		List attendance records
// @Description	Kayıtları listeler.
// @Tags			Attendance
// @Produce		json
// @Param			class_id	query		string					false	"Sınıf kimliği (UUID)"
// @Param			course_id	query		string					false	"Ders kimliği (UUID)"
// @Param			student_id	query		string					false	"Öğrenci kimliği (UUID)"
// @Param			from		query		string					false	"Başlangıç tarihi (YYYY-MM-DD)"
// @Param			to			query		string					false	"Bitiş tarihi (YYYY-MM-DD)"
// @Param			status		query		string					false	"Durum filtresi"
// @Param			limit		query		int						false	"Sayfa başına kayıt (en fazla 200)"
// @Success		200			{array}		attendance.Attendance	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attendance [GET]
// @Router			/api/classes/{classID}/attendance [GET]
// @Router			/api/students/{studentID}/attendance [GET]
func (handler *AttendanceHandler) List(c *fiber.Ctx) error {
	params, err := handler.listParams(c)
	if err != nil {
		return err
	}
	_, params.Limit = pageParams(c)
	result, err := handler.service.List(c.UserContext(), params)
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Attendance summary for a date range
// @Description	Tarih aralığındaki yoklamaların durum toplamlarını ve günlük kırılımını döner.
// @Tags			Attendance
// @Produce		json
// @Param			class_id	query	string	false	"Sınıf kimliği (UUID)"
// @Param			course_id	query	string	false	"Ders kimliği (UUID)"
// @Param			student_id	query	string	false	"Öğrenci kimliği (UUID)"
// @Param			from		query	string	false	"Başlangıç tarihi (YYYY-MM-DD)"
// @Param			to			query	string	false	"Bitiş tarihi (YYYY-MM-DD)"
// @Param			status		query	string	false	"Durum filtresi"
// @Success		200			{object}	attendance.PeriodSummary	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse		"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		422			{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attendance/summary [GET]
func (handler *AttendanceHandler) Summary(c *fiber.Ctx) error {
	params, err := handler.listParams(c)
	if err != nil {
		return err
	}
	result, err := handler.service.Summarize(c.UserContext(), params)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

func (handler *AttendanceHandler) listParams(c *fiber.Ctx) (attendance.ListParams, error) {
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	studentID, err := queryUUID(c, "student_id")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	teacherID, err := queryUUID(c, "teacher_id")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	if classID == nil {
		if classID, err = paramUUID(c, "classID"); err != nil {
			return attendance.ListParams{}, badRequest(c, err)
		}
	}
	if studentID == nil {
		if studentID, err = paramUUID(c, "studentID"); err != nil {
			return attendance.ListParams{}, badRequest(c, err)
		}
	}
	if studentID, err = studentSubject(c, studentID); err != nil {
		return attendance.ListParams{}, err
	}
	from, err := queryDate(c, "from")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	to, err := queryDate(c, "to")
	if err != nil {
		return attendance.ListParams{}, badRequest(c, err)
	}
	classIDs, applied := apiMiddleware.ScopedClassIDs(c)
	return attendance.ListParams{
		ClassID:      classID,
		ClassIDs:     classIDs,
		ScopeApplied: applied,
		CourseID:     courseID,
		StudentID:    studentID,
		TeacherID:    teacherID,
		From:         derefString(from),
		To:           derefString(to),
		Status:       attendance.Status(c.Query("status")),
	}, nil
}

type GradeHandler struct{ service grades.Service }

func NewGradeHandler(service grades.Service) *GradeHandler { return &GradeHandler{service: service} }

// @Summary		Record Grade
// @Description	Tek kayıt, dizi veya {"grades": [...]} sarmalayıcısı kabul eder.
// @Tags			Grade
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		grades.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	grades.Grade			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/grades [POST]
func (handler *GradeHandler) Record(c *fiber.Ctx) error {
	input, err := bodyParser[grades.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.ClassID == "" {
		input.ClassID = c.Params("classID")
	}
	if input.CourseID == "" {
		input.CourseID = c.Params("courseID")
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	entity, err := handler.service.Record(c.UserContext(), actorID, role, input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary		Record grades in bulk
// @Description	Records multiple grades atomically after validating teacher assignment and enrollment.
// @Tags			Grade
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		[]grades.CreateInput	true	"Grade records"
// @Success		201		{array}		grades.Grade
// @Failure		400		{object}	handlers.ErrorResponse
// @Failure		403		{object}	handlers.ErrorResponse
// @Failure		422		{object}	handlers.ErrorResponse
// @Security		BearerAuth
// @Router			/api/classes/{classID}/grades/bulk [POST]
func (handler *GradeHandler) RecordBulk(c *fiber.Ctx) error {
	inputs, err := batchParser[grades.CreateInput](c, "grades")
	if err != nil {
		return badRequest(c, err)
	}
	classID := c.Params("classID")
	for index := range inputs {
		if inputs[index].ClassID == "" {
			inputs[index].ClassID = classID
		}
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	result, err := handler.service.RecordBulk(c.UserContext(), actorID, role, inputs)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, result)
}

// @Summary	Export grades as CSV
// @Tags		Grade
// @Produce	text/csv
// @Security	BearerAuth
// @Success	200	{string}	string
// @Router		/api/grades/export [GET]
func (handler *GradeHandler) Export(c *fiber.Ctx) error {
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return badRequest(c, err)
	}
	studentID, err := queryUUID(c, "student_id")
	if err != nil {
		return badRequest(c, err)
	}
	result, err := handler.service.List(c.UserContext(), grades.ListParams{ClassID: classID, StudentID: studentID, Term: strings.TrimSpace(c.Query("term")), Type: grades.GradeType(c.Query("type")), Limit: 500})
	if err != nil {
		return writeDomainError(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="grades.csv"`)
	writer := csv.NewWriter(c)
	if err := writer.Write([]string{"student_id", "class_id", "course_id", "term", "quiz_id", "type", "score", "note"}); err != nil {
		return err
	}
	for _, grade := range result {
		quizID := ""
		if grade.QuizID != nil {
			quizID = grade.QuizID.String()
		}
		if err := writer.Write([]string{grade.StudentID.String(), grade.ClassID.String(), grade.CourseID.String(), grade.Term, quizID, string(grade.Type), strconv.FormatFloat(grade.Score, 'f', -1, 64), grade.Note}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// @Summary	Import grades from CSV
// @Tags		Grade
// @Accept		text/csv
// @Produce	json
// @Security	BearerAuth
// @Success	201	{array}	grades.Grade
// @Router		/api/grades/import [POST]
func (handler *GradeHandler) Import(c *fiber.Ctx) error {
	reader := csv.NewReader(strings.NewReader(string(c.Body())))
	reader.FieldsPerRecord = -1
	inputs := make([]grades.CreateInput, 0)
	first := true
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil || len(record) < 8 {
			return writeError(c, fiber.ErrBadRequest)
		}
		if first && strings.EqualFold(strings.TrimSpace(record[0]), "student_id") {
			first = false
			continue
		}
		first = false
		score, parseErr := strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
		if parseErr != nil {
			return writeError(c, fiber.ErrBadRequest)
		}
		inputs = append(inputs, grades.CreateInput{StudentID: strings.TrimSpace(record[0]), ClassID: strings.TrimSpace(record[1]), CourseID: strings.TrimSpace(record[2]), Term: strings.TrimSpace(record[3]), QuizID: strings.TrimSpace(record[4]), Type: strings.TrimSpace(record[5]), Score: score, Note: strings.TrimSpace(record[7])})
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	result, err := handler.service.RecordBulk(c.UserContext(), actorID, role, inputs)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, result)
}

// @Summary		Update Grade
// @Description	Kaydı günceller.
// @Tags			Grade
// @Produce		json
// @Accept			json
// @Param			gradeID	path		string					true	"Not kaydı kimliği (UUID)"
// @Param			request	body		grades.UpdateInput		true	"İstek gövdesi"
// @Success		200		{object}	grades.Grade			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/grades/{gradeID} [PUT]
func (handler *GradeHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[grades.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	entity, err := handler.service.Update(c.UserContext(), actorID, role, c.Params("gradeID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary	List grade audit history
// @Tags		Grade
// @Produce	json
// @Param		gradeID	path	string	true	"Grade UUID"
// @Success	200		{array}	grades.Audit
// @Security	BearerAuth
// @Router		/api/grades/{gradeID}/audit [GET]
func (handler *GradeHandler) Audit(c *fiber.Ctx) error {
	result, err := handler.service.ListAudits(c.UserContext(), c.Params("gradeID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

// @Summary		Delete Grade
// @Description	Kaydı siler.
// @Tags			Grade
// @Produce		json
// @Param			gradeID	path	string	true	"Not kaydı kimliği (UUID)"
// @Success		204		"no content"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/grades/{gradeID} [DELETE]
func (handler *GradeHandler) Delete(c *fiber.Ctx) error {
	actorID, role, err := identity(c)
	if err != nil {
		return err
	}
	if err := handler.service.Delete(c.UserContext(), actorID, role, c.Params("gradeID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		List Grade
// @Description	Kayıtları listeler.
// @Tags			Grade
// @Produce		json
// @Param			term	query		string					false	"term"
// @Param			type	query		string					false	"type"
// @Success		200		{array}		grades.Grade			"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/grades [GET]
// @Router			/api/grades [GET]
// @Router			/api/students/{studentID}/grades [GET]
func (handler *GradeHandler) List(c *fiber.Ctx) error {
	studentID, err := queryUUID(c, "student_id")
	if err != nil {
		return badRequest(c, err)
	}
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return badRequest(c, err)
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	quizID, err := queryUUID(c, "quiz_id")
	if err != nil {
		return badRequest(c, err)
	}
	if classID == nil {
		if classID, err = paramUUID(c, "classID"); err != nil {
			return badRequest(c, err)
		}
	}
	if studentID == nil {
		if studentID, err = paramUUID(c, "studentID"); err != nil {
			return badRequest(c, err)
		}
	}
	if studentID, err = studentSubject(c, studentID); err != nil {
		return err
	}
	if quizID == nil {
		if quizID, err = paramUUID(c, "quizID"); err != nil {
			return badRequest(c, err)
		}
	}
	_, limit := pageParams(c)
	classIDs, applied := apiMiddleware.ScopedClassIDs(c)
	result, err := handler.service.List(c.UserContext(), grades.ListParams{
		ClassIDs:     classIDs,
		ScopeApplied: applied,
		StudentID:    studentID,
		ClassID:      classID,
		CourseID:     courseID,
		QuizID:       quizID,
		Term:         c.Query("term"),
		Type:         grades.GradeType(c.Query("type")),
		Limit:        limit,
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

type NoteHandler struct{ service notes.Service }

func NewNoteHandler(service notes.Service) *NoteHandler { return &NoteHandler{service: service} }

// @Summary		Create Note
// @Description	Notu veren öğrenci sınıfa kayıtlı olmalıdır.
// @Tags			Note
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		notes.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	notes.Note				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/notes [POST]
func (handler *NoteHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[notes.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.ClassID == "" {
		input.ClassID = c.Params("classID")
	}
	if input.CourseID == "" {
		input.CourseID = c.Params("courseID")
	}
	authorID, role, err := identity(c)
	if err != nil {
		return err
	}
	entity, err := handler.service.Create(c.UserContext(), authorID, role, input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, entity)
}

// @Summary		Get Note
// @Description	Tek kaydı döner.
// @Tags			Note
// @Produce		json
// @Param			noteID	path		string					true	"Not kimliği (UUID)"
// @Success		200		{object}	notes.Note				"successful response"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/notes/{noteID} [GET]
func (handler *NoteHandler) Get(c *fiber.Ctx) error {
	entity, err := handler.service.GetByID(c.UserContext(), c.Params("noteID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if !scopeguard.MayReadNote(role, entity.VisibleToStudent) {
		return fiber.ErrNotFound // öğrenci özel öğretmen notunun varlığını bile öğrenmez
	}
	return ok(c, entity)
}

// @Summary		Update Note
// @Description	Kaydı günceller.
// @Tags			Note
// @Produce		json
// @Accept			json
// @Param			noteID	path		string					true	"Not kimliği (UUID)"
// @Param			request	body		notes.UpdateInput		true	"İstek gövdesi"
// @Success		200		{object}	notes.Note				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/notes/{noteID} [PUT]
func (handler *NoteHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[notes.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	authorID, role, err := identity(c)
	if err != nil {
		return err
	}
	entity, err := handler.service.Update(c.UserContext(), authorID, role, c.Params("noteID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, entity)
}

// @Summary		Delete Note
// @Description	Kaydı siler.
// @Tags			Note
// @Produce		json
// @Param			noteID	path	string	true	"Not kimliği (UUID)"
// @Success		204		"no content"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/notes/{noteID} [DELETE]
func (handler *NoteHandler) Delete(c *fiber.Ctx) error {
	authorID, _, err := identity(c)
	if err != nil {
		return err
	}
	if err := handler.service.Delete(c.UserContext(), authorID, c.Params("noteID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		List Note
// @Description	Kayıtları listeler.
// @Tags			Note
// @Produce		json
// @Success		200	{array}		notes.Note				"successful response"
// @Failure		400	{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/notes [GET]
// @Router			/api/notes [GET]
// @Router			/api/students/{studentID}/notes [GET]
func (handler *NoteHandler) List(c *fiber.Ctx) error {
	authorID, err := queryUUID(c, "author_id")
	if err != nil {
		return badRequest(c, err)
	}
	studentID, err := queryUUID(c, "student_id")
	if err != nil {
		return badRequest(c, err)
	}
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return badRequest(c, err)
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	if classID == nil {
		if classID, err = paramUUID(c, "classID"); err != nil {
			return badRequest(c, err)
		}
	}
	if studentID == nil {
		if studentID, err = paramUUID(c, "studentID"); err != nil {
			return badRequest(c, err)
		}
	}
	if studentID, err = studentSubject(c, studentID); err != nil {
		return err
	}
	sharedOnly, err := sharedNotesOnly(c)
	if err != nil {
		return err
	}
	_, limit := pageParams(c)
	classIDs, applied := apiMiddleware.ScopedClassIDs(c)
	result, err := handler.service.List(c.UserContext(), notes.ListParams{
		ClassIDs:     classIDs,
		ScopeApplied: applied,
		AuthorID:     authorID,
		StudentID:    studentID,
		ClassID:      classID,
		CourseID:     courseID,
		SharedOnly:   sharedOnly,
		Limit:        limit,
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

type ActivityHandler struct{ service activities.Service }

func NewActivityHandler(service activities.Service) *ActivityHandler {
	return &ActivityHandler{service: service}
}

// @Summary		Record a student activity
// @Description	Kaydı oluşturur.
// @Tags			Activity
// @Produce		json
// @Success		201	{object}	map[string]interface{}	"successful response"
// @Failure		400	{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401	{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		409	{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422	{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/students/me/activities [POST]
func (handler *ActivityHandler) Record(c *fiber.Ctx) error {
	input, err := bodyParser[struct {
		Events []activities.EventInput `json:"events"`
	}](c)
	if err != nil {
		return badRequest(c, err)
	}
	studentID, err := studentScope(c)
	if err != nil {
		return err
	}
	if err := handler.service.Record(c.UserContext(), studentID, input.Events); err != nil {
		return writeDomainError(c, err)
	}
	return created(c, fiber.Map{"recorded": len(input.Events)})
}

// @Summary		List student activities
// @Description	Kayıtları listeler.
// @Tags			Activity
// @Produce		json
// @Param			type	query		string					false	"type"
// @Success		200		{array}		activities.Activity		"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/students/me/activities [GET]
// @Router			/api/students/{studentID}/activities [GET]
func (handler *ActivityHandler) List(c *fiber.Ctx) error {
	studentID, err := studentScope(c)
	if err != nil {
		return err
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	lessonID, err := queryUUID(c, "lesson_id")
	if err != nil {
		return badRequest(c, err)
	}
	quizID, err := queryUUID(c, "quiz_id")
	if err != nil {
		return badRequest(c, err)
	}
	from, err := queryTime(c, "from")
	if err != nil {
		return badRequest(c, err)
	}
	to, err := queryTime(c, "to")
	if err != nil {
		return badRequest(c, err)
	}
	_, limit := pageParams(c)
	result, err := handler.service.List(c.UserContext(), activities.ListParams{
		StudentID: &studentID,
		CourseID:  courseID,
		LessonID:  lessonID,
		QuizID:    quizID,
		Type:      activities.EventType(c.Query("type")),
		From:      from,
		To:        to,
		Limit:     limit,
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Get daily activity summary
// @Description	Seçilen gün için öğrencinin aktivite özetini döner.
// @Tags			Activity
// @Produce		json
// @Success		200	{array}		activities.DailyAggregate	"successful response"
// @Failure		400	{object}	handlers.ErrorResponse		"geçersiz istek"
// @Failure		401	{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403	{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		422	{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/students/me/activities/daily [GET]
// @Router			/api/students/{studentID}/activities/daily [GET]
func (handler *ActivityHandler) Daily(c *fiber.Ctx) error {
	studentID, err := studentScope(c)
	if err != nil {
		return err
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	from, err := queryDate(c, "from")
	if err != nil {
		return badRequest(c, err)
	}
	to, err := queryDate(c, "to")
	if err != nil {
		return badRequest(c, err)
	}
	result, err := handler.service.ListDaily(c.UserContext(), activities.DailyParams{
		StudentID: studentID,
		CourseID:  courseID,
		From:      derefString(from),
		To:        derefString(to),
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, result)
}

func studentScope(c *fiber.Ctx) (uuid.UUID, error) {
	value := c.Params("studentID")
	if value == "" {
		userID, role, err := identity(c)
		if err != nil {
			return uuid.Nil, err
		}
		if role != users.RoleStudent {
			return uuid.Nil, fiber.ErrForbidden
		}
		return userID, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fiber.ErrNotFound
	}
	return parsed, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
