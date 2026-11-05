package schedules

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

const minutesInDay = 24 * 60

type CreateInput struct {
	CourseID    string `json:"course_id"`
	TeacherID   string `json:"teacher_id"`
	Term        string `json:"term"`
	DayOfWeek   int    `json:"day_of_week"`
	StartMinute int    `json:"start_minute"`
	EndMinute   int    `json:"end_minute"`
}

type UpdateInput = CreateInput

type Service interface {
	Create(ctx context.Context, classID string, input CreateInput) (*ScheduleEntry, error)
	GetByID(ctx context.Context, id string) (*ScheduleEntry, error)
	List(ctx context.Context, params ListParams) ([]ScheduleEntry, error)
	Update(ctx context.Context, id string, input UpdateInput) (*ScheduleEntry, error)
	Delete(ctx context.Context, id string) error
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CourseOfferLookup interface {
	ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error)
}

type TeacherAssignmentLookup interface {
	Exists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, teacherID uuid.UUID) (bool, error)
}

type service struct {
	repository  Repository
	classes     ClassLookup
	courses     CourseOfferLookup
	assignments TeacherAssignmentLookup
}

func NewService(repository Repository, classes ClassLookup, courses CourseOfferLookup, assignments TeacherAssignmentLookup) Service {
	return &service{repository: repository, classes: classes, courses: courses, assignments: assignments}
}

func (service *service) Create(ctx context.Context, classID string, input CreateInput) (*ScheduleEntry, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	entry, err := build(input)
	if err != nil {
		return nil, err
	}
	entry.ClassID = parsedClassID
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	exists, err := service.classes.Exists(ctx, parsedClassID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, core.ErrNotFound
	}
	offered, err := service.courses.ClassCourseExists(ctx, parsedClassID, entry.CourseID)
	if err != nil {
		return nil, err
	}
	if !offered {
		return nil, core.ErrValidation
	}
	assigned, err := service.assignments.Exists(ctx, parsedClassID, entry.CourseID, entry.TeacherID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, core.ErrValidation
	}
	if err := service.repository.Create(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*ScheduleEntry, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) List(ctx context.Context, params ListParams) ([]ScheduleEntry, error) {
	return service.repository.List(ctx, params, 300)
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*ScheduleEntry, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	entry, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	updated, err := build(input)
	if err != nil {
		return nil, err
	}
	// Ders veya öğretmen değiştiğinde sınıf-içi atamalar yeniden doğrulanır.
	if updated.CourseID != entry.CourseID || updated.TeacherID != entry.TeacherID {
		offered, err := service.courses.ClassCourseExists(ctx, entry.ClassID, updated.CourseID)
		if err != nil {
			return nil, err
		}
		if !offered {
			return nil, core.ErrValidation
		}
		assigned, err := service.assignments.Exists(ctx, entry.ClassID, updated.CourseID, updated.TeacherID)
		if err != nil {
			return nil, err
		}
		if !assigned {
			return nil, core.ErrValidation
		}
	}
	entry.CourseID = updated.CourseID
	entry.TeacherID = updated.TeacherID
	entry.Term = updated.Term
	entry.DayOfWeek = updated.DayOfWeek
	entry.StartMinute = updated.StartMinute
	entry.EndMinute = updated.EndMinute
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func build(input CreateInput) (*ScheduleEntry, error) {
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	teacherID, err := parseID(input.TeacherID)
	if err != nil {
		return nil, core.ErrValidation
	}
	entry := &ScheduleEntry{
		CourseID:    courseID,
		TeacherID:   teacherID,
		Term:        strings.TrimSpace(input.Term),
		DayOfWeek:   input.DayOfWeek,
		StartMinute: input.StartMinute,
		EndMinute:   input.EndMinute,
		Active:      true,
	}
	return entry, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
