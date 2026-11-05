package attendance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/users"
)

const dateLayout = "2006-01-02"

const maxBatchSize = 200

type RecordInput struct {
	StudentID string `json:"student_id"`
	ClassID   string `json:"class_id"`
	CourseID  string `json:"course_id"`
	Date      string `json:"date"`
	Status    Status `json:"status"`
	Note      string `json:"note"`
}

type Service interface {
	Record(ctx context.Context, actorID uuid.UUID, role users.Role, records []RecordInput) ([]Attendance, error)
	Update(ctx context.Context, actorID uuid.UUID, role users.Role, id string, input RecordInput) (*Attendance, error)
	Delete(ctx context.Context, actorID uuid.UUID, role users.Role, id string) error
	List(ctx context.Context, params ListParams) ([]Attendance, error)
	Summarize(ctx context.Context, params ListParams) (PeriodSummary, error)
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type TeacherAssignmentLookup interface {
	Exists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, teacherID uuid.UUID) (bool, error)
}

type EnrollmentLookup interface {
	Exists(ctx context.Context, classID uuid.UUID, studentID uuid.UUID, courseID *uuid.UUID) (bool, error)
}

type service struct {
	repository  Repository
	classes     ClassLookup
	assignments TeacherAssignmentLookup
	enrollments EnrollmentLookup
	invalidator performance.Invalidator
}

func NewService(repository Repository, classes ClassLookup, assignments TeacherAssignmentLookup, enrollments EnrollmentLookup, invalidator performance.Invalidator) Service {
	return &service{repository: repository, classes: classes, assignments: assignments, enrollments: enrollments, invalidator: invalidator}
}

func (service *service) Record(ctx context.Context, actorID uuid.UUID, role users.Role, records []RecordInput) ([]Attendance, error) {
	if len(records) == 0 || len(records) > maxBatchSize {
		return nil, core.ErrValidation
	}
	entities := make([]Attendance, 0, len(records))
	for _, input := range records {
		entity, err := service.build(ctx, actorID, role, input)
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}
	if err := service.repository.Upsert(ctx, entities); err != nil {
		return nil, err
	}
	service.invalidate(ctx, entities)
	return entities, nil
}

func (service *service) Update(ctx context.Context, actorID uuid.UUID, role users.Role, id string, input RecordInput) (*Attendance, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	record, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	entity, err := service.build(ctx, actorID, role, input)
	if err != nil {
		return nil, err
	}
	record.StudentID = entity.StudentID
	record.ClassID = entity.ClassID
	record.CourseID = entity.CourseID
	record.Date = entity.Date
	record.Status = entity.Status
	record.Note = entity.Note
	record.TeacherID = actorID
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, record); err != nil {
		return nil, err
	}
	service.invalidate(ctx, []Attendance{*record})
	return record, nil
}

func (service *service) Delete(ctx context.Context, actorID uuid.UUID, role users.Role, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	record, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if _, err := service.authorize(ctx, actorID, role, record.ClassID, record.CourseID); err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	service.invalidate(ctx, []Attendance{*record})
	return nil
}

func (service *service) List(ctx context.Context, params ListParams) ([]Attendance, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	if params.Limit <= 0 || params.Limit > 1000 {
		params.Limit = 200
	}
	return service.repository.List(ctx, params)
}

func (service *service) Summarize(ctx context.Context, params ListParams) (PeriodSummary, error) {
	if err := validateParams(params); err != nil {
		return PeriodSummary{}, err
	}
	return service.repository.Summarize(ctx, params)
}

func validateParams(params ListParams) error {
	if params.From != "" {
		if _, err := time.Parse(dateLayout, params.From); err != nil {
			return core.ErrValidation
		}
	}
	if params.To != "" {
		if _, err := time.Parse(dateLayout, params.To); err != nil {
			return core.ErrValidation
		}
	}
	if params.From != "" && params.To != "" && params.From > params.To {
		return core.ErrValidation
	}
	if params.Status != "" && !params.Status.IsValid() {
		return core.ErrValidation
	}
	return nil
}

func (service *service) build(ctx context.Context, actorID uuid.UUID, role users.Role, input RecordInput) (*Attendance, error) {
	studentID, err := parseID(input.StudentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	classID, err := parseID(input.ClassID)
	if err != nil {
		return nil, core.ErrValidation
	}
	courseID, err := parseID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	date, err := time.Parse(dateLayout, strings.TrimSpace(input.Date))
	if err != nil {
		fmt.Println("Time format GG")
		return nil, core.ErrValidation
	}
	record := &Attendance{
		StudentID: studentID,
		ClassID:   classID,
		CourseID:  courseID,
		Date:      date.Format(dateLayout),
		TeacherID: actorID,
		Status:    input.Status,
		Note:      strings.TrimSpace(input.Note),
	}
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if _, err := service.authorize(ctx, actorID, role, classID, courseID); err != nil {
		return nil, err
	}
	enrolled, err := service.enrollments.Exists(ctx, classID, studentID, &courseID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, core.ErrValidation
	}
	return record, nil
}

func (service *service) authorize(ctx context.Context, actorID uuid.UUID, role users.Role, classID uuid.UUID, courseID uuid.UUID) (uuid.UUID, error) {
	if role == users.RoleSuperAdmin {
		return actorID, nil
	}
	classExists, err := service.classes.Exists(ctx, classID)
	if err != nil {
		return uuid.Nil, err
	}
	if !classExists {
		return uuid.Nil, core.ErrNotFound
	}
	if role == users.RoleTeacher {
		assigned, err := service.assignments.Exists(ctx, classID, courseID, actorID)
		if err != nil {
			return uuid.Nil, err
		}
		if !assigned {
			return uuid.Nil, core.ErrForbidden
		}
		return actorID, nil
	}
	return uuid.Nil, core.ErrForbidden
}

func (service *service) invalidate(ctx context.Context, records []Attendance) {
	if service.invalidator == nil {
		return
	}
	for _, record := range records {
		_ = service.invalidator.InvalidateStudent(ctx, record.StudentID)
	}
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
