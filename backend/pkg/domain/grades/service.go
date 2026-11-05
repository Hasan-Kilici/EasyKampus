package grades

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/users"
)

type Service interface {
	Record(ctx context.Context, actorID uuid.UUID, role users.Role, input CreateInput) (*Grade, error)
	RecordBulk(ctx context.Context, actorID uuid.UUID, role users.Role, inputs []CreateInput) ([]Grade, error)
	Update(ctx context.Context, actorID uuid.UUID, role users.Role, id string, input UpdateInput) (*Grade, error)
	Delete(ctx context.Context, actorID uuid.UUID, role users.Role, id string) error
	List(ctx context.Context, params ListParams) ([]Grade, error)
	ListAudits(ctx context.Context, id string) ([]Audit, error)
}

func (service *service) ListAudits(ctx context.Context, id string) ([]Audit, error) {
	gradeID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListAudits(ctx, gradeID)
}

func (service *service) RecordBulk(ctx context.Context, actorID uuid.UUID, role users.Role, inputs []CreateInput) ([]Grade, error) {
	if len(inputs) == 0 {
		return nil, core.ErrValidation
	}
	prepared := make([]*Grade, 0, len(inputs))
	for _, input := range inputs {
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
		quizID, err := parseOptionalID(input.QuizID)
		if err != nil {
			return nil, core.ErrValidation
		}
		grade := &Grade{StudentID: studentID, ClassID: classID, CourseID: courseID, Term: strings.TrimSpace(input.Term), QuizID: quizID, Score: input.Score, Type: GradeType(strings.TrimSpace(input.Type)), GradedBy: &actorID, Note: strings.TrimSpace(input.Note)}
		if err := grade.Validate(); err != nil {
			return nil, err
		}
		if err := service.allowed(ctx, actorID, role, classID, courseID, studentID); err != nil {
			return nil, err
		}
		prepared = append(prepared, grade)
	}
	audits := make([]*Audit, 0, len(prepared))
	for _, grade := range prepared {
		audits = append(audits, &Audit{GradeID: grade.ID, ActorID: actorID, Action: "create", AfterScore: &grade.Score, AfterNote: grade.Note})
	}
	if err := service.repository.BulkUpsertWithAudits(ctx, prepared, audits); err != nil {
		return nil, err
	}
	for _, grade := range prepared {
		service.invalidate(ctx, grade)
	}
	result := make([]Grade, len(prepared))
	for index, grade := range prepared {
		result[index] = *grade
	}
	return result, nil
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

func (service *service) Record(ctx context.Context, actorID uuid.UUID, role users.Role, input CreateInput) (*Grade, error) {
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
	quizID, err := parseOptionalID(input.QuizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	grade := &Grade{
		StudentID: studentID,
		ClassID:   classID,
		CourseID:  courseID,
		Term:      strings.TrimSpace(input.Term),
		QuizID:    quizID,
		Score:     input.Score,
		Type:      GradeType(strings.TrimSpace(input.Type)),
		GradedBy:  &actorID,
		Note:      strings.TrimSpace(input.Note),
	}
	if err := grade.Validate(); err != nil {
		return nil, err
	}
	if err := service.allowed(ctx, actorID, role, classID, courseID, studentID); err != nil {
		return nil, err
	}
	if err := service.repository.Upsert(ctx, grade); err != nil {
		return nil, err
	}
	if err := service.repository.CreateAudits(ctx, []*Audit{{GradeID: grade.ID, ActorID: actorID, Action: "create", AfterScore: &grade.Score, AfterNote: grade.Note}}); err != nil {
		return nil, err
	}
	service.invalidate(ctx, grade)
	return grade, nil
}

func (service *service) Update(ctx context.Context, actorID uuid.UUID, role users.Role, id string, input UpdateInput) (*Grade, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	grade, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	if err := service.allowed(ctx, actorID, role, grade.ClassID, grade.CourseID, grade.StudentID); err != nil {
		return nil, err
	}
	beforeScore, beforeNote := grade.Score, grade.Note
	grade.Score = input.Score
	grade.Note = strings.TrimSpace(input.Note)
	grade.GradedBy = &actorID
	if err := grade.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, grade); err != nil {
		return nil, err
	}
	if err := service.repository.CreateAudits(ctx, []*Audit{{GradeID: grade.ID, ActorID: actorID, Action: "update", BeforeScore: &beforeScore, AfterScore: &grade.Score, BeforeNote: beforeNote, AfterNote: grade.Note}}); err != nil {
		return nil, err
	}
	service.invalidate(ctx, grade)
	return grade, nil
}

func (service *service) Delete(ctx context.Context, actorID uuid.UUID, role users.Role, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	grade, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return err
	}
	if err := service.allowed(ctx, actorID, role, grade.ClassID, grade.CourseID, grade.StudentID); err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, parsedID); err != nil {
		return err
	}
	if err := service.repository.CreateAudits(ctx, []*Audit{{GradeID: grade.ID, ActorID: actorID, Action: "delete", BeforeScore: &grade.Score, BeforeNote: grade.Note}}); err != nil {
		return err
	}
	service.invalidate(ctx, grade)
	return nil
}

func (service *service) List(ctx context.Context, params ListParams) ([]Grade, error) {
	if params.Limit <= 0 || params.Limit > 500 {
		params.Limit = 100
	}
	return service.repository.List(ctx, params)
}

func (service *service) allowed(ctx context.Context, actorID uuid.UUID, role users.Role, classID uuid.UUID, courseID uuid.UUID, studentID uuid.UUID) error {
	switch role {
	case users.RoleSuperAdmin:
	case users.RoleAdmin, users.RoleTeacher:
		assigned, err := service.assignments.Exists(ctx, classID, courseID, actorID)
		if err != nil {
			return err
		}
		if !assigned {
			return core.ErrForbidden
		}
	default:
		return core.ErrForbidden
	}
	exists, err := service.classes.Exists(ctx, classID)
	if err != nil {
		return err
	}
	if !exists {
		return core.ErrNotFound
	}
	enrolled, err := service.enrollments.Exists(ctx, classID, studentID, &courseID)
	if err != nil {
		return err
	}
	if !enrolled {
		return core.ErrValidation
	}
	return nil
}

func (service *service) invalidate(ctx context.Context, grade *Grade) {
	if service.invalidator == nil {
		return
	}
	_ = service.invalidator.InvalidateStudent(ctx, grade.StudentID)
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
