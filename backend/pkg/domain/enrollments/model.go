package enrollments

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Status string

const (
	StatusActive    Status = "active"    // Öğrenci sınıfa kayıtlı ve derslere erişebilir.
	StatusWithdrawn Status = "withdrawn" // Kayıt sonlandırıldı.
	StatusCompleted Status = "completed" // Dönem tamamlandı.
)

type Enrollment struct {
	base.Model
	StudentID  uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_enrollments,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	ClassID    uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_enrollments,priority:2,where:deleted_at IS NULL;index" json:"class_id"`
	CourseID   *uuid.UUID `gorm:"type:char(36);uniqueIndex:idx_enrollments,priority:3,where:deleted_at IS NULL;index" json:"course_id"`
	Term       string     `gorm:"size:32;not null;uniqueIndex:idx_enrollments,priority:4,where:deleted_at IS NULL" json:"term"`
	Status     Status     `gorm:"size:20;not null;index" json:"status"`
	EnrolledAt time.Time  `gorm:"not null" json:"enrolled_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Note       string     `gorm:"type:text" json:"note"`
}

type ListParams struct {
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	ClassID   *uuid.UUID `json:"class_id"`
	StudentID *uuid.UUID `json:"student_id"`
	Status    Status     `json:"status"`
}

type Repository interface {
	Create(ctx context.Context, enrollment *Enrollment) error
	CreateBulk(ctx context.Context, enrollments []*Enrollment) error
	Transfer(ctx context.Context, fromClassID uuid.UUID, studentID uuid.UUID, enrollment *Enrollment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Enrollment, error)
	Update(ctx context.Context, enrollment *Enrollment) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Enrollment, error)
	CountByClass(ctx context.Context, classID uuid.UUID) (int64, error)
	Exists(ctx context.Context, classID uuid.UUID, studentID uuid.UUID, courseID *uuid.UUID, term string) (bool, error)
}
