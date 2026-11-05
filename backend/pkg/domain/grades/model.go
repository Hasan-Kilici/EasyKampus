package grades

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type GradeType string

const (
	GradeTypeExam          GradeType = "exam"          // Sınav notu.
	GradeTypeAssignment    GradeType = "assignment"    // Ödev notu.
	GradeTypeProject       GradeType = "project"       // Proje notu.
	GradeTypeParticipation GradeType = "participation" // Katılım notu.
)

type Grade struct {
	base.Model
	StudentID uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_grades_scope,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	ClassID   uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_grades_scope,priority:2,where:deleted_at IS NULL;index" json:"class_id"`
	CourseID  uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_grades_scope,priority:3,where:deleted_at IS NULL;index" json:"course_id"`
	Term      string     `gorm:"size:32;not null;uniqueIndex:idx_grades_scope,priority:4,where:deleted_at IS NULL;index" json:"term"`
	QuizID    *uuid.UUID `gorm:"type:char(36);index" json:"quiz_id"`
	Score     float64    `gorm:"not null;index" json:"score"`
	Type      GradeType  `gorm:"size:20;not null;index" json:"type"`
	GradedBy  *uuid.UUID `gorm:"type:char(36);index" json:"graded_by"`
	Note      string     `gorm:"size:255" json:"note"`
}

type Audit struct {
	base.Model
	GradeID     uuid.UUID `gorm:"type:char(36);not null;index" json:"grade_id"`
	ActorID     uuid.UUID `gorm:"type:char(36);not null;index" json:"actor_id"`
	Action      string    `gorm:"size:16;not null;index" json:"action"`
	BeforeScore *float64  `json:"before_score"`
	AfterScore  *float64  `json:"after_score"`
	BeforeNote  string    `gorm:"size:255" json:"before_note"`
	AfterNote   string    `gorm:"size:255" json:"after_note"`
}
type ListParams struct {
	ClassID      *uuid.UUID  `json:"class_id"`
	ClassIDs     []uuid.UUID `json:"-"`
	ScopeApplied bool        `json:"-"`
	StudentID    *uuid.UUID  `json:"student_id"`
	CourseID     *uuid.UUID  `json:"course_id"`
	QuizID       *uuid.UUID  `json:"quiz_id"`
	TeacherID    *uuid.UUID  `json:"teacher_id"`
	Term         string      `json:"term"`
	Type         GradeType   `json:"type"`
	Limit        int         `json:"limit"`
}

type Repository interface {
	Upsert(ctx context.Context, grade *Grade) error
	BulkUpsert(ctx context.Context, grades []*Grade) error
	BulkUpsertWithAudits(ctx context.Context, grades []*Grade, audits []*Audit) error
	CreateAudits(ctx context.Context, audits []*Audit) error
	ListAudits(ctx context.Context, gradeID uuid.UUID) ([]Audit, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Grade, error)
	Update(ctx context.Context, grade *Grade) error
	List(ctx context.Context, params ListParams) ([]Grade, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, term string) ([]Grade, error)
}

type CreateInput struct {
	StudentID string  `json:"student_id"`
	ClassID   string  `json:"class_id"`
	CourseID  string  `json:"course_id"`
	Term      string  `json:"term"`
	QuizID    string  `json:"quiz_id"`
	Type      string  `json:"type"`
	Score     float64 `json:"score"`
	Note      string  `json:"note"`
}

type UpdateInput struct {
	Score float64 `json:"score"`
	Note  string  `json:"note"`
}
