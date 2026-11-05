package schedules

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type ScheduleEntry struct {
	base.Model
	ClassID     uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_schedule_entries,priority:1,where:deleted_at IS NULL;index" json:"class_id"`
	CourseID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_schedule_entries,priority:2,where:deleted_at IS NULL;index" json:"course_id"`
	TeacherID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_schedule_entries,priority:3,where:deleted_at IS NULL;index" json:"teacher_id"`
	Term        string    `gorm:"size:32;not null;uniqueIndex:idx_schedule_entries,priority:4,where:deleted_at IS NULL" json:"term"`
	DayOfWeek   int       `gorm:"not null;index" json:"day_of_week"`
	StartMinute int       `gorm:"not null" json:"start_minute"`
	EndMinute   int       `gorm:"not null" json:"end_minute"`
	Active      bool      `gorm:"not null;default:true;index" json:"active"`
}

type ListParams struct {
	ClassID      *uuid.UUID  `json:"class_id"`
	ClassIDs     []uuid.UUID `json:"-"`
	ScopeApplied bool        `json:"-"`
	CourseID     *uuid.UUID  `json:"course_id"`
	TeacherID    *uuid.UUID  `json:"teacher_id"`
	Term         string      `json:"term"`
	DayOfWeek    *int        `json:"day_of_week"`
	ActiveOnly   bool        `json:"active_only"`
}

type Repository interface {
	Create(ctx context.Context, entry *ScheduleEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*ScheduleEntry, error)
	Update(ctx context.Context, entry *ScheduleEntry) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams, limit int) ([]ScheduleEntry, error)
}
