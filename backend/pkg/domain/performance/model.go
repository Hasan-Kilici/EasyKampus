package performance

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

const ContractVersion = 1

type SubjectType string

const (
	SubjectStudent SubjectType = "student"
	SubjectTeacher SubjectType = "teacher"
)

type Status string

const (
	StatusExcellent      Status = "excellent"
	StatusGood           Status = "good"
	StatusNeedsAttention Status = "needs_attention"
	StatusWeak           Status = "weak"
	StatusUnknown        Status = "unknown"
)

type Severity string

const (
	SeverityInfo   Severity = "info"
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Snapshot struct {
	base.Model
	SubjectType   SubjectType `gorm:"size:20;not null;index:idx_snapshots_subject,priority:1" json:"subject_type"`
	SubjectID     uuid.UUID   `gorm:"type:char(36);not null;index:idx_snapshots_subject,priority:2" json:"subject_id"`
	Version       int         `gorm:"not null" json:"version"`
	Engine        string      `gorm:"size:50" json:"engine"`
	OverallScore  float64     `gorm:"not null" json:"overall_score"`
	OverallStatus Status      `gorm:"size:20;not null;index" json:"overall_status"`
	GeneratedAt   time.Time   `gorm:"not null;index" json:"generated_at"`
	Metadata      string      `gorm:"type:text" json:"metadata"`
}

type SnapshotCourse struct {
	base.Model
	SnapshotID     uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_courses,priority:1,where:deleted_at IS NULL;index" json:"snapshot_id"`
	CourseID       uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_courses,priority:2,where:deleted_at IS NULL;index" json:"course_id"`
	Score          float64   `gorm:"not null" json:"score"`
	Status         Status    `gorm:"size:20;not null;index" json:"status"`
	ClassCount     int       `gorm:"not null;default:0" json:"class_count"`
	StudentCount   int       `gorm:"not null;default:0" json:"student_count"`
	AttendanceRate float64   `gorm:"not null;default:0" json:"attendance_rate"`
	ActivityScore  float64   `gorm:"not null;default:0" json:"activity_score"`
	CompletionRate float64   `gorm:"not null;default:0" json:"completion_rate"`
	QuizCount      int       `gorm:"not null;default:0" json:"quiz_count"`
	CompletedQuiz  int       `gorm:"not null;default:0" json:"completed_quiz"`
}

type SnapshotTopic struct {
	base.Model
	SnapshotID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_topics,priority:1,where:deleted_at IS NULL;index" json:"snapshot_id"`
	CourseID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_topics,priority:2,where:deleted_at IS NULL;index" json:"course_id"`
	TopicID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_topics,priority:3,where:deleted_at IS NULL;index" json:"topic_id"`
	Score      float64   `gorm:"not null" json:"score"`
	Status     Status    `gorm:"size:20;not null;index" json:"status"`
	Correct    int       `gorm:"not null;default:0" json:"correct"`
	Questions  int       `gorm:"not null;default:0" json:"questions"`
}

type Problem struct {
	base.Model
	SnapshotID  uuid.UUID  `gorm:"type:char(36);not null;index" json:"snapshot_id"`
	Type        string     `gorm:"size:60;not null;index" json:"type"`
	Severity    Severity   `gorm:"size:20;not null;index" json:"severity"`
	Subject     string     `gorm:"size:150;index" json:"subject"`
	TopicID     *uuid.UUID `gorm:"type:char(36);index" json:"topic_id"`
	Message     string     `gorm:"type:text;not null" json:"message"`
	GeneratedAt time.Time  `gorm:"not null;index" json:"generated_at"`
}

type Recommendation struct {
	base.Model
	SnapshotID  uuid.UUID  `gorm:"type:char(36);not null;index" json:"snapshot_id"`
	Type        string     `gorm:"size:60;not null;index" json:"type"`
	Severity    Severity   `gorm:"size:20;not null;index" json:"severity"`
	Subject     string     `gorm:"size:150;index" json:"subject"`
	TopicID     *uuid.UUID `gorm:"type:char(36);index" json:"topic_id"`
	Title       string     `gorm:"size:200;not null" json:"title"`
	Message     string     `gorm:"type:text;not null" json:"message"`
	Action      string     `gorm:"type:text" json:"action"`
	Priority    int        `gorm:"not null;default:0;index" json:"priority"`
	GeneratedAt time.Time  `gorm:"not null;index" json:"generated_at"`
}

type Invalidator interface {
	InvalidateStudent(ctx context.Context, studentID uuid.UUID) error
	InvalidateTeacher(ctx context.Context, teacherID uuid.UUID) error
	InvalidateClass(ctx context.Context, classID uuid.UUID) error
	InvalidateCourse(ctx context.Context, courseID uuid.UUID) error
}
