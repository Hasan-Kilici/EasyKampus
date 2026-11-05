package terms

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

type Term struct {
	base.Model
	SchoolID uuid.UUID  `gorm:"type:char(36);not null;index:idx_terms_school_status" json:"school_id"`
	Name     string     `gorm:"size:32;not null;uniqueIndex:idx_terms_school_name" json:"name"`
	StartsAt time.Time  `gorm:"not null" json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	Status   Status     `gorm:"size:16;not null;index:idx_terms_school_status" json:"status"`
}

type StartInput struct {
	SchoolID string    `json:"school_id"`
	Name     string    `json:"name"`
	StartsAt time.Time `json:"starts_at"`
}

type Service interface {
	Start(ctx context.Context, input StartInput) (*Term, error)
	Close(ctx context.Context, id string) (*Term, error)
	List(ctx context.Context, schoolID string) ([]Term, error)
}

type Repository interface {
	Create(ctx context.Context, term *Term) error
	GetByID(ctx context.Context, id uuid.UUID) (*Term, error)
	List(ctx context.Context, schoolID uuid.UUID) ([]Term, error)
	HasActive(ctx context.Context, schoolID uuid.UUID) (bool, error)
	Update(ctx context.Context, term *Term) error
}
