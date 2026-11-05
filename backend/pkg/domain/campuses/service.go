package campuses

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	SchoolID string `json:"school_id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Campus, error)
	GetByID(ctx context.Context, id string) (*Campus, error)
	List(ctx context.Context, params ListParams) ([]Campus, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Campus, error)
	Delete(ctx context.Context, id string) error
}

type SchoolLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type service struct {
	repository Repository
	schools    SchoolLookup
}

func NewService(repository Repository, schools SchoolLookup) Service {
	return &service{repository: repository, schools: schools}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Campus, error) {
	campus, err := build(input)
	if err != nil {
		return nil, err
	}
	exists, err := service.schools.Exists(ctx, campus.SchoolID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	if err := service.repository.Create(ctx, campus); err != nil {
		return nil, err
	}
	return campus, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Campus, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, params ListParams) ([]Campus, error) {
	return service.repository.List(ctx, params)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Campus, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	campus, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	exists, err := service.schools.Exists(ctx, updated.SchoolID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	campus.SchoolID = updated.SchoolID
	campus.Name = updated.Name
	campus.Code = updated.Code
	if err := campus.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, campus); err != nil {
		return nil, err
	}
	return campus, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func build(input CreateInput) (*Campus, error) {
	schoolID, err := parseID(input.SchoolID)
	if err != nil {
		return nil, core.ErrValidation
	}
	campus := &Campus{
		SchoolID: schoolID,
		Name:     strings.TrimSpace(input.Name),
		Code:     strings.ToUpper(strings.TrimSpace(input.Code)),
	}
	if err := campus.Validate(); err != nil {
		return nil, err
	}
	return campus, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
