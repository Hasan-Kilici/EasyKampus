package options

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Option struct {
	base.Model
	QuestionID uuid.UUID `gorm:"type:char(36);not null;index" json:"question_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Answer     bool      `gorm:"not null;default:false" json:"answer"`
	Position   int       `gorm:"not null;default:0;index" json:"position"`
}

type ListParams struct {
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	QuestionID uuid.UUID `json:"question_id"`
}

type Repository interface {
	Create(ctx context.Context, option *Option) error
	GetByID(ctx context.Context, id uuid.UUID) (*Option, error)
	GetByIDInQuestion(ctx context.Context, id uuid.UUID, questionID uuid.UUID) (*Option, error)
	Update(ctx context.Context, option *Option) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByQuestion(ctx context.Context, questionID uuid.UUID, page int, limit int) ([]Option, error)
}
