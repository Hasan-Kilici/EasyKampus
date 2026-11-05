package topics

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	CourseID string  `json:"course_id"`
	ParentID string  `json:"parent_id"`
	Name     string  `json:"name"`
	Code     string  `json:"code"`
	Weight   float64 `json:"weight"`
	Active   bool    `json:"active"`
	// Description zorunludur; konunun ne anlattığını açıklar.
	Description string `json:"description"`
	// StudyHours, konunun saat cinsinden süresidir (EFOR).
	StudyHours float64 `json:"study_hours"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Topic, error)
	GetByID(ctx context.Context, id string) (*Topic, error)
	ListByCourse(ctx context.Context, courseID string) ([]Topic, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Topic, error)
	Delete(ctx context.Context, id string) error
}

type CourseLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type service struct {
	repository Repository
	courses    CourseLookup
}

func NewService(repository Repository, courses CourseLookup) Service {
	return &service{repository: repository, courses: courses}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*Topic, error) {
	topic, err := build(input)
	if err != nil {
		return nil, err
	}
	exists, err := service.courses.Exists(ctx, topic.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	if topic.ParentID != nil {
		parent, err := service.repository.GetByID(ctx, *topic.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.CourseID != topic.CourseID {
			return nil, core.ErrValidation
		}
	}
	if err := service.repository.Create(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Topic, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) ListByCourse(ctx context.Context, courseID string) ([]Topic, error) {
	parsedID, err := parseID(courseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListByCourse(ctx, parsedID, false)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*Topic, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	topic, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	if updated.ParentID != nil {
		if *updated.ParentID == topic.ID {
			return nil, core.ErrValidation
		}
		parent, err := service.repository.GetByID(ctx, *updated.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.CourseID != updated.CourseID {
			return nil, core.ErrValidation
		}
	}
	topic.CourseID = updated.CourseID
	topic.ParentID = updated.ParentID
	topic.Name = updated.Name
	topic.Code = updated.Code
	topic.Weight = updated.Weight
	topic.Active = updated.Active
	topic.Description = updated.Description
	topic.StudyHours = updated.StudyHours
	if err := topic.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func build(input CreateInput) (*Topic, error) {
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	topic := &Topic{
		CourseID: courseID,
		Name:     strings.TrimSpace(input.Name),
		Code:     strings.ToLower(strings.TrimSpace(input.Code)),
		Weight:   input.Weight,
		Active:   input.Active,

		Description: strings.TrimSpace(input.Description),
		StudyHours:  input.StudyHours,
	}
	if input.ParentID != "" {
		parentID, err := parseID(input.ParentID)
		if err != nil {
			return nil, core.ErrValidation
		}
		topic.ParentID = &parentID
	}
	if err := topic.Validate(); err != nil {
		return nil, err
	}
	return topic, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
