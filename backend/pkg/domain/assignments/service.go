package assignments

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type AssignCourseInput struct {
	CourseID string `json:"course_id"`
	Term     string `json:"term"`
}

type AssignTeacherInput struct {
	TeacherID string `json:"teacher_id"`
	CourseID  string `json:"course_id"`
	Term      string `json:"term"`
}

type Service interface {
	AssignCourse(ctx context.Context, classID string, input AssignCourseInput) (*ClassCourse, error)
	ListCourses(ctx context.Context, classID string) ([]ClassCourse, error)
	RemoveCourse(ctx context.Context, classID string, courseID string) error
	AssignTeacher(ctx context.Context, classID string, input AssignTeacherInput) (*TeacherAssignment, error)
	ListTeachers(ctx context.Context, classID string) ([]TeacherAssignment, error)
	RemoveTeacher(ctx context.Context, classID string, teacherID string) error
}

type ClassLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CourseLookup interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type service struct {
	repository Repository
	classes    ClassLookup
	courses    CourseLookup
}

func NewService(repository Repository, classes ClassLookup, courses CourseLookup) Service {
	return &service{repository: repository, classes: classes, courses: courses}
}

func (service *service) AssignCourse(ctx context.Context, classID string, input AssignCourseInput) (*ClassCourse, error) {
	parsedClassID, courseID, term, err := parseAssignInput(classID, input.CourseID, input.Term)
	if err != nil {
		return nil, err
	}
	if err := service.verifyExists(ctx, parsedClassID, courseID); err != nil {
		return nil, err
	}
	exists, err := service.repository.ClassCourseExists(ctx, parsedClassID, courseID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, core.ErrAlreadyExists
	}
	entity := &ClassCourse{ClassID: parsedClassID, CourseID: courseID, Term: term, Active: true}
	if err := entity.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.CreateClassCourse(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (service *service) ListCourses(ctx context.Context, classID string) ([]ClassCourse, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListClassCourses(ctx, ListParams{ClassID: &parsedClassID, Limit: 200})
}

func (service *service) RemoveCourse(ctx context.Context, classID string, courseID string) error {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return core.ErrValidation
	}
	parsedCourseID, err := parseID(courseID)
	if err != nil {
		return core.ErrValidation
	}
	entities, err := service.repository.ListClassCourses(ctx, ListParams{ClassID: &parsedClassID, CourseID: &parsedCourseID, Limit: 10})
	if err != nil {
		return err
	}
	if len(entities) == 0 {
		return core.ErrNotFound
	}
	for index := range entities {
		if err := service.repository.DeleteClassCourse(ctx, entities[index].ID); err != nil {
			return err
		}
	}
	return nil
}

func (service *service) AssignTeacher(ctx context.Context, classID string, input AssignTeacherInput) (*TeacherAssignment, error) {
	parsedClassID, courseID, term, err := parseAssignInput(classID, input.CourseID, input.Term)
	if err != nil {
		return nil, err
	}
	teacherID, err := parseID(input.TeacherID)
	if err != nil {
		return nil, core.ErrValidation
	}
	if err := service.verifyExists(ctx, parsedClassID, courseID); err != nil {
		return nil, err
	}
	offered, err := service.repository.ClassCourseExists(ctx, parsedClassID, courseID)
	if err != nil {
		return nil, err
	}
	if !offered {
		return nil, core.ErrValidation
	}
	entity := &TeacherAssignment{TeacherID: teacherID, ClassID: parsedClassID, CourseID: courseID, Term: term, Active: true}
	if err := entity.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.CreateTeacherAssignment(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (service *service) ListTeachers(ctx context.Context, classID string) ([]TeacherAssignment, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.ListTeacherAssignments(ctx, ListParams{ClassID: &parsedClassID, Limit: 200})
}

func (service *service) RemoveTeacher(ctx context.Context, classID string, teacherID string) error {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return core.ErrValidation
	}
	parsedTeacherID, err := parseID(teacherID)
	if err != nil {
		return core.ErrValidation
	}
	entities, err := service.repository.ListTeacherAssignments(ctx, ListParams{ClassID: &parsedClassID, TeacherID: &parsedTeacherID, Limit: 50})
	if err != nil {
		return err
	}
	if len(entities) == 0 {
		return core.ErrNotFound
	}
	for index := range entities {
		if err := service.repository.DeleteTeacherAssignment(ctx, entities[index].ID); err != nil {
			return err
		}
	}
	return nil
}

func (service *service) verifyExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) error {
	classExists, err := service.classes.Exists(ctx, classID)
	if err != nil {
		return err
	}
	if !classExists {
		return core.ErrNotFound
	}
	courseExists, err := service.courses.Exists(ctx, courseID)
	if err != nil {
		return err
	}
	if !courseExists {
		return core.ErrNotFound
	}
	return nil
}

func parseAssignInput(classID string, courseID string, term string) (uuid.UUID, uuid.UUID, string, error) {
	parsedClassID, err := parseID(classID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", core.ErrValidation
	}
	parsedCourseID, err := parseID(courseID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", core.ErrValidation
	}
	trimmedTerm := strings.TrimSpace(term)
	if trimmedTerm == "" || len(trimmedTerm) > 32 {
		return uuid.Nil, uuid.Nil, "", core.ErrValidation
	}
	return parsedClassID, parsedCourseID, trimmedTerm, nil
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
