package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/exams"
	"lms/backend/pkg/domain/users"
)

// ExamHandler, sınavın bütünsel iş akışı uçlarıdır: personel için atomik
// oluşturma/düzenleme/onay/sonuç, öğrenci için cevap anahtarsız çözme ve sonuç.
type ExamHandler struct {
	exams    exams.Service
	attempts attempts.Service
}

func NewExamHandler(examService exams.Service, attemptService attempts.Service) *ExamHandler {
	return &ExamHandler{exams: examService, attempts: attemptService}
}

type examReviewInput struct {
	Action approvals.Action `json:"action"`
	Note   string           `json:"note"`
}

type takeQuiz struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	MaxAttempts     int        `json:"max_attempts"`
	CloseAt         *time.Time `json:"close_at"`
}

// @Summary		Compose an exam
// @Description	Sınavı sorular ve seçenekleriyle birlikte tek işlemde taslak olarak oluşturur. Eksik içerik taslakta serbesttir; yayın/onay için tam doğrulama gerekir.
// @Tags			Exam
// @Accept			json
// @Produce		json
// @Param			classID	path		string		true	\"Sınıf kimliği (UUID)\"
// @Param			request	body		exams.Input	true	\"Sınav, sorular ve seçenekler\"
// @Success		201			{object}	exams.Detail	\"successful response\"
// @Failure		422			{object}	handlers.ErrorResponse	\"doğrulama hatası (issues listesi ile)\"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/exams [POST]
func (handler *ExamHandler) Compose(c *fiber.Ctx) error {
	input, err := bodyParser[exams.Input](c)
	if err != nil {
		return badRequest(c, err)
	}
	actor, err := examActor(c)
	if err != nil {
		return err
	}
	detail, err := handler.exams.Compose(c.UserContext(), actor, c.Params("classID"), input)
	if err != nil {
		return writeExamError(c, err)
	}
	return created(c, detail)
}

// @Summary		Get an exam (staff view)
// @Description	Sınavı cevap anahtarı, yayın öncesi eksik listesi ve kilit durumuyla döner (yalnızca personel).
// @Tags			Exam
// @Produce		json
// @Param			classID	path		string	true	\"Sınıf kimliği (UUID)\"
// @Param			quizID	path		string	true	\"Sınav kimliği (UUID)\"
// @Success		200			{object}	exams.Detail	\"successful response\"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/exams/{quizID} [GET]
func (handler *ExamHandler) Get(c *fiber.Ctx) error {
	detail, err := handler.exams.Detail(c.UserContext(), c.Params("quizID"))
	if err != nil {
		return writeExamError(c, err)
	}
	return ok(c, detail)
}

// @Summary		Replace an exam
// @Description	Sınav bilgilerini ve tüm soru/seçenekleri atomik olarak yeniden yazar; sınav taslağa döner. Çözülmeye başlanmış sınavın soruları değiştirilemez (403).
// @Tags			Exam
// @Accept			json
// @Produce		json
// @Param			classID	path		string		true	\"Sınıf kimliği (UUID)\"
// @Param			quizID	path		string		true	\"Sınav kimliği (UUID)\"
// @Param			request	body		exams.Input	true	\"Sınav, sorular ve seçenekler\"
// @Success		200			{object}	exams.Detail	\"successful response\"
// @Failure		403			{object}	handlers.ErrorResponse	\"sınav çözülmeye başlandı\"
// @Failure		422			{object}	handlers.ErrorResponse	\"doğrulama hatası (issues listesi ile)\"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/exams/{quizID} [PUT]
func (handler *ExamHandler) Replace(c *fiber.Ctx) error {
	input, err := bodyParser[exams.Input](c)
	if err != nil {
		return badRequest(c, err)
	}
	actor, err := examActor(c)
	if err != nil {
		return err
	}
	detail, err := handler.exams.Replace(c.UserContext(), actor, c.Params("quizID"), input)
	if err != nil {
		return writeExamError(c, err)
	}
	return ok(c, detail)
}

// @Summary		Review an exam
// @Description	Action: submit (onaya gönder), approve, reject. Onay/ret yalnızca yöneticidir. submit ve approve tam doğrulama ister; sorular sınavın durumunu izler.
// @Tags			Exam
// @Accept			json
// @Produce		json
// @Param			classID	path		string				true	\"Sınıf kimliği (UUID)\"
// @Param			quizID	path		string				true	\"Sınav kimliği (UUID)\"
// @Param			request	body		handlers.examReviewInput	true	\"Karar\"
// @Success		200			{object}	exams.Detail	\"successful response\"
// @Failure		403			{object}	handlers.ErrorResponse	\"yetkisiz işlem\"
// @Failure		422			{object}	handlers.ErrorResponse	\"eksik içerik (issues listesi ile)\"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/exams/{quizID}/review [POST]
func (handler *ExamHandler) Review(c *fiber.Ctx) error {
	input, err := bodyParser[examReviewInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	actor, err := examActor(c)
	if err != nil {
		return err
	}
	detail, err := handler.exams.Review(c.UserContext(), actor, c.Params("quizID"), input.Action, input.Note)
	if err != nil {
		return writeExamError(c, err)
	}
	return ok(c, detail)
}

// @Summary		Exam results (staff)
// @Description	Tamamlanmış denemelerden öğrenci bazlı özet ve soru bazlı doğru oranı.
// @Tags			Exam
// @Produce		json
// @Param			classID	path		string	true	\"Sınıf kimliği (UUID)\"
// @Param			quizID	path		string	true	\"Sınav kimliği (UUID)\"
// @Success		200			{object}	exams.Summary	\"successful response\"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/exams/{quizID}/results [GET]
func (handler *ExamHandler) Results(c *fiber.Ctx) error {
	summary, err := handler.exams.Results(c.UserContext(), c.Params("quizID"))
	if err != nil {
		return writeExamError(c, err)
	}
	return ok(c, summary)
}

// @Summary		Exam for an in-progress attempt
// @Description	Sürmekte olan denemenin soruları ve seçenekleri (cevap anahtarsız), kayıtlı cevaplar, sunucu saati ve bitiş anı. Deneme bittiyse 409 döner.
// @Tags			Attempt
// @Produce		json
// @Param			attemptID	path		string	true	\"Deneme kimliği (UUID)\"
// @Success		200				{object}	object	\"successful response\"
// @Failure		409				{object}	handlers.ErrorResponse	\"deneme sürmüyor\"
// @Security		BearerAuth
// @Router			/api/attempts/{attemptID}/exam [GET]
func (handler *ExamHandler) Take(c *fiber.Ctx) error {
	attempt, err := handler.attempts.GetByID(c.UserContext(), c.Params("attemptID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	if attempt.Status != attempts.StatusInProgress {
		return writeConflict(c, "attempt is not in progress")
	}
	detail, err := handler.exams.DetailFor(c.UserContext(), attempt.QuizID)
	if err != nil {
		return writeDomainError(c, err)
	}
	var deadline *time.Time
	if at, limited := detail.Quiz.Deadline(attempt.StartedAt); limited {
		deadline = &at
	}
	selected := make(map[string]string, len(attempt.Answers))
	for _, answer := range attempt.Answers {
		if answer.SelectedOptionID != nil {
			selected[answer.QuestionID.String()] = answer.SelectedOptionID.String()
		}
	}
	return ok(c, fiber.Map{
		"attempt": attempt.View(false),
		"quiz": takeQuiz{
			ID:              detail.Quiz.ID.String(),
			Name:            detail.Quiz.Name,
			Description:     detail.Quiz.Description,
			DurationMinutes: detail.Quiz.DurationMinutes,
			MaxAttempts:     detail.Quiz.MaxAttempts,
			CloseAt:         detail.Quiz.CloseAt,
		},
		"deadline":    deadline,
		"server_time": time.Now().UTC(),
		"questions":   detail.ForStudent(),
		"answers":     selected,
	})
}

// @Summary		Attempt result
// @Description	Tamamlanmış denemenin sonucu. Doğru cevaplar öğrenciye yalnızca sınav kapandıysa ya da deneme hakkı bittiyse açılır (answers_revealed).
// @Tags			Attempt
// @Produce		json
// @Param			attemptID	path		string	true	\"Deneme kimliği (UUID)\"
// @Success		200				{object}	exams.Result	\"successful response\"
// @Failure		409				{object}	handlers.ErrorResponse	\"deneme henüz tamamlanmadı\"
// @Security		BearerAuth
// @Router			/api/attempts/{attemptID}/result [GET]
func (handler *ExamHandler) Result(c *fiber.Ctx) error {
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	attempt, err := handler.attempts.GetByID(c.UserContext(), c.Params("attemptID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	if attempt.Status != attempts.StatusCompleted {
		return writeConflict(c, "attempt is not completed")
	}
	detail, err := handler.exams.DetailFor(c.UserContext(), attempt.QuizID)
	if err != nil {
		return writeDomainError(c, err)
	}
	reveal := role != users.RoleStudent
	if !reveal {
		studentID, quizID := attempt.StudentID, attempt.QuizID
		completed, err := handler.attempts.List(c.UserContext(), attempts.ListParams{StudentID: &studentID, QuizID: &quizID, Status: attempts.StatusCompleted, Limit: 100})
		if err != nil {
			return writeDomainError(c, err)
		}
		reveal = exams.ShouldReveal(detail.Quiz, len(completed), time.Now().UTC())
	}
	return ok(c, exams.BuildResult(*attempt, *detail, reveal))
}

func examActor(c *fiber.Ctx) (exams.Actor, error) {
	userID, role, err := identity(c)
	if err != nil {
		return exams.Actor{}, err
	}
	return exams.Actor{ID: userID, Role: role}, nil
}

// writeExamError, doğrulama hatalarında ayrıntılı eksik listesini (issues) ekler.
func writeExamError(c *fiber.Ctx, err error) error {
	var validation *exams.ValidationError
	if errors.As(err, &validation) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"code":    fiber.ErrUnprocessableEntity.Message,
				"message": "validation failed",
				"issues":  validation.Issues,
			},
		})
	}
	return writeDomainError(c, err)
}

func writeConflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"data":  nil,
		"error": fiber.Map{"code": fiber.ErrConflict.Message, "message": message},
	})
}

// denyStudent, öğrencinin soru/seçenek gibi cevap anahtarına yakın uçlara erişimini
// engeller; öğrenci soruları yalnızca kendi denemesi üzerinden görür.
func denyStudent(c *fiber.Ctx) error {
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleStudent {
		return fiber.ErrForbidden
	}
	return nil
}

// requireAdminForDecision, onay ve reddi yöneticiye ayırır; öğretmen yalnızca onaya gönderebilir.
func requireAdminForDecision(c *fiber.Ctx, action approvals.Action) error {
	if action != approvals.ActionApprove && action != approvals.ActionReject {
		return nil
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role != users.RoleAdmin && role != users.RoleSuperAdmin {
		return fiber.ErrForbidden
	}
	return nil
}

// attemptResponse, öğrenciye cevap anahtarsız görünüm, personele tam kaydı döner.
func attemptResponse(c *fiber.Ctx, attempt *attempts.QuizAttempt) (any, error) {
	_, role, err := identity(c)
	if err != nil {
		return nil, err
	}
	if role == users.RoleStudent {
		return attempt.View(false), nil
	}
	return attempt, nil
}
