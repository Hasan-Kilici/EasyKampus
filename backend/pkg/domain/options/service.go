package options

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
)

type CreateInput struct {
	QuestionID string `json:"question_id"`
	Content    string `json:"content"`
	Answer     bool   `json:"answer"`
	Position   int    `json:"position"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Option, error)
	GetByID(ctx context.Context, id string) (*Option, error)
	GetByIDInQuestion(ctx context.Context, id string, questionID string) (*Option, error)
	ListByQuestion(ctx context.Context, questionID string, page int, limit int) ([]Option, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Option, error)
	Delete(ctx context.Context, id string) error
}

type ScopeResolver interface {
	CourseIDForQuestion(ctx context.Context, questionID uuid.UUID) (uuid.UUID, error)
}

type service struct {
	repository  Repository
	scopes      ScopeResolver
	invalidator performance.Invalidator
}

func NewService(repository Repository, scopes ScopeResolver, invalidator performance.Invalidator) Service {
	return &service{repository: repository, scopes: scopes, invalidator: invalidator}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Option, error) {
	option, err := build(input)
	if err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, option); err != nil {
		return nil, err
	}
	service.invalidate(ctx, option.QuestionID)
	return option, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Option, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) GetByIDInQuestion(ctx context.Context, id string, questionID string) (*Option, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	parsedQuestionID, err := parseID(questionID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByIDInQuestion(ctx, parsedID, parsedQuestionID)
}

func (service *service) ListByQuestion(ctx context.Context, questionID string, page int, limit int) ([]Option, error) {
	parsedID, err := parseID(questionID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListByQuestion(ctx, parsedID, page, limit)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Option, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	option, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	option.QuestionID = updated.QuestionID
	option.Content = updated.Content
	option.Answer = updated.Answer
	option.Position = updated.Position
	if err := service.repository.Update(ctx, option); err != nil {
		return nil, err
	}
	service.invalidate(ctx, option.QuestionID)
	return option, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	option, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	service.invalidate(ctx, option.QuestionID)
	return nil
}

func (service *service) invalidate(ctx context.Context, questionID uuid.UUID) {
	if service.invalidator == nil || service.scopes == nil {
		return
	}
	courseID, err := service.scopes.CourseIDForQuestion(ctx, questionID)
	if err != nil {
		return
	}
	_ = service.invalidator.InvalidateCourse(ctx, courseID)
}

func build(input CreateInput) (*Option, error) {
	questionID, err := parseID(input.QuestionID)
	if err != nil {
		return nil, core.ErrValidation
	}
	option := &Option{
		QuestionID: questionID,
		Content:    strings.TrimSpace(input.Content),
		Answer:     input.Answer,
		Position:   input.Position,
	}
	if err := option.Validate(); err != nil {
		return nil, err
	}
	return option, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
