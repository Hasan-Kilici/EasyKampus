package exams

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/users"
)

// Actor, işlemi yapan kullanıcıdır.
type Actor struct {
	ID   uuid.UUID
	Role users.Role
}

func (actor Actor) isAdmin() bool {
	return actor.Role == users.RoleAdmin || actor.Role == users.RoleSuperAdmin
}

// ValidationError, kullanıcıya gösterilecek ayrıntılı eksikleri taşır ve
// core.ErrValidation olarak da tanınır (errors.Is).
type ValidationError struct{ Issues []Issue }

func (err *ValidationError) Error() string { return "exam validation failed" }

func (err *ValidationError) Is(target error) bool { return target == core.ErrValidation }

// Tx, tek bir veritabanı işlemi içindeki yazma/okuma adımlarıdır.
type Tx interface {
	CreateQuiz(quiz *quizes.Quiz) error
	SaveQuiz(quiz *quizes.Quiz) error
	GetQuiz(id uuid.UUID) (*quizes.Quiz, error)
	AttemptCount(quizID uuid.UUID) (int64, error)
	DeleteQuestions(quizID uuid.UUID) error
	CreateQuestion(question *questions.Question) error
	CreateOption(option *options.Option) error
	SetQuestionsState(quizID uuid.UUID, review approvals.Review) error
	LoadDetail(quizID uuid.UUID) (*Detail, error)
}

// Store, kalıcılık sınırıdır. InTx hata dönerse tüm yazmalar geri alınmalıdır.
type Store interface {
	InTx(ctx context.Context, fn func(tx Tx) error) error
	LoadDetail(ctx context.Context, quizID uuid.UUID) (*Detail, error)
	AttemptCount(ctx context.Context, quizID uuid.UUID) (int64, error)
	QuestionStats(ctx context.Context, quizID uuid.UUID) ([]QuestionStat, error)
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CourseOfferLookup interface {
	ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error)
}

type TopicLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (courseID uuid.UUID, err error)
}

// Service, sınavın bütünsel iş akışıdır.
type Service interface {
	Compose(ctx context.Context, actor Actor, classID string, input Input) (*Detail, error)
	Replace(ctx context.Context, actor Actor, quizID string, input Input) (*Detail, error)
	Review(ctx context.Context, actor Actor, quizID string, action approvals.Action, note string) (*Detail, error)
	Detail(ctx context.Context, quizID string) (*Detail, error)
	DetailFor(ctx context.Context, quizID uuid.UUID) (*Detail, error)
	Results(ctx context.Context, quizID string) (*Summary, error)
}

type service struct {
	store       Store
	classes     ClassLookup
	offers      CourseOfferLookup
	topics      TopicLookup
	attempts    attempts.Service
	invalidator performance.Invalidator
	cache       core.Cache
	now         func() time.Time
}

func NewService(store Store, classes ClassLookup, offers CourseOfferLookup, topics TopicLookup, attemptService attempts.Service, invalidator performance.Invalidator, caches ...core.Cache) Service {
	var cache core.Cache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &service{
		store: store, classes: classes, offers: offers, topics: topics, attempts: attemptService,
		invalidator: invalidator, cache: cache, now: func() time.Time { return time.Now().UTC() },
	}
}

func (service *service) Compose(ctx context.Context, actor Actor, classID string, input Input) (*Detail, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	if issues := CheckDraft(input); len(issues) > 0 {
		return nil, &ValidationError{Issues: issues}
	}
	exists, err := service.classes.Exists(ctx, parsedClassID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	offered, err := service.offers.ClassCourseExists(ctx, parsedClassID, courseID)
	if err != nil {
		return nil, err
	}
	if !offered {
		return nil, core.ErrValidation
	}
	topicIDs, err := service.resolveTopics(ctx, input.Questions, courseID)
	if err != nil {
		return nil, err
	}

	quiz := &quizes.Quiz{
		ClassID:         parsedClassID,
		CourseID:        courseID,
		Name:            strings.TrimSpace(input.Name),
		Description:     strings.TrimSpace(input.Description),
		OpenAt:          input.OpenAt,
		CloseAt:         input.CloseAt,
		DurationMinutes: input.DurationMinutes,
		Points:          TotalPoints(input),
		MaxAttempts:     input.MaxAttempts,
		CreatedByID:     actor.ID,
	}
	quiz.State = approvals.StateDraft
	if err := quiz.Validate(); err != nil {
		return nil, err
	}
	err = service.store.InTx(ctx, func(tx Tx) error {
		if err := tx.CreateQuiz(quiz); err != nil {
			return err
		}
		return writeQuestions(tx, quiz, actor.ID, input.Questions, topicIDs)
	})
	if err != nil {
		return nil, err
	}
	service.invalidate(ctx, quiz.ClassID, quiz.CourseID)
	return service.DetailFor(ctx, quiz.ID)
}

func (service *service) Replace(ctx context.Context, actor Actor, quizID string, input Input) (*Detail, error) {
	id, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	if issues := CheckDraft(input); len(issues) > 0 {
		return nil, &ValidationError{Issues: issues}
	}
	var classID, courseID uuid.UUID
	err = service.store.InTx(ctx, func(tx Tx) error {
		quiz, err := tx.GetQuiz(id)
		if err != nil {
			return err
		}
		classID, courseID = quiz.ClassID, quiz.CourseID
		if input.CourseID != "" {
			requested, err := parseID(input.CourseID)
			if err != nil || requested != quiz.CourseID {
				return core.ErrValidation // sınavın dersi sonradan değiştirilemez
			}
		}
		count, err := tx.AttemptCount(id)
		if err != nil {
			return err
		}
		if count > 0 {
			return core.ErrForbidden // geçmiş cevaplar soru kimliklerine bağlı
		}
		topicIDs, err := service.resolveTopics(ctx, input.Questions, quiz.CourseID)
		if err != nil {
			return err
		}
		quiz.Name = strings.TrimSpace(input.Name)
		quiz.Description = strings.TrimSpace(input.Description)
		quiz.OpenAt = input.OpenAt
		quiz.CloseAt = input.CloseAt
		quiz.DurationMinutes = input.DurationMinutes
		quiz.MaxAttempts = input.MaxAttempts
		quiz.Points = TotalPoints(input)
		quiz.State = approvals.StateDraft
		quiz.ReviewedByID = nil
		quiz.ReviewedAt = nil
		if err := quiz.Validate(); err != nil {
			return err
		}
		if err := tx.SaveQuiz(quiz); err != nil {
			return err
		}
		if err := tx.DeleteQuestions(id); err != nil {
			return err
		}
		return writeQuestions(tx, quiz, actor.ID, input.Questions, topicIDs)
	})
	if err != nil {
		return nil, err
	}
	service.invalidate(ctx, classID, courseID)
	return service.DetailFor(ctx, id)
}

func (service *service) Review(ctx context.Context, actor Actor, quizID string, action approvals.Action, note string) (*Detail, error) {
	id, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	// Öğretmen yalnızca onaya gönderebilir; onay/ret yönetici işidir. (Eski
	// /review ucu bunu denetlemiyordu: öğretmen kendi sınavını onaylayabiliyordu.)
	if (action == approvals.ActionApprove || action == approvals.ActionReject) && !actor.isAdmin() {
		return nil, core.ErrForbidden
	}
	var classID, courseID uuid.UUID
	err = service.store.InTx(ctx, func(tx Tx) error {
		quiz, err := tx.GetQuiz(id)
		if err != nil {
			return err
		}
		classID, courseID = quiz.ClassID, quiz.CourseID
		decision, err := approvals.Decide(action, actor.ID, note, service.now(), quiz.State)
		if err != nil {
			return err
		}
		if action == approvals.ActionSubmit || action == approvals.ActionApprove {
			detail, err := tx.LoadDetail(id)
			if err != nil {
				return err
			}
			if issues := Check(detail.Input()); len(issues) > 0 {
				return &ValidationError{Issues: issues}
			}
		}
		if err := approvals.Apply(&quiz.Review, decision); err != nil {
			return err
		}
		if err := tx.SaveQuiz(quiz); err != nil {
			return err
		}
		// Sorular sınavın durumunu izler; öğrenci ile puanlama aynı soru kümesini görür.
		return tx.SetQuestionsState(id, quiz.Review)
	})
	if err != nil {
		return nil, err
	}
	service.invalidate(ctx, classID, courseID)
	return service.DetailFor(ctx, id)
}

func (service *service) Detail(ctx context.Context, quizID string) (*Detail, error) {
	id, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.DetailFor(ctx, id)
}

// DetailFor, sınavı cevap anahtarıyla (personel görünümü) ve eksik listesiyle döner.
func (service *service) DetailFor(ctx context.Context, quizID uuid.UUID) (*Detail, error) {
	detail, err := service.store.LoadDetail(ctx, quizID)
	if err != nil {
		return nil, err
	}
	count, err := service.store.AttemptCount(ctx, quizID)
	if err != nil {
		return nil, err
	}
	detail.AttemptCount = count
	detail.QuestionsEditable = count == 0
	detail.Issues = Check(detail.Input())
	if detail.Issues == nil {
		detail.Issues = []Issue{}
	}
	return detail, nil
}

func (service *service) Results(ctx context.Context, quizID string) (*Summary, error) {
	id, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	// Süresi dolup teslim etmeyen denemeler listelenirken otomatik tamamlanır.
	if _, err := service.attempts.List(ctx, attempts.ListParams{QuizID: &id, Status: attempts.StatusInProgress, Limit: 500}); err != nil {
		return nil, err
	}
	completed, err := service.attempts.List(ctx, attempts.ListParams{QuizID: &id, Status: attempts.StatusCompleted, Limit: 500})
	if err != nil {
		return nil, err
	}
	stats, err := service.store.QuestionStats(ctx, id)
	if err != nil {
		return nil, err
	}
	summary := Summarize(id, completed, stats)
	return &summary, nil
}

func (service *service) resolveTopics(ctx context.Context, items []QuestionInput, courseID uuid.UUID) ([]*uuid.UUID, error) {
	result := make([]*uuid.UUID, len(items))
	for index, item := range items {
		if strings.TrimSpace(item.TopicID) == "" {
			continue
		}
		topicID, err := parseID(item.TopicID)
		if err != nil {
			return nil, core.ErrValidation
		}
		topicCourse, err := service.topics.GetByID(ctx, topicID)
		if err != nil {
			return nil, err
		}
		if topicCourse != courseID {
			return nil, core.ErrValidation
		}
		result[index] = &topicID
	}
	return result, nil
}

func (service *service) invalidate(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) {
	if service.cache != nil {
		_ = service.cache.Clear(ctx)
	}
	if service.invalidator == nil {
		return
	}
	_ = service.invalidator.InvalidateClass(ctx, classID)
	_ = service.invalidator.InvalidateCourse(ctx, courseID)
}

func writeQuestions(tx Tx, quiz *quizes.Quiz, createdBy uuid.UUID, items []QuestionInput, topicIDs []*uuid.UUID) error {
	for index, item := range items {
		question := &questions.Question{
			QuizID:      quiz.ID,
			TopicID:     topicIDs[index],
			Question:    strings.TrimSpace(item.Question),
			Points:      effectivePoints(item.Points),
			Position:    index,
			CreatedByID: createdBy,
		}
		question.State = quiz.State
		if err := tx.CreateQuestion(question); err != nil {
			return err
		}
		for position, entry := range item.Options {
			option := &options.Option{
				QuestionID: question.ID,
				Content:    strings.TrimSpace(entry.Content),
				Answer:     entry.Answer,
				Position:   position,
			}
			if err := tx.CreateOption(option); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
