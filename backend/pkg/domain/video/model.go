package video

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Video struct {
	base.Model
	LessonID    uuid.UUID `gorm:"type:char(36);not null;index" json:"lesson_id"`
	Title       string    `gorm:"size:200;not null" json:"title"`
	URL         string    `gorm:"size:500;not null" json:"url"`
	Duration    int       `gorm:"not null;default:0" json:"duration"`
	Position    int       `gorm:"not null;default:0;index" json:"position"`
	CreatedByID uuid.UUID `gorm:"type:char(36);not null;index" json:"created_by_id"`
}

type ListParams struct {
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	LessonID uuid.UUID `json:"lesson_id"`
}

type VideoCompletion struct {
	base.Model
	StudentID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_video_completions,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	VideoID     uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_video_completions,priority:2,where:deleted_at IS NULL;index" json:"video_id"`
	LessonID    uuid.UUID `gorm:"type:char(36);not null;index" json:"lesson_id"`
	CourseID    uuid.UUID `gorm:"type:char(36);not null;index" json:"course_id"`
	CompletedAt time.Time `gorm:"not null;index" json:"completed_at"`
}

type Repository interface {
	Create(ctx context.Context, video *Video) error
	GetByID(ctx context.Context, id uuid.UUID) (*Video, error)
	Update(ctx context.Context, video *Video) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByLesson(ctx context.Context, params ListParams) ([]Video, error)
	UpsertCompletions(ctx context.Context, completions []VideoCompletion) error
	ListCompletions(ctx context.Context, studentID uuid.UUID, lessonID *uuid.UUID) ([]VideoCompletion, error)
}
