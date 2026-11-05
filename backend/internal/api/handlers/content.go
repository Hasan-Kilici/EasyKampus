package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apiMiddleware "lms/backend/internal/api/middleware"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/users"
)

type LessonHandler struct{ service lessons.Service }

func NewLessonHandler(service lessons.Service) *LessonHandler {
	return &LessonHandler{service: service}
}

// @Summary		Create Lesson
// @Description	Kaydı oluşturur.
// @Tags			Lesson
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			request		body		lessons.CreateInput		true	"İstek gövdesi"
// @Success		201			{object}	lessons.Lesson			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons [POST]
func (handler *LessonHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[lessons.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.CourseID == "" {
		courseID, err := coursePath(c)
		if err != nil {
			return err
		}
		input.CourseID = courseID.String()
	}
	if input.CreatedByID == "" || input.CreatedByRole == "" {
		userID, role, err := identity(c)
		if err != nil {
			return err
		}
		input.CreatedByID = userID.String()
		input.CreatedByRole = role
	}
	lesson, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, lesson)
}

// @Summary		List Lesson
// @Description	Kayıtları listeler.
// @Tags			Lesson
// @Produce		json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Success		200			{array}		lessons.Lesson			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons [GET]
func (handler *LessonHandler) List(c *fiber.Ctx) error {
	courseID, err := coursePath(c)
	if err != nil {
		return err
	}
	result, err := handler.service.ListByCourse(c.UserContext(), courseID.String(), queryBool(c, "published"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Get Lesson
// @Description	Tek kaydı döner.
// @Tags			Lesson
// @Produce		json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Success		200			{object}	lessons.Lesson			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons/{lessonID} [GET]
func (handler *LessonHandler) Get(c *fiber.Ctx) error {
	lesson, err := handler.service.GetByID(c.UserContext(), c.Params("lessonID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, lesson)
}

// @Summary		Update Lesson
// @Description	Kaydı günceller.
// @Tags			Lesson
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param			request		body		lessons.UpdateInput		true	"İstek gövdesi"
// @Success		200			{object}	lessons.Lesson			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons/{lessonID} [PUT]
func (handler *LessonHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[lessons.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	lesson, err := handler.service.Update(c.UserContext(), c.Params("lessonID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, lesson)
}

// @Summary		Delete Lesson
// @Description	Kaydı siler.
// @Tags			Lesson
// @Produce		json
// @Param			courseID	path	string	true	"Ders kimliği (UUID)"
// @Param			lessonID	path	string	true	"Ders içeriği kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons/{lessonID} [DELETE]
func (handler *LessonHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("lessonID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		Review a lesson for approval
// @Description	İçerik onayını yönetir.
// @Tags			Lesson
// @Produce		json
// @Accept			json
// @Param			courseID	path		string					true	"Ders kimliği (UUID)"
// @Param			lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param			request		body		lessons.ReviewInput		true	"İstek gövdesi"
// @Success		200			{object}	lessons.Lesson			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons/{lessonID}/review [POST]
func (handler *LessonHandler) Review(c *fiber.Ctx) error {
	input, err := bodyParser[lessons.ReviewInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.ReviewerID == "" {
		userID, _, err := identity(c)
		if err != nil {
			return err
		}
		input.ReviewerID = userID.String()
	}
	lesson, err := handler.service.Review(c.UserContext(), c.Params("lessonID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, lesson)
}

// @Summary	View lesson content
// @Tags		Lesson
// @Produce	json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Success	200			{object}	lessons.Lesson			"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure	422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/view [POST]
func (handler *LessonHandler) View(c *fiber.Ctx) error {
	studentID, err := studentIdentity(c)
	if err != nil {
		return err
	}
	lesson, err := handler.service.View(c.UserContext(), studentID.String(), c.Params("lessonID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, lesson)
}

// @Summary		Mark a lesson as completed
// @Description	Ders içeriğini tamamlanmış işaretler.
// @Tags			Lesson
// @Produce		json
// @Param			courseID	path		string						true	"Ders kimliği (UUID)"
// @Param			lessonID	path		string						true	"Ders içeriği kimliği (UUID)"
// @Success		200			{object}	lessons.LessonCompletion	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse		"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse		"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse		"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse		"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/courses/{courseID}/lessons/{lessonID}/complete [POST]
func (handler *LessonHandler) Complete(c *fiber.Ctx) error {
	studentID, err := studentIdentity(c)
	if err != nil {
		return err
	}
	completion, err := handler.service.Complete(c.UserContext(), studentID.String(), c.Params("lessonID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, completion)
}

type QuizHandler struct{ service quizes.Service }

func NewQuizHandler(service quizes.Service) *QuizHandler { return &QuizHandler{service: service} }

// @Summary		Create Quiz
// @Description	Kaydı oluşturur.
// @Tags			Quiz
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			request	body		quizes.CreateInput		true	"İstek gövdesi"
// @Success		201		{object}	quizes.Quiz				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes [POST]
func (handler *QuizHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[quizes.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.ClassID == "" {
		input.ClassID = c.Params("classID")
	}
	if input.CourseID == "" {
		input.CourseID = c.Params("courseID")
	}
	if input.CreatedByID == "" || input.CreatedByRole == "" {
		userID, role, err := identity(c)
		if err != nil {
			return err
		}
		input.CreatedByID = userID.String()
		input.CreatedByRole = role
	}
	quiz, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, quiz)
}

// @Summary		List Quiz
// @Description	Kayıtları listeler.
// @Tags			Quiz
// @Produce		json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Success		200		{array}		quizes.Quiz				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes [GET]
func (handler *QuizHandler) List(c *fiber.Ctx) error {
	classID, err := uuid.Parse(c.Params("classID"))
	if err != nil {
		return fiber.ErrNotFound
	}
	courseID, err := queryUUID(c, "course_id")
	if err != nil {
		return badRequest(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	// Öğrenci taslak/onaysız sınavları göremez; filtre istemci parametresine bırakılmaz.
	published := queryBool(c, "published") || role == users.RoleStudent
	result, err := handler.service.ListByClass(c.UserContext(), classID.String(), published)
	if err != nil {
		return writeDomainError(c, err)
	}
	if courseID != nil {
		filtered := make([]quizes.Quiz, 0, len(result))
		for _, quiz := range result {
			if quiz.CourseID == *courseID {
				filtered = append(filtered, quiz)
			}
		}
		result = filtered
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Get Quiz
// @Description	Tek kaydı döner.
// @Tags			Quiz
// @Produce		json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID	path		string					true	"Quiz kimliği (UUID)"
// @Success		200		{object}	quizes.Quiz				"successful response"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID} [GET]
func (handler *QuizHandler) Get(c *fiber.Ctx) error {
	quiz, err := handler.service.GetByID(c.UserContext(), c.Params("quizID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleStudent && !quiz.State.IsPublished() {
		return fiber.ErrNotFound
	}
	return ok(c, quiz)
}

// @Summary		Update Quiz
// @Description	Kaydı günceller.
// @Tags			Quiz
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID	path		string					true	"Quiz kimliği (UUID)"
// @Param			request	body		quizes.UpdateInput		true	"İstek gövdesi"
// @Success		200		{object}	quizes.Quiz				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID} [PUT]
func (handler *QuizHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[quizes.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	quiz, err := handler.service.Update(c.UserContext(), c.Params("quizID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, quiz)
}

// @Summary		Delete Quiz
// @Description	Kaydı siler.
// @Tags			Quiz
// @Produce		json
// @Param			classID	path	string	true	"Sınıf kimliği (UUID)"
// @Param			quizID	path	string	true	"Quiz kimliği (UUID)"
// @Success		204		"no content"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID} [DELETE]
func (handler *QuizHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("quizID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		Review a quiz for approval
// @Description	İçerik onayını yönetir.
// @Tags			Quiz
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID	path		string					true	"Quiz kimliği (UUID)"
// @Param			request	body		quizes.ReviewInput		true	"İstek gövdesi"
// @Success		200		{object}	quizes.Quiz				"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/review [POST]
func (handler *QuizHandler) Review(c *fiber.Ctx) error {
	input, err := bodyParser[quizes.ReviewInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if err := requireAdminForDecision(c, input.Action); err != nil {
		return err
	}
	if input.ReviewerID == "" {
		userID, _, err := identity(c)
		if err != nil {
			return err
		}
		input.ReviewerID = userID.String()
	}
	quiz, err := handler.service.Review(c.UserContext(), c.Params("quizID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, quiz)
}

type QuestionHandler struct{ service questions.Service }

func NewQuestionHandler(service questions.Service) *QuestionHandler {
	return &QuestionHandler{service: service}
}

// @Summary		Create Question
// @Description	Kaydı oluşturur.
// @Tags			Question
// @Produce		json
// @Accept			json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID	path		string					true	"Quiz kimliği (UUID)"
// @Param			request	body		questions.CreateInput	true	"İstek gövdesi"
// @Success		201		{object}	questions.Question		"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409		{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions [POST]
func (handler *QuestionHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[questions.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.QuizID == "" {
		input.QuizID = c.Params("quizID")
	}
	if input.CreatedByID == "" || input.CreatedByRole == "" {
		userID, role, err := identity(c)
		if err != nil {
			return err
		}
		input.CreatedByID = userID.String()
		input.CreatedByRole = role
	}
	question, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, question)
}

// @Summary		List Question
// @Description	Kayıtları listeler.
// @Tags			Question
// @Produce		json
// @Param			classID	path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID	path		string					true	"Quiz kimliği (UUID)"
// @Success		200		{array}		questions.Question		"successful response"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404		{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions [GET]
func (handler *QuestionHandler) List(c *fiber.Ctx) error {
	if err := denyStudent(c); err != nil {
		return err
	}
	result, err := handler.service.ListByQuiz(c.UserContext(), c.Params("quizID"), queryBool(c, "published"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Get Question
// @Description	Tek kaydı döner.
// @Tags			Question
// @Produce		json
// @Param			classID		path		string					true		"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true		"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true		"Soru kimliği (UUID)"
// @Success		200			{object}	courseID				uuid.UUID	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID} [GET]
func (handler *QuestionHandler) Get(c *fiber.Ctx) error {
	if err := denyStudent(c); err != nil {
		return err
	}
	question, err := handler.service.GetByID(c.UserContext(), c.Params("questionID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, question)
}

// @Summary		Update Question
// @Description	Kaydı günceller.
// @Tags			Question
// @Produce		json
// @Accept			json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Param			request		body		questions.UpdateInput	true	"İstek gövdesi"
// @Success		200			{object}	questions.Question		"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID} [PUT]
func (handler *QuestionHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[questions.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	question, err := handler.service.Update(c.UserContext(), c.Params("questionID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, question)
}

// @Summary		Delete Question
// @Description	Kaydı siler.
// @Tags			Question
// @Produce		json
// @Param			classID		path	string	true	"Sınıf kimliği (UUID)"
// @Param			quizID		path	string	true	"Quiz kimliği (UUID)"
// @Param			questionID	path	string	true	"Soru kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID} [DELETE]
func (handler *QuestionHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("questionID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary		Review a question for approval
// @Description	İçerik onayını yönetir.
// @Tags			Question
// @Produce		json
// @Accept			json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Param			request		body		questions.ReviewInput	true	"İstek gövdesi"
// @Success		200			{object}	questions.Question		"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/review [POST]
func (handler *QuestionHandler) Review(c *fiber.Ctx) error {
	input, err := bodyParser[questions.ReviewInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if err := requireAdminForDecision(c, input.Action); err != nil {
		return err
	}
	if input.ReviewerID == "" {
		userID, _, err := identity(c)
		if err != nil {
			return err
		}
		input.ReviewerID = userID.String()
	}
	question, err := handler.service.Review(c.UserContext(), c.Params("questionID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, question)
}

type OptionHandler struct{ service options.Service }

func NewOptionHandler(service options.Service) *OptionHandler {
	return &OptionHandler{service: service}
}

// @Summary		Create Option
// @Description	Kaydı oluşturur.
// @Tags			Option
// @Produce		json
// @Accept			json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Param			request		body		options.CreateInput		true	"İstek gövdesi"
// @Success		201			{object}	options.Option			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/options [POST]
func (handler *OptionHandler) Create(c *fiber.Ctx) error {
	input, err := bodyParser[options.CreateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.QuestionID == "" {
		input.QuestionID = c.Params("questionID")
	}
	option, err := handler.service.Create(c.UserContext(), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return created(c, option)
}

// @Summary		List Option
// @Description	Kayıtları listeler.
// @Tags			Option
// @Produce		json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Success		200			{array}		options.Option			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/options [GET]
func (handler *OptionHandler) List(c *fiber.Ctx) error {
	if err := denyStudent(c); err != nil {
		return err
	}
	page, limit := pageParams(c)
	result, err := handler.service.ListByQuestion(c.UserContext(), c.Params("questionID"), page, limit)
	if err != nil {
		return writeDomainError(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleStudent {
		return list(c, redactOptionsForStudent(result), int64(len(result)))
	}
	return list(c, result, int64(len(result)))
}

// @Summary		Get Option
// @Description	Tek kaydı döner.
// @Tags			Option
// @Produce		json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Param			optionID	path		string					true	"Şık kimliği (UUID)"
// @Success		200			{object}	options.Option			"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/options/{optionID} [GET]
func (handler *OptionHandler) Get(c *fiber.Ctx) error {
	if err := denyStudent(c); err != nil {
		return err
	}
	option, err := handler.service.GetByID(c.UserContext(), c.Params("optionID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleStudent {
		return ok(c, redactOptionForStudent(*option))
	}
	return ok(c, option)
}

// @Summary		Update Option
// @Description	Kaydı günceller.
// @Tags			Option
// @Produce		json
// @Accept			json
// @Param			classID		path		string					true	"Sınıf kimliği (UUID)"
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			questionID	path		string					true	"Soru kimliği (UUID)"
// @Param			optionID	path		string					true	"Şık kimliği (UUID)"
// @Param			request		body		options.UpdateInput		true	"İstek gövdesi"
// @Success		200			{object}	options.Option			"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/options/{optionID} [PUT]
func (handler *OptionHandler) Update(c *fiber.Ctx) error {
	input, err := bodyParser[options.UpdateInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	if input.QuestionID == "" {
		input.QuestionID = c.Params("questionID")
	}
	option, err := handler.service.Update(c.UserContext(), c.Params("optionID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, option)
}

// @Summary		Delete Option
// @Description	Kaydı siler.
// @Tags			Option
// @Produce		json
// @Param			classID		path	string	true	"Sınıf kimliği (UUID)"
// @Param			quizID		path	string	true	"Quiz kimliği (UUID)"
// @Param			questionID	path	string	true	"Soru kimliği (UUID)"
// @Param			optionID	path	string	true	"Şık kimliği (UUID)"
// @Success		204			"no content"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/classes/{classID}/quizzes/{quizID}/questions/{questionID}/options/{optionID} [DELETE]
func (handler *OptionHandler) Delete(c *fiber.Ctx) error {
	if err := handler.service.Delete(c.UserContext(), c.Params("optionID")); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

type AttemptHandler struct{ service attempts.Service }

func NewAttemptHandler(service attempts.Service) *AttemptHandler {
	return &AttemptHandler{service: service}
}

// @Summary		Start a quiz attempt
// @Description	Denemeyi başlatır ve kimlik döner.
// @Tags			Attempt
// @Produce		json
// @Param			quizID		path		string					true	"Quiz kimliği (UUID)"
// @Param			student_id	query		string					false	"Sonuçları öğrenciye göre filtrele"
// @Success		201			{object}	attempts.QuizAttempt	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attempts/quizzes/{quizID}/start [POST]
func (handler *AttemptHandler) Start(c *fiber.Ctx) error {
	userID, role, err := identity(c)
	if err != nil {
		return err
	}
	studentID := userID.String()
	if role != users.RoleStudent {
		target := c.Params("studentID")
		if target == "" {
			target = c.Query("student_id")
		}
		if target == "" {
			return badRequest(c, errors.New("student_id is required for staff initiated attempts"))
		}
		parsed, err := uuid.Parse(target)
		if err != nil {
			return badRequest(c, err)
		}
		studentID = parsed.String()
	}
	attempt, err := handler.service.Start(c.UserContext(), studentID, c.Params("quizID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	response, err := attemptResponse(c, attempt)
	if err != nil {
		return err
	}
	return created(c, response)
}

// @Summary		Save an attempt answer
// @Description	Yanıtı kaydeder.
// @Tags			Attempt
// @Produce		json
// @Accept			json
// @Param			attemptID	path		string					true	"Deneme kimliği (UUID)"
// @Param			request		body		attempts.AnswerInput	true	"İstek gövdesi"
// @Success		200			{object}	attempts.AttemptAnswer	"successful response"
// @Failure		400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attempts/{attemptID}/answers [PUT]
func (handler *AttemptHandler) Answer(c *fiber.Ctx) error {
	input, err := bodyParser[attempts.AnswerInput](c)
	if err != nil {
		return badRequest(c, err)
	}
	answer, err := handler.service.Answer(c.UserContext(), c.Params("attemptID"), input)
	if err != nil {
		return writeDomainError(c, err)
	}
	_, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleStudent {
		return ok(c, answer.View(false)) // doğru/yanlış bilgisi sınav sürerken öğrenciye verilmez
	}
	return ok(c, answer)
}

// @Summary		Submit an attempt for grading
// @Description	Denemeyi teslim eder ve puanı hesaplar.
// @Tags			Attempt
// @Produce		json
// @Param			attemptID	path		string					true	"Deneme kimliği (UUID)"
// @Success		200			{object}	attempts.QuizAttempt	"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		409			{object}	handlers.ErrorResponse	"kayıt zaten mevcut"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attempts/{attemptID}/submit [POST]
func (handler *AttemptHandler) Submit(c *fiber.Ctx) error {
	attempt, err := handler.service.Submit(c.UserContext(), c.Params("attemptID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	response, err := attemptResponse(c, attempt)
	if err != nil {
		return err
	}
	return ok(c, response)
}

// @Summary		Get an attempt with answers
// @Description	Tek kaydı döner.
// @Tags			Attempt
// @Produce		json
// @Param			attemptID	path		string					true	"Deneme kimliği (UUID)"
// @Success		200			{object}	attempts.QuizScope		"successful response"
// @Failure		401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Failure		422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attempts/{attemptID} [GET]
func (handler *AttemptHandler) Get(c *fiber.Ctx) error {
	attempt, err := handler.service.GetByID(c.UserContext(), c.Params("attemptID"))
	if err != nil {
		return writeDomainError(c, err)
	}
	response, err := attemptResponse(c, attempt)
	if err != nil {
		return err
	}
	return ok(c, response)
}

// @Summary		List attempts
// @Description	Kayıtları listeler.
// @Tags			Attempt
// @Produce		json
// @Param			status	query		string					false	"Durum filtresi"
// @Success		200		{array}		attempts.QuizAttempt	"successful response"
// @Failure		400		{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure		401		{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure		403		{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure		422		{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security		BearerAuth
// @Router			/api/attempts [GET]
// @Router			/api/classes/{classID}/attempts [GET]
func (handler *AttemptHandler) List(c *fiber.Ctx) error {
	studentID, err := queryUUID(c, "student_id")
	if err != nil {
		return badRequest(c, err)
	}
	quizID, err := queryUUID(c, "quiz_id")
	if err != nil {
		return badRequest(c, err)
	}
	classID, err := queryUUID(c, "class_id")
	if err != nil {
		return badRequest(c, err)
	}
	if studentID, err = studentSubject(c, studentID); err != nil {
		return err
	}
	_, limit := pageParams(c)
	classIDs, applied := apiMiddleware.ScopedClassIDs(c)
	result, err := handler.service.List(c.UserContext(), attempts.ListParams{
		ClassIDs:     classIDs,
		ScopeApplied: applied,
		StudentID:    studentID,
		QuizID:       quizID,
		ClassID:      classID,
		Status:       attempts.Status(c.Query("status")),
		Limit:        limit,
	})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, result, int64(len(result)))
}

func coursePath(c *fiber.Ctx) (uuid.UUID, error) {
	parsed, err := uuid.Parse(c.Params("courseID"))
	if err != nil {
		return uuid.Nil, fiber.ErrNotFound
	}
	return parsed, nil
}

func redactOptionsForStudent(result []options.Option) []fiber.Map {
	redacted := make([]fiber.Map, 0, len(result))
	for _, option := range result {
		redacted = append(redacted, redactOptionForStudent(option))
	}
	return redacted
}

func redactOptionForStudent(option options.Option) fiber.Map {
	return fiber.Map{
		"id":          option.ID,
		"question_id": option.QuestionID,
		"content":     option.Content,
		"position":    option.Position,
		"created_at":  option.CreatedAt,
		"updated_at":  option.UpdatedAt,
	}
}
