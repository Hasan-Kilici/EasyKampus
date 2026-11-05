package schools

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type UpdateInput = CreateInput

type StaffInput struct {
	UserID string    `json:"user_id"`
	Role   StaffRole `json:"role"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (*School, error)
	GetByID(ctx context.Context, id string) (*School, error)
	List(ctx context.Context, params ListParams) ([]School, error)
	Update(ctx context.Context, id string, input UpdateInput) (*School, error)
	Delete(ctx context.Context, id string) error
	ListStaff(ctx context.Context, id string) ([]Staff, error)
	AssignStaff(ctx context.Context, id string, input StaffInput) error
	RemoveStaff(ctx context.Context, id string, userID string) error
}

type service struct{ repository Repository }

func NewService(repository Repository) Service { return &service{repository: repository} }

func (service *service) Create(ctx context.Context, input CreateInput) (*School, error) {
	school, err := build(input)
	if err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, school); err != nil {
		return nil, err
	}
	return school, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*School, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, params ListParams) ([]School, error) {
	return service.repository.List(ctx, params)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*School, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	school, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	school.Name = updated.Name
	school.Code = updated.Code
	if err := school.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, school); err != nil {
		return nil, err
	}
	return school, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func (service *service) ListStaff(ctx context.Context, id string) ([]Staff, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListStaff(ctx, parsedID)
}

func (service *service) AssignStaff(ctx context.Context, id string, input StaffInput) error {
	schoolID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	userID, err := parseID(input.UserID)
	if err != nil {
		return core.ErrValidation
	}
	if !input.Role.IsValid() {
		return core.ErrValidation
	}
	return service.repository.AssignStaff(ctx, &Staff{SchoolID: schoolID, UserID: userID, Role: input.Role})
}

func (service *service) RemoveStaff(ctx context.Context, id string, userID string) error {
	parsedSchoolID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	parsedUserID, err := parseID(userID)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.RemoveStaff(ctx, parsedSchoolID, parsedUserID)
}

func build(input CreateInput) (*School, error) {
	school := &School{Name: strings.TrimSpace(input.Name), Code: normalizeCode(input.Code)}
	if err := school.Validate(); err != nil {
		return nil, err
	}
	return school, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
