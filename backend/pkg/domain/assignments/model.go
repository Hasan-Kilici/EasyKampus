package assignments

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type ClassCourse struct {
	base.Model
	ClassID  uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_class_courses,priority:1,where:deleted_at IS NULL;index" json:"class_id"`
	CourseID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_class_courses,priority:2,where:deleted_at IS NULL;index" json:"course_id"`
	Term     string    `gorm:"size:32;not null;uniqueIndex:idx_class_courses,priority:3,where:deleted_at IS NULL" json:"term"`
	Active   bool      `gorm:"not null;default:true;index" json:"active"`
}

type TeacherAssignment struct {
	base.Model
	TeacherID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_teacher_assignments,priority:1,where:deleted_at IS NULL;index" json:"teacher_id"`
	ClassID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_teacher_assignments,priority:2,where:deleted_at IS NULL;index" json:"class_id"`
	CourseID  uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_teacher_assignments,priority:3,where:deleted_at IS NULL;index" json:"course_id"`
	Term      string    `gorm:"size:32;not null;uniqueIndex:idx_teacher_assignments,priority:4,where:deleted_at IS NULL" json:"term"`
	Active    bool      `gorm:"not null;default:true;index" json:"active"`
}

type ListParams struct {
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	ClassID   *uuid.UUID `json:"class_id"`
	CourseID  *uuid.UUID `json:"course_id"`
	TeacherID *uuid.UUID `json:"teacher_id"`
	Term      string     `json:"term"`
}

type Repository interface {
	CreateClassCourse(ctx context.Context, entity *ClassCourse) error
	GetClassCourse(ctx context.Context, id uuid.UUID) (*ClassCourse, error)
	DeleteClassCourse(ctx context.Context, id uuid.UUID) error
	ListClassCourses(ctx context.Context, params ListParams) ([]ClassCourse, error)
	ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error)

	CreateTeacherAssignment(ctx context.Context, entity *TeacherAssignment) error
	GetTeacherAssignment(ctx context.Context, id uuid.UUID) (*TeacherAssignment, error)
	DeleteTeacherAssignment(ctx context.Context, id uuid.UUID) error
	ListTeacherAssignments(ctx context.Context, params ListParams) ([]TeacherAssignment, error)
}
