package classes

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	SchoolID string `json:"school_id"`
	CampusID string `json:"campus_id"`
	Name     string `json:"name"`
	Level    string `json:"level"`
	No       string `json:"no"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Class, error)
	GetByID(ctx context.Context, id string) (*Class, error)
	List(ctx context.Context, scope Scope, params ListParams) ([]Class, error)
	Count(ctx context.Context, scope Scope, params ListParams) (int64, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Class, error)
	Delete(ctx context.Context, id string) error
}

type SchoolLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CampusLookup interface {
	GetSchoolID(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
}

type service struct {
	repository Repository
	schools    SchoolLookup
	campuses   CampusLookup
}

func NewService(repository Repository, schools SchoolLookup, campuses CampusLookup) Service {
	return &service{repository: repository, schools: schools, campuses: campuses}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Class, error) {
	class, err := build(input)
	if err != nil {
		return nil, err
	}
	if err := service.verifyOrganization(ctx, class); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, class); err != nil {
		return nil, err
	}
	return class, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Class, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, scope Scope, params ListParams) ([]Class, error) {
	return service.repository.List(ctx, params, scope)
}

func (service *service) Count(ctx context.Context, scope Scope, params ListParams) (int64, error) {
	return service.repository.Count(ctx, params, scope)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Class, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	class, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	if err := service.verifyOrganization(ctx, updated); err != nil {
		return nil, err
	}
	class.SchoolID = updated.SchoolID
	class.CampusID = updated.CampusID
	class.Name = updated.Name
	class.Level = updated.Level
	class.No = updated.No
	if err := class.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, class); err != nil {
		return nil, err
	}
	return class, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func (service *service) verifyOrganization(ctx context.Context, class *Class) error {
	exists, err := service.schools.Exists(ctx, class.SchoolID)
	if err != nil {
		return err
	}
	if !exists {
		return core.ErrNotFound
	}
	campusSchoolID, err := service.campuses.GetSchoolID(ctx, class.CampusID)
	if err != nil {
		return err
	}
	if campusSchoolID != class.SchoolID {
		return core.ErrValidation
	}
	return nil
}

func build(input CreateInput) (*Class, error) {
	schoolID, err := parseID(input.SchoolID)
	if err != nil {
		return nil, core.ErrValidation
	}
	campusID, err := parseID(input.CampusID)
	if err != nil {
		return nil, core.ErrValidation
	}
	class := &Class{
		SchoolID: schoolID,
		CampusID: campusID,
		Name:     strings.TrimSpace(input.Name),
		Level:    strings.TrimSpace(input.Level),
		No:       strings.ToUpper(strings.TrimSpace(input.No)),
	}
	if err := class.Validate(); err != nil {
		return nil, err
	}
	return class, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
