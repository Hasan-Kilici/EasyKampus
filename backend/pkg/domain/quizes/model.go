package quizes

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/base"
)

type Quiz struct {
	base.Model
	ClassID         uuid.UUID  `gorm:"type:char(36);not null;index" json:"class_id"`
	CourseID        uuid.UUID  `gorm:"type:char(36);not null;index" json:"course_id"`
	Name            string     `gorm:"size:200;not null" json:"name"`
	Description     string     `gorm:"type:text" json:"description"`
	OpenAt          *time.Time `gorm:"index" json:"open_at"`
	CloseAt         *time.Time `gorm:"index" json:"close_at"`
	DurationMinutes int        `gorm:"not null;default:0" json:"duration_minutes"`
	Points          float64    `gorm:"not null;default:0" json:"points"`
	MaxAttempts     int        `gorm:"not null;default:1" json:"max_attempts"`
	CreatedByID     uuid.UUID  `gorm:"type:char(36);not null;index" json:"created_by_id"`
	approvals.Review
}

type ListParams struct {
	Page          int        `json:"page"`
	Limit         int        `json:"limit"`
	ClassID       uuid.UUID  `json:"class_id"`
	CourseID      *uuid.UUID `json:"course_id"`
	OnlyPublished bool       `json:"only_published"`
}

type Repository interface {
	Create(ctx context.Context, quiz *Quiz) error
	GetByID(ctx context.Context, id uuid.UUID) (*Quiz, error)
	Update(ctx context.Context, quiz *Quiz) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByClass(ctx context.Context, params ListParams) ([]Quiz, error)
}
