package notes

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Note struct {
	base.Model
	AuthorID         uuid.UUID  `gorm:"type:char(36);not null;index" json:"author_id"`
	StudentID        uuid.UUID  `gorm:"type:char(36);not null;index" json:"student_id"`
	ClassID          *uuid.UUID `gorm:"type:char(36);index" json:"class_id"`
	CourseID         *uuid.UUID `gorm:"type:char(36);index" json:"course_id"`
	Body             string     `gorm:"size:2000;not null" json:"body"`
	VisibleToStudent bool       `gorm:"not null;default:false" json:"visible_to_student"`
}

type ListParams struct {
	AuthorID     *uuid.UUID  `json:"author_id"`
	StudentID    *uuid.UUID  `json:"student_id"`
	ClassID      *uuid.UUID  `json:"class_id"`
	ClassIDs     []uuid.UUID `json:"-"`
	ScopeApplied bool        `json:"-"`
	CourseID     *uuid.UUID  `json:"course_id"`
	SharedOnly   bool        `json:"shared_only"`
	Limit        int         `json:"limit"`
}

type Repository interface {
	Create(ctx context.Context, note *Note) error
	GetByID(ctx context.Context, id uuid.UUID) (*Note, error)
	Update(ctx context.Context, note *Note) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Note, error)
}
