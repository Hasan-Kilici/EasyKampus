package lessons

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/base"
)

type Lesson struct {
	base.Model
	CourseID        uuid.UUID `gorm:"type:char(36);not null;index" json:"course_id"`
	Title           string    `gorm:"size:200;not null" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	MarkdownContent string    `gorm:"type:text" json:"markdown_content"`
	Position        int       `gorm:"not null;default:0;index" json:"position"`
	DurationMinutes int       `gorm:"not null;default:0" json:"duration_minutes"`
	CreatedByID     uuid.UUID `gorm:"type:char(36);not null;index" json:"created_by_id"`
	approvals.Review
}

type ListParams struct {
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
	CourseID      uuid.UUID `json:"course_id"`
	OnlyPublished bool      `json:"only_published"`
}

type LessonCompletion struct {
	base.Model
	StudentID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_lesson_completions,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	LessonID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_lesson_completions,priority:2,where:deleted_at IS NULL;index" json:"lesson_id"`
	CourseID    uuid.UUID `gorm:"type:char(36);not null;index" json:"course_id"`
	CompletedAt time.Time `gorm:"not null;index" json:"completed_at"`
}

type Repository interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id uuid.UUID) (*Lesson, error)
	Update(ctx context.Context, lesson *Lesson) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCourse(ctx context.Context, params ListParams) ([]Lesson, error)
	UpsertCompletions(ctx context.Context, completions []LessonCompletion) error
	ListCompletions(ctx context.Context, studentID uuid.UUID, courseID *uuid.UUID) ([]LessonCompletion, error)
}
