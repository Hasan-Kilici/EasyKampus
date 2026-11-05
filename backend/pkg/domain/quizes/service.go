package quizes

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/users"
)

type CreateInput struct {
	ClassID         string     `json:"class_id"`
	CourseID        string     `json:"course_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	OpenAt          *time.Time `json:"open_at"`
	CloseAt         *time.Time `json:"close_at"`
	DurationMinutes int        `json:"duration_minutes"`
	Points          float64    `json:"points"`
	MaxAttempts     int        `json:"max_attempts"`
	CreatedByID     string     `json:"created_by_id"`
	CreatedByRole   users.Role `json:"created_by_role"`
}

type UpdateInput struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	OpenAt          *time.Time `json:"open_at"`
	CloseAt         *time.Time `json:"close_at"`
	DurationMinutes int        `json:"duration_minutes"`
	Points          float64    `json:"points"`
	MaxAttempts     int        `json:"max_attempts"`
}

type ReviewInput struct {
	Action     approvals.Action `json:"action"`
	ReviewerID string           `json:"reviewer_id"`
	Note       string           `json:"note"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Quiz, error)
	GetByID(ctx context.Context, id string) (*Quiz, error)
	ListByClass(ctx context.Context, classID string, onlyPublished bool) ([]Quiz, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Quiz, error)
	Delete(ctx context.Context, id string) error
	Review(ctx context.Context, id string, input ReviewInput) (*Quiz, error)
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CourseOfferLookup interface {
	ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error)
}

type service struct {
	repository  Repository
	classes     ClassLookup
	courses     CourseOfferLookup
	cache       core.Cache
	invalidator performance.Invalidator
	flight      singleflight.Group
}

func NewService(repository Repository, classes ClassLookup, courses CourseOfferLookup, invalidator performance.Invalidator, caches ...core.Cache) Service {
	var cache core.Cache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &service{repository: repository, classes: classes, courses: courses, cache: cache, invalidator: invalidator}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Quiz, error) {
	classID, err := parseID(input.ClassID)
	if err != nil {
		return nil, core.ErrValidation
	}
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	createdByID, err := parseID(input.CreatedByID)
	if err != nil {
		return nil, core.ErrValidation
	}
	classExists, err := service.classes.Exists(ctx, classID)
	if err != nil {
		return nil, err
	}
	if !classExists {
		return nil, core.ErrNotFound
	}
	offered, err := service.courses.ClassCourseExists(ctx, classID, courseID)
	if err != nil {
		return nil, err
	}
	if !offered {
		return nil, core.ErrValidation
	}
	quiz := &Quiz{
		ClassID:         classID,
		CourseID:        courseID,
		Name:            strings.TrimSpace(input.Name),
		Description:     strings.TrimSpace(input.Description),
		OpenAt:          input.OpenAt,
		CloseAt:         input.CloseAt,
		DurationMinutes: input.DurationMinutes,
		Points:          input.Points,
		MaxAttempts:     input.MaxAttempts,
		CreatedByID:     createdByID,
	}
	quiz.State = initialState(input.CreatedByRole)
	if err := quiz.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, quiz); err != nil {
		return nil, err
	}
	return quiz, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Quiz, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	key := "quiz:" + parsedID.String()
	if value, ok := service.cached(ctx, key); ok {
		if quiz, ok := value.(*Quiz); ok {
			return quiz, nil
		}
	}
	value, err, _ := service.flight.Do(key, func() (any, error) {
		if cached, ok := service.cached(ctx, key); ok {
			return cached, nil
		}
		quiz, loadErr := service.repository.GetByID(ctx, parsedID)
		if loadErr == nil {
			_ = service.store(ctx, key, quiz)
		}
		return quiz, loadErr
	})
	if err != nil {
		return nil, err
	}
	quiz, ok := value.(*Quiz)
	if !ok {
		return nil, core.ErrNotFound
	}
	return quiz, nil
}

func (service *service) ListByClass(ctx context.Context, classID string, onlyPublished bool) ([]Quiz, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	key := "quizzes:class:" + parsedClassID.String() + ":published:" + strconv.FormatBool(onlyPublished)
	if value, ok := service.cached(ctx, key); ok {
		if result, ok := value.([]Quiz); ok {
			return result, nil
		}
	}
	value, err, _ := service.flight.Do(key, func() (any, error) {
		if cached, ok := service.cached(ctx, key); ok {
			return cached, nil
		}
		result, loadErr := service.repository.ListByClass(ctx, ListParams{ClassID: parsedClassID, OnlyPublished: onlyPublished, Limit: 200})
		if loadErr == nil {
			_ = service.store(ctx, key, result)
		}
		return result, loadErr
	})
	if err != nil {
		return nil, err
	}
	result, ok := value.([]Quiz)
	if !ok {
		return nil, core.ErrNotFound
	}
	return result, nil
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Quiz, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	quiz, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	quiz.Name = strings.TrimSpace(input.Name)
	quiz.Description = strings.TrimSpace(input.Description)
	quiz.OpenAt = input.OpenAt
	quiz.CloseAt = input.CloseAt
	quiz.DurationMinutes = input.DurationMinutes
	quiz.Points = input.Points
	quiz.MaxAttempts = input.MaxAttempts
	if err := quiz.Validate(); err != nil {
		return nil, err
	}
	resetToDraft(quiz)
	if err := service.repository.Update(ctx, quiz); err != nil {
		return nil, err
	}
	service.invalidate(ctx, quiz.ClassID, quiz.CourseID)
	return quiz, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	quiz, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	service.invalidate(ctx, quiz.ClassID, quiz.CourseID)
	return nil
}

func (service *service) Review(ctx context.Context, id string, input ReviewInput) (*Quiz, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	reviewerID, err := parseID(input.ReviewerID)
	if err != nil {
		return nil, core.ErrValidation
	}
	quiz, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	decision, err := approvals.Decide(input.Action, reviewerID, input.Note, time.Now().UTC(), quiz.State)
	if err != nil {
		return nil, err
	}
	if err := approvals.Apply(&quiz.Review, decision); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, quiz); err != nil {
		return nil, err
	}
	service.invalidate(ctx, quiz.ClassID, quiz.CourseID)
	return quiz, nil
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

func (service *service) cached(ctx context.Context, key string) (any, bool) {
	if service.cache == nil {
		return nil, false
	}
	return service.cache.Get(ctx, key)
}

func (service *service) store(ctx context.Context, key string, value any) error {
	if service.cache == nil {
		return nil
	}
	return service.cache.Set(ctx, key, value)
}

func initialState(role users.Role) approvals.State {
	switch role {
	case users.RoleAdmin, users.RoleSuperAdmin:
		return approvals.StateApproved
	default:
		return approvals.StateDraft
	}
}

// Onaylanmış içerik düzenlendiğinde yeniden onaya döner, aksi halde öğrenci
// onaylanmamış değişikliği görmeye devam eder.
func resetToDraft(quiz *Quiz) {
	quiz.State = approvals.StateDraft
	quiz.ReviewedByID = nil
	quiz.ReviewedAt = nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
