package questions

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/base"
)

type Question struct {
	base.Model
	QuizID      uuid.UUID  `gorm:"type:char(36);not null;index" json:"quiz_id"`
	TopicID     *uuid.UUID `gorm:"type:char(36);index" json:"topic_id"`
	Question    string     `gorm:"type:text;not null" json:"question"`
	Points      float64    `gorm:"not null;default:1" json:"points"`
	Position    int        `gorm:"not null;default:0;index" json:"position"`
	CreatedByID uuid.UUID  `gorm:"type:char(36);not null;index" json:"created_by_id"`
	approvals.Review
}

type ListParams struct {
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
	QuizID        uuid.UUID `json:"quiz_id"`
	OnlyPublished bool      `json:"only_published"`
}

type Repository interface {
	Create(ctx context.Context, question *Question) error
	GetByID(ctx context.Context, id uuid.UUID) (*Question, error)
	Update(ctx context.Context, question *Question) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByQuiz(ctx context.Context, params ListParams) ([]Question, error)
}
