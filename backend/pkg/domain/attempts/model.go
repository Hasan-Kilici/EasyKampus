package attempts

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Status string

const (
	StatusInProgress Status = "in_progress" // Sınav çözülüyor.
	StatusCompleted  Status = "completed"   // Sınav teslim edildi ve puanlandı.
	StatusAbandoned  Status = "abandoned"   // Süre doldu, teslim edilmedi.
)

type QuizAttempt struct {
	base.Model
	StudentID       uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_attempts_active,priority:1,where:deleted_at IS NULL AND status = 'in_progress';index" json:"student_id"`
	QuizID          uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_attempts_active,priority:2,where:deleted_at IS NULL AND status = 'in_progress';index" json:"quiz_id"`
	Status          Status          `gorm:"size:20;not null;index" json:"status"`
	StartedAt       time.Time       `gorm:"not null;index" json:"started_at"`
	CompletedAt     *time.Time      `gorm:"index" json:"completed_at"`
	Score           float64         `gorm:"not null;default:0" json:"score"`
	MaxScore        float64         `gorm:"not null;default:0" json:"max_score"`
	DurationSeconds int             `gorm:"not null;default:0" json:"duration_seconds"`
	ClassID         uuid.UUID       `gorm:"type:char(36);not null;index" json:"class_id"`
	CourseID        uuid.UUID       `gorm:"type:char(36);not null;index" json:"course_id"`
	Answers         []AttemptAnswer `gorm:"foreignKey:AttemptID" json:"answers"`
}

type AttemptAnswer struct {
	base.Model
	AttemptID        uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_attempt_answers,priority:1,where:deleted_at IS NULL;index" json:"attempt_id"`
	QuestionID       uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_attempt_answers,priority:2,where:deleted_at IS NULL;index" json:"question_id"`
	TopicID          *uuid.UUID `gorm:"type:char(36);index" json:"topic_id"`
	SelectedOptionID *uuid.UUID `gorm:"type:char(36);index" json:"selected_option_id"`
	Correct          bool       `gorm:"not null;default:false" json:"correct"`
	Points           float64    `gorm:"not null;default:0" json:"points"`
	AnsweredAt       time.Time  `gorm:"not null" json:"answered_at"`
}

type ListParams struct {
	StudentID    *uuid.UUID  `json:"student_id"`
	QuizID       *uuid.UUID  `json:"quiz_id"`
	ClassID      *uuid.UUID  `json:"class_id"`
	ClassIDs     []uuid.UUID `json:"-"`
	ScopeApplied bool        `json:"-"`
	Status       Status      `json:"status"`
	Limit        int         `json:"limit"`
}

type Repository interface {
	Create(ctx context.Context, attempt *QuizAttempt) error
	GetByID(ctx context.Context, id uuid.UUID) (*QuizAttempt, error)
	Update(ctx context.Context, attempt *QuizAttempt) error
	SaveAnswers(ctx context.Context, answers []AttemptAnswer) error
	List(ctx context.Context, params ListParams) ([]QuizAttempt, error)
	CountByStudentAndQuiz(ctx context.Context, studentID uuid.UUID, quizID uuid.UUID, status Status) (int64, error)
}
