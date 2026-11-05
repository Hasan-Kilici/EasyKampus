package classes

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/users"
)

type Class struct {
	base.Model
	SchoolID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_classes_school_level_no,priority:1;index" json:"school_id"`
	CampusID uuid.UUID `gorm:"type:char(36);not null;index" json:"campus_id"`
	Name     string    `gorm:"size:100;not null" json:"name"`
	Level    string    `gorm:"size:100;not null;uniqueIndex:idx_classes_school_level_no,priority:2" json:"level"`
	No       string    `gorm:"size:1;not null;uniqueIndex:idx_classes_school_level_no,priority:3" json:"no"`
}

type ListParams struct {
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
	SchoolID *uuid.UUID `json:"school_id"`
	CampusID *uuid.UUID `json:"campus_id"`
	CourseID *uuid.UUID `json:"course_id"`
	Query    string     `json:"query"`
}

type Scope struct {
	UserID uuid.UUID  `json:"user_id"`
	Role   users.Role `json:"role"`
}

type Repository interface {
	Create(ctx context.Context, class *Class) error
	GetByID(ctx context.Context, id uuid.UUID) (*Class, error)
	Update(ctx context.Context, class *Class) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams, scope Scope) ([]Class, error)
	Count(ctx context.Context, params ListParams, scope Scope) (int64, error)
}

type AccessRepository interface {
	HasAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error)
	HasClassManagementAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error)
	HasTeacherAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error)
	HasSchoolAccess(ctx context.Context, userID uuid.UUID, schoolID uuid.UUID) (bool, error)
	HasSchoolStaffAccess(ctx context.Context, userID uuid.UUID, schoolID uuid.UUID) (bool, error)
	HasTeacherSubjectAccess(ctx context.Context, userID uuid.UUID, teacherID uuid.UUID) (bool, error)
	HasCampusAccess(ctx context.Context, userID uuid.UUID, campusID uuid.UUID) (bool, error)
	HasCourseAccess(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (bool, error)
	HasCourseWriteAccess(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (bool, error)
	HasStudentAccess(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (bool, error)
	ScopedClassIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	SchoolIDsForStaff(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
