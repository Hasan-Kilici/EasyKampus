package courses

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	SchoolID    string `json:"school_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	SubjectArea string `json:"subject_area"`
	Description string `json:"description"`
	Active      bool   `json:"active"`

	Department         string  `json:"department"`
	DepartmentKeywords string  `json:"department_keywords"`
	HomeworkHours      float64 `json:"homework_hours"`
	ProjectHours       float64 `json:"project_hours"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Course, error)
	GetByID(ctx context.Context, id string) (*Course, error)
	List(ctx context.Context, params ListParams) ([]Course, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Course, error)
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

func (service *service) Create(ctx context.Context, input CreateInput) (*Course, error) {
	course, err := build(input)
	if err != nil {
		return nil, err
	}
	exists, err := service.schools.Exists(ctx, course.SchoolID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	if err := service.repository.Create(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Course, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, params ListParams) ([]Course, error) {
	return service.repository.List(ctx, params)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Course, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	course, err := service.repository.GetByID(ctx, parsedID)
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
	course.SchoolID = updated.SchoolID
	course.Name = updated.Name
	course.Code = updated.Code
	course.SubjectArea = updated.SubjectArea
	course.Description = updated.Description
	course.Active = updated.Active
	course.Department = updated.Department
	course.DepartmentKeywords = updated.DepartmentKeywords
	course.HomeworkHours = updated.HomeworkHours
	course.ProjectHours = updated.ProjectHours
	if err := course.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func build(input CreateInput) (*Course, error) {
	schoolID, err := parseID(input.SchoolID)
	if err != nil {
		return nil, core.ErrValidation
	}
	course := &Course{
		SchoolID:    schoolID,
		Name:        strings.TrimSpace(input.Name),
		Code:        strings.ToUpper(strings.TrimSpace(input.Code)),
		SubjectArea: strings.TrimSpace(input.SubjectArea),
		Description: strings.TrimSpace(input.Description),
		Active:      input.Active,

		Department:         strings.TrimSpace(input.Department),
		DepartmentKeywords: strings.TrimSpace(input.DepartmentKeywords),
		HomeworkHours:      input.HomeworkHours,
		ProjectHours:       input.ProjectHours,
	}
	if err := course.Validate(); err != nil {
		return nil, err
	}
	return course, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
