package lessons

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/users"
)

type CreateInput struct {
	CourseID        string     `json:"course_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	MarkdownContent string     `json:"markdown_content"`
	Position        int        `json:"position"`
	DurationMinutes int        `json:"duration_minutes"`
	CreatedByID     string     `json:"created_by_id"`
	CreatedByRole   users.Role `json:"created_by_role"`
}

type UpdateInput struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	MarkdownContent string `json:"markdown_content"`
	Position        int    `json:"position"`
	DurationMinutes int    `json:"duration_minutes"`
}

type ReviewInput struct {
	Action     approvals.Action `json:"action"`
	ReviewerID string           `json:"reviewer_id"`
	Note       string           `json:"note"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Lesson, error)
	GetByID(ctx context.Context, id string) (*Lesson, error)
	ListByCourse(ctx context.Context, courseID string, onlyPublished bool) ([]Lesson, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Lesson, error)
	Delete(ctx context.Context, id string) error
	Review(ctx context.Context, id string, input ReviewInput) (*Lesson, error)
	View(ctx context.Context, studentID string, lessonID string) (*Lesson, error)
	Complete(ctx context.Context, studentID string, lessonID string) (*LessonCompletion, error)
	ListCompletions(ctx context.Context, studentID string, courseID string) ([]LessonCompletion, error)
}

type CourseLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type ActivityRecorder interface {
	Record(ctx context.Context, studentID uuid.UUID, events []activities.EventInput) error
}

type service struct {
	repository  Repository
	courses     CourseLookup
	cache       core.Cache
	invalidator performance.Invalidator
	activity    ActivityRecorder
	flight      singleflight.Group
}

func NewService(repository Repository, courses CourseLookup, invalidator performance.Invalidator, caches ...core.Cache) Service {
	var cache core.Cache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &service{repository: repository, courses: courses, cache: cache, invalidator: invalidator}
}

func NewCompletionService(repository Repository, courses CourseLookup, invalidator performance.Invalidator, activity ActivityRecorder, caches ...core.Cache) Service {
	var cache core.Cache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &service{repository: repository, courses: courses, cache: cache, invalidator: invalidator, activity: activity}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Lesson, error) {
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	createdByID, err := parseID(input.CreatedByID)
	if err != nil {
		return nil, core.ErrValidation
	}
	exists, err := service.courses.Exists(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	lesson := &Lesson{
		CourseID:        courseID,
		Title:           strings.TrimSpace(input.Title),
		Description:     strings.TrimSpace(input.Description),
		MarkdownContent: input.MarkdownContent,
		Position:        input.Position,
		DurationMinutes: input.DurationMinutes,
		CreatedByID:     createdByID,
	}
	lesson.State = initialState(input.CreatedByRole)
	if err := lesson.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, lesson); err != nil {
		return nil, err
	}
	return lesson, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Lesson, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	key := "lesson:" + parsedID.String()
	if value, ok := service.cached(ctx, key); ok {
		if lesson, ok := value.(*Lesson); ok {
			return lesson, nil
		}
	}
	value, err, _ := service.flight.Do(key, func() (any, error) {
		if cached, ok := service.cached(ctx, key); ok {
			return cached, nil
		}
		lesson, loadErr := service.repository.GetByID(ctx, parsedID)
		if loadErr == nil {
			_ = service.store(ctx, key, lesson)
		}
		return lesson, loadErr
	})
	if err != nil {
		return nil, err
	}
	lesson, ok := value.(*Lesson)
	if !ok {
		return nil, core.ErrNotFound
	}
	return lesson, nil
}

func (service *service) ListByCourse(ctx context.Context, courseID string, onlyPublished bool) ([]Lesson, error) {
	parsedCourseID, err := parseID(courseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	key := "lessons:course:" + parsedCourseID.String() + ":published:" + strconv.FormatBool(onlyPublished)
	if value, ok := service.cached(ctx, key); ok {
		if result, ok := value.([]Lesson); ok {
			return result, nil
		}
	}
	value, err, _ := service.flight.Do(key, func() (any, error) {
		if cached, ok := service.cached(ctx, key); ok {
			return cached, nil
		}
		result, loadErr := service.repository.ListByCourse(ctx, ListParams{CourseID: parsedCourseID, OnlyPublished: onlyPublished, Limit: 200})
		if loadErr == nil {
			_ = service.store(ctx, key, result)
		}
		return result, loadErr
	})
	if err != nil {
		return nil, err
	}
	result, ok := value.([]Lesson)
	if !ok {
		return nil, core.ErrNotFound
	}
	return result, nil
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Lesson, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	lesson, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	lesson.Title = strings.TrimSpace(input.Title)
	lesson.Description = strings.TrimSpace(input.Description)
	lesson.MarkdownContent = input.MarkdownContent
	lesson.Position = input.Position
	lesson.DurationMinutes = input.DurationMinutes
	if err := lesson.Validate(); err != nil {
		return nil, err
	}
	resetToDraft(lesson)
	if err := service.repository.Update(ctx, lesson); err != nil {
		return nil, err
	}
	service.invalidate(ctx, lesson.CourseID)
	return lesson, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	lesson, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	if service.cache != nil {
		_ = service.cache.Clear(ctx)
	}
	service.invalidate(ctx, lesson.CourseID)
	return nil
}

func (service *service) Review(ctx context.Context, id string, input ReviewInput) (*Lesson, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	reviewerID, err := parseID(input.ReviewerID)
	if err != nil {
		return nil, core.ErrValidation
	}
	lesson, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	decision, err := approvals.Decide(input.Action, reviewerID, input.Note, time.Now().UTC(), lesson.State)
	if err != nil {
		return nil, err
	}
	if err := approvals.Apply(&lesson.Review, decision); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, lesson); err != nil {
		return nil, err
	}
	if service.cache != nil {
		_ = service.cache.Clear(ctx)
	}
	service.invalidate(ctx, lesson.CourseID)
	return lesson, nil
}

func (service *service) View(ctx context.Context, studentID string, lessonID string) (*Lesson, error) {
	parsedStudent, err := parseID(studentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	lesson, err := service.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	service.record(ctx, parsedStudent, activities.EventInput{Type: activities.EventLessonView, CourseID: lesson.CourseID.String(), LessonID: lesson.ID.String()})
	return lesson, nil
}

func (service *service) Complete(ctx context.Context, studentID string, lessonID string) (*LessonCompletion, error) {
	parsedStudent, err := parseID(studentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	lesson, err := service.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson.State != approvals.StateApproved {
		return nil, core.ErrForbidden
	}
	completion := &LessonCompletion{
		StudentID:   parsedStudent,
		LessonID:    lesson.ID,
		CourseID:    lesson.CourseID,
		CompletedAt: time.Now().UTC(),
	}
	if err := service.repository.UpsertCompletions(ctx, []LessonCompletion{*completion}); err != nil {
		return nil, err
	}
	service.record(ctx, parsedStudent, activities.EventInput{Type: activities.EventLessonCompletion, CourseID: lesson.CourseID.String(), LessonID: lesson.ID.String()})
	return completion, nil
}

func (service *service) ListCompletions(ctx context.Context, studentID string, courseID string) ([]LessonCompletion, error) {
	parsedStudent, err := parseID(studentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	var parsedCourse *uuid.UUID
	if courseID != "" {
		parsed, err := parseID(courseID)
		if err != nil {
			return nil, core.ErrValidation
		}
		parsedCourse = &parsed
	}
	ids, err := service.repository.ListCompletions(ctx, parsedStudent, parsedCourse)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (service *service) record(ctx context.Context, studentID uuid.UUID, event activities.EventInput) {
	if service.activity == nil {
		return
	}
	_ = service.activity.Record(ctx, studentID, []activities.EventInput{event})
}

func (service *service) invalidate(ctx context.Context, courseID uuid.UUID) {
	if service.cache != nil {
		_ = service.cache.Clear(ctx)
	}
	if service.invalidator == nil {
		return
	}
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
func resetToDraft(lesson *Lesson) {
	lesson.State = approvals.StateDraft
	lesson.ReviewedByID = nil
	lesson.ReviewedAt = nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
