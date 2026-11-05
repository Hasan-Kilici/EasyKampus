package enrollments

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/users"
)

type EnrollInput struct {
	StudentID string `json:"student_id"`
	CourseID  string `json:"course_id"`
	Term      string `json:"term"`
	Note      string `json:"note"`
}

type TransferInput struct {
	StudentID  string `json:"student_id"`
	NewClassID string `json:"new_class_id"`
	CourseID   string `json:"course_id"`
	Term       string `json:"term"`
	Note       string `json:"note"`
}

type Service interface {
	Enroll(ctx context.Context, classID string, input EnrollInput) (*Enrollment, error)
	EnrollBulk(ctx context.Context, classID string, inputs []EnrollInput) ([]Enrollment, error)
	Transfer(ctx context.Context, fromClassID string, input TransferInput) (*Enrollment, error)
	ListByClass(ctx context.Context, classID string) ([]Enrollment, error)
	ListByStudent(ctx context.Context, studentID string) ([]Enrollment, error)
	Withdraw(ctx context.Context, classID string, studentID string) error
}

func (service *service) EnrollBulk(ctx context.Context, classID string, inputs []EnrollInput) ([]Enrollment, error) {
	if len(inputs) == 0 {
		return nil, core.ErrValidation
	}
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	classExists, err := service.classes.Exists(ctx, parsedClassID)
	if err != nil {
		return nil, err
	}
	if !classExists {
		return nil, core.ErrNotFound
	}
	prepared := make([]*Enrollment, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		studentID, err := parseID(input.StudentID)
		if err != nil {
			return nil, core.ErrValidation
		}
		student, err := service.accounts.GetByID(ctx, studentID)
		if err != nil {
			return nil, err
		}
		if student.Role != users.RoleStudent {
			return nil, core.ErrValidation
		}
		courseID, err := parseOptionalID(input.CourseID)
		if err != nil {
			return nil, core.ErrValidation
		}
		if courseID != nil {
			offered, err := service.courses.ClassCourseExists(ctx, parsedClassID, *courseID)
			if err != nil {
				return nil, err
			}
			if !offered {
				return nil, core.ErrValidation
			}
		}
		term := strings.TrimSpace(input.Term)
		courseKey := ""
		if courseID != nil {
			courseKey = courseID.String()
		}
		dedupKey := studentID.String() + "|" + courseKey + "|" + term
		if _, duplicate := seen[dedupKey]; duplicate {
			return nil, core.ErrAlreadyExists
		}
		seen[dedupKey] = struct{}{}
		exists, err := service.repository.Exists(ctx, parsedClassID, studentID, courseID, term)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, core.ErrAlreadyExists
		}
		enrollment := &Enrollment{StudentID: studentID, ClassID: parsedClassID, CourseID: courseID, Term: term, Status: StatusActive, EnrolledAt: time.Now().UTC(), Note: strings.TrimSpace(input.Note)}
		if err := enrollment.Validate(); err != nil {
			return nil, err
		}
		prepared = append(prepared, enrollment)
	}
	if err := service.repository.CreateBulk(ctx, prepared); err != nil {
		return nil, err
	}
	result := make([]Enrollment, len(prepared))
	for index, enrollment := range prepared {
		result[index] = *enrollment
	}
	return result, nil
}

func (service *service) Transfer(ctx context.Context, fromClassID string, input TransferInput) (*Enrollment, error) {
	fromID, err := parseID(fromClassID)
	if err != nil {
		return nil, core.ErrValidation
	}
	toID, err := parseID(input.NewClassID)
	if err != nil || fromID == toID {
		return nil, core.ErrValidation
	}
	studentID, err := parseID(input.StudentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	if exists, err := service.classes.Exists(ctx, toID); err != nil {
		return nil, err
	} else if !exists {
		return nil, core.ErrNotFound
	}
	student, err := service.accounts.GetByID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if student.Role != users.RoleStudent {
		return nil, core.ErrValidation
	}
	courseID, err := parseOptionalID(input.CourseID)
	if err != nil {
		return nil, core.ErrValidation
	}
	if courseID != nil {
		_, err := service.courses.ClassCourseExists(ctx, toID, *courseID) //OFFERED GELECEK
		if err != nil {
			return nil, err
		}
	}
	old, err := service.repository.List(ctx, ListParams{ClassID: &fromID, StudentID: &studentID, Status: StatusActive, Limit: 100})
	if err != nil {
		return nil, err
	}
	if len(old) == 0 {
		return nil, core.ErrNotFound
	}
	term := strings.TrimSpace(input.Term)
	if exists, err := service.repository.Exists(ctx, toID, studentID, courseID, term); err != nil {
		return nil, err
	} else if exists {
		return nil, core.ErrAlreadyExists
	}
	newEnrollment := &Enrollment{StudentID: studentID, ClassID: toID, CourseID: courseID, Term: term, Status: StatusActive, EnrolledAt: time.Now().UTC(), Note: strings.TrimSpace(input.Note)}
	if err := newEnrollment.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Transfer(ctx, fromID, studentID, newEnrollment); err != nil {
		return nil, err
	}
	return newEnrollment, nil
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type UserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*users.User, error)
}

type CourseOfferLookup interface {
	ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error)
}

type service struct {
	repository Repository
	classes    ClassLookup
	accounts   UserLookup
	courses    CourseOfferLookup
}

func NewService(repository Repository, classes ClassLookup, accounts UserLookup, courses CourseOfferLookup) Service {
	return &service{repository: repository, classes: classes, accounts: accounts, courses: courses}
}

func (service *service) Enroll(ctx context.Context, classID string, input EnrollInput) (*Enrollment, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	studentID, err := parseID(input.StudentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	classExists, err := service.classes.Exists(ctx, parsedClassID)
	if err != nil {
		return nil, err
	}
	if !classExists {
		return nil, core.ErrNotFound
	}
	student, err := service.accounts.GetByID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if student.Role != users.RoleStudent {
		return nil, core.ErrValidation
	}
	courseID, err := parseOptionalID(input.CourseID)
	if err != nil {
		return nil, err
	}
	if courseID != nil {
		_, err := service.courses.ClassCourseExists(ctx, parsedClassID, *courseID) //OFFERED GELECEK
		if err != nil {
			return nil, err
		}
	}
	term := strings.TrimSpace(input.Term)
	exists, err := service.repository.Exists(ctx, parsedClassID, studentID, courseID, term)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, core.ErrAlreadyExists
	}
	enrollment := &Enrollment{
		StudentID:  studentID,
		ClassID:    parsedClassID,
		CourseID:   courseID,
		Term:       term,
		Status:     StatusActive,
		EnrolledAt: time.Now().UTC(),
		Note:       strings.TrimSpace(input.Note),
	}
	if err := enrollment.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, enrollment); err != nil {
		return nil, err
	}
	return enrollment, nil
}

func (service *service) ListByClass(ctx context.Context, classID string) ([]Enrollment, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.List(ctx, ListParams{ClassID: &parsedClassID, Status: StatusActive, Limit: 500})
}

func (service *service) ListByStudent(ctx context.Context, studentID string) ([]Enrollment, error) {
	parsedStudentID, err := parseID(studentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.List(ctx, ListParams{StudentID: &parsedStudentID, Status: StatusActive, Limit: 100})
}

func (service *service) Withdraw(ctx context.Context, classID string, studentID string) error {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return core.ErrValidation
	}
	parsedStudentID, err := parseID(studentID)
	if err != nil {
		return core.ErrValidation
	}
	enrollments, err := service.repository.List(ctx, ListParams{ClassID: &parsedClassID, StudentID: &parsedStudentID, Limit: 50})
	if err != nil {
		return err
	}
	if len(enrollments) == 0 {
		return core.ErrNotFound
	}
	now := time.Now().UTC()
	for index := range enrollments {
		enrollments[index].Status = StatusWithdrawn
		enrollments[index].FinishedAt = &now
		if err := service.repository.Update(ctx, &enrollments[index]); err != nil {
			return err
		}
	}
	return nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }

func parseOptionalID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseID(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
