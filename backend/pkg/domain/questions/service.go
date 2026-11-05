package questions

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/users"
)

type CreateInput struct {
	QuizID        string     `json:"quiz_id"`
	TopicID       string     `json:"topic_id"`
	Question      string     `json:"question"`
	Points        float64    `json:"points"`
	Position      int        `json:"position"`
	CreatedByID   string     `json:"created_by_id"`
	CreatedByRole users.Role `json:"created_by_role"`
}

type UpdateInput struct {
	TopicID  string  `json:"topic_id"`
	Question string  `json:"question"`
	Points   float64 `json:"points"`
	Position int     `json:"position"`
}

type ReviewInput struct {
	Action     approvals.Action `json:"action"`
	ReviewerID string           `json:"reviewer_id"`
	Note       string           `json:"note"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Question, error)
	GetByID(ctx context.Context, id string) (*Question, error)
	ListByQuiz(ctx context.Context, quizID string, onlyPublished bool) ([]Question, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Question, error)
	Delete(ctx context.Context, id string) error
	Review(ctx context.Context, id string, input ReviewInput) (*Question, error)
}

type QuizLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*QuizScope, error)
}

type QuizScope struct {
	ID       uuid.UUID
	ClassID  uuid.UUID
	CourseID uuid.UUID
}

type TopicLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (courseID uuid.UUID, err error)
}

type service struct {
	repository  Repository
	quizzes     QuizLookup
	topicSource TopicLookup
	invalidator performance.Invalidator
}

func NewService(repository Repository, quizzes QuizLookup, topicSource TopicLookup, invalidator performance.Invalidator) Service {
	return &service{repository: repository, quizzes: quizzes, topicSource: topicSource, invalidator: invalidator}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Question, error) {
	quizID, err := parseID(input.QuizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	createdByID, err := parseID(input.CreatedByID)
	if err != nil {
		return nil, core.ErrValidation
	}
	quiz, err := service.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	topicID, err := service.resolveTopic(ctx, input.TopicID, quiz.CourseID)
	if err != nil {
		return nil, err
	}
	question := &Question{
		QuizID:      quizID,
		TopicID:     topicID,
		Question:    strings.TrimSpace(input.Question),
		Points:      input.Points,
		Position:    input.Position,
		CreatedByID: createdByID,
	}
	question.State = initialState(input.CreatedByRole)
	if question.Points == 0 {
		question.Points = 1
	}
	if err := question.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Question, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) ListByQuiz(ctx context.Context, quizID string, onlyPublished bool) ([]Question, error) {
	parsedQuizID, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListByQuiz(ctx, ListParams{QuizID: parsedQuizID, OnlyPublished: onlyPublished, Limit: 200})
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Question, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	question, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	quiz, err := service.quizzes.GetByID(ctx, question.QuizID)
	if err != nil {
		return nil, err
	}
	topicID, err := service.resolveTopic(ctx, input.TopicID, quiz.CourseID)
	if err != nil {
		return nil, err
	}
	question.TopicID = topicID
	question.Question = strings.TrimSpace(input.Question)
	question.Points = input.Points
	question.Position = input.Position
	if err := question.Validate(); err != nil {
		return nil, err
	}
	resetToDraft(question)
	if err := service.repository.Update(ctx, question); err != nil {
		return nil, err
	}
	service.invalidate(ctx, quiz.CourseID)
	return question, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	question, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	quiz, err := service.quizzes.GetByID(ctx, question.QuizID)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	service.invalidate(ctx, quiz.CourseID)
	return nil
}

func (service *service) Review(ctx context.Context, id string, input ReviewInput) (*Question, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	reviewerID, err := parseID(input.ReviewerID)
	if err != nil {
		return nil, core.ErrValidation
	}
	question, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	quiz, err := service.quizzes.GetByID(ctx, question.QuizID)
	if err != nil {
		return nil, err
	}
	decision, err := approvals.Decide(input.Action, reviewerID, input.Note, time.Now().UTC(), question.State)
	if err != nil {
		return nil, err
	}
	if err := approvals.Apply(&question.Review, decision); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, question); err != nil {
		return nil, err
	}
	service.invalidate(ctx, quiz.CourseID)
	return question, nil
}

func (service *service) resolveTopic(ctx context.Context, value string, courseID uuid.UUID) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	topicID, err := parseID(value)
	if err != nil {
		return nil, core.ErrValidation
	}
	topicCourseID, err := service.topicSource.GetByID(ctx, topicID)
	if err != nil {
		return nil, err
	}
	if topicCourseID != courseID {
		return nil, core.ErrValidation
	}
	return &topicID, nil
}

func (service *service) invalidate(ctx context.Context, courseID uuid.UUID) {
	if service.invalidator == nil {
		return
	}
	_ = service.invalidator.InvalidateCourse(ctx, courseID)
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
func resetToDraft(question *Question) {
	question.State = approvals.StateDraft
	question.ReviewedByID = nil
	question.ReviewedAt = nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
