package notes

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/users"
)

const maxBodyLength = 2000

type CreateInput struct {
	StudentID        string `json:"student_id"`
	ClassID          string `json:"class_id"`
	CourseID         string `json:"course_id"`
	Body             string `json:"body"`
	VisibleToStudent bool   `json:"visible_to_student"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, authorID uuid.UUID, role users.Role, input CreateInput) (*Note, error)
	GetByID(ctx context.Context, id string) (*Note, error)
	List(ctx context.Context, params ListParams) ([]Note, error)
	Update(ctx context.Context, authorID uuid.UUID, role users.Role, id string, input UpdateInput) (*Note, error)
	Delete(ctx context.Context, authorID uuid.UUID, id string) error
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type TeacherAssignmentLookup interface {
	ExistsInScope(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, teacherID uuid.UUID) (bool, error)
}

type service struct {
	repository  Repository
	classes     ClassLookup
	assignments TeacherAssignmentLookup
}

func NewService(repository Repository, classes ClassLookup, assignments TeacherAssignmentLookup) Service {
	return &service{repository: repository, classes: classes, assignments: assignments}
}

func (service *service) Create(ctx context.Context, authorID uuid.UUID, role users.Role, input CreateInput) (*Note, error) {
	note, err := build(input)
	if err != nil {
		return nil, err
	}
	if authorID == uuid.Nil {
		return nil, core.ErrForbidden
	}
	note.AuthorID = authorID
	if err := note.Validate(); err != nil {
		return nil, err
	}
	if err := service.allowed(ctx, authorID, role, note); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*Note, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, params ListParams) ([]Note, error) {
	if params.Limit <= 0 || params.Limit > 200 {
		params.Limit = 50
	}
	return service.repository.List(ctx, params)
}

func (service *service) Update(ctx context.Context, authorID uuid.UUID, role users.Role, id string, input UpdateInput) (*Note, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	note, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	if note.AuthorID != authorID {
		return nil, core.ErrForbidden
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	if err := service.allowed(ctx, authorID, role, updated); err != nil {
		return nil, err
	}
	note.StudentID = updated.StudentID
	note.ClassID = updated.ClassID
	note.CourseID = updated.CourseID
	note.Body = updated.Body
	note.VisibleToStudent = updated.VisibleToStudent
	if err := note.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (service *service) Delete(ctx context.Context, authorID uuid.UUID, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	note, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if note.AuthorID != authorID {
		return core.ErrForbidden
	}
	return service.repository.Delete(ctx, parsedID)
}

func (service *service) allowed(ctx context.Context, authorID uuid.UUID, role users.Role, note *Note) error {
	if note.ClassID == nil {
		return nil
	}
	exists, err := service.classes.Exists(ctx, *note.ClassID)
	if err != nil {
		return err
	}
	if !exists {
		return core.ErrNotFound
	}
	if role == users.RoleSuperAdmin || role == users.RoleAdmin {
		return nil
	}
	if note.CourseID != nil {
		assigned, err := service.assignments.ExistsInScope(ctx, *note.ClassID, *note.CourseID, authorID)
		if err != nil {
			return err
		}
		if !assigned {
			return core.ErrForbidden
		}
	}
	return nil
}

func build(input CreateInput) (*Note, error) {
	studentID, err := parseID(input.StudentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	classID, err := parseOptionalID(input.ClassID)
	if err != nil {
		return nil, err
	}
	courseID, err := parseOptionalID(input.CourseID)
	if err != nil {
		return nil, err
	}
	if courseID != nil && classID == nil {
		return nil, core.ErrValidation
	}
	note := &Note{
		StudentID:        studentID,
		ClassID:          classID,
		CourseID:         courseID,
		Body:             strings.TrimSpace(input.Body),
		VisibleToStudent: input.VisibleToStudent,
	}
	return note, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }

func parseOptionalID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseID(value)
	if err != nil {
		return nil, core.ErrValidation
	}
	return &parsed, nil
}
