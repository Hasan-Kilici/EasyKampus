package terms

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type service struct{ repository Repository }

func NewService(repository Repository) Service { return &service{repository: repository} }

func (service *service) Start(ctx context.Context, input StartInput) (*Term, error) {
	schoolID, err := uuid.Parse(strings.TrimSpace(input.SchoolID))
	if err != nil || strings.TrimSpace(input.Name) == "" || input.StartsAt.IsZero() {
		return nil, core.ErrValidation
	}
	active, err := service.repository.HasActive(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, core.ErrAlreadyExists
	}
	term := &Term{SchoolID: schoolID, Name: strings.TrimSpace(input.Name), StartsAt: input.StartsAt, Status: StatusActive}
	if err := service.repository.Create(ctx, term); err != nil {
		return nil, err
	}
	return term, nil
}

func (service *service) Close(ctx context.Context, id string) (*Term, error) {
	termID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, core.ErrValidation
	}
	term, err := service.repository.GetByID(ctx, termID)
	if err != nil {
		return nil, err
	}
	if term.Status != StatusActive {
		return nil, core.ErrValidation
	}
	now := time.Now().UTC()
	term.Status = StatusClosed
	term.EndsAt = &now
	if err := service.repository.Update(ctx, term); err != nil {
		return nil, err
	}
	return term, nil
}

func (service *service) List(ctx context.Context, schoolID string) ([]Term, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(schoolID))
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.List(ctx, parsedID)
}
