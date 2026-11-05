package attendance

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Status string

const (
	StatusPresent Status = "present" // Derse katıldı.
	StatusAbsent  Status = "absent"  // Devamsız.
	StatusLate    Status = "late"    // Geç kaldı.
	StatusExcused Status = "excused" // Mazeretli.
)

type Attendance struct {
	base.Model
	StudentID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_attendance_records,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	CourseID  uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_attendance_records,priority:2,where:deleted_at IS NULL;index" json:"course_id"`
	ClassID   uuid.UUID `gorm:"type:char(36);not null;index" json:"class_id"`
	Date      string    `gorm:"size:10;not null;uniqueIndex:idx_attendance_records,priority:3,where:deleted_at IS NULL;index" json:"date"`
	TeacherID uuid.UUID `gorm:"type:char(36);not null;index" json:"teacher_id"`
	Status    Status    `gorm:"size:20;not null;index" json:"status"`
	Note      string    `gorm:"size:255" json:"note"`
}

type ListParams struct {
	ClassID      *uuid.UUID  `json:"class_id"`
	ClassIDs     []uuid.UUID `json:"-"`
	ScopeApplied bool        `json:"-"`
	CourseID     *uuid.UUID  `json:"course_id"`
	StudentID    *uuid.UUID  `json:"student_id"`
	TeacherID    *uuid.UUID  `json:"teacher_id"`
	From         string      `json:"from"`
	To           string      `json:"to"`
	Status       Status      `json:"status"`
	Limit        int         `json:"limit"`
}

type Summary struct {
	Total   int     `json:"total"`
	Present int     `json:"present"`
	Absent  int     `json:"absent"`
	Late    int     `json:"late"`
	Excused int     `json:"excused"`
	Rate    float64 `json:"rate"`
}

type DaySummary struct {
	Date    string `json:"date"`
	Total   int    `json:"total"`
	Present int    `json:"present"`
	Absent  int    `json:"absent"`
	Late    int    `json:"late"`
	Excused int    `json:"excused"`
}

type PeriodSummary struct {
	Summary
	Days []DaySummary `json:"days"`
}

type Repository interface {
	Upsert(ctx context.Context, records []Attendance) error
	GetByID(ctx context.Context, id uuid.UUID) (*Attendance, error)
	Update(ctx context.Context, record *Attendance) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Attendance, error)
	Summarize(ctx context.Context, params ListParams) (PeriodSummary, error)
}
