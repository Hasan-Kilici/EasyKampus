package activities

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type EventType string

const (
	EventLessonView          EventType = "lesson_view"           // Ders içeriği görüntülendi.
	EventLessonCompletion    EventType = "lesson_completion"     // Ders tamamlandı olarak işaretlendi.
	EventQuizStarted         EventType = "quiz_started"          // Sınav başlatıldı.
	EventQuizCompleted       EventType = "quiz_completed"        // Sınav tamamlandı.
	EventQuestionAnswered    EventType = "question_answered"     // Bir soru cevaplandı.
	EventAssignmentCompleted EventType = "assignment_completion" // Ödev tamamlandı.
	EventLogin               EventType = "login"                 // Oturum açıldı.
	EventCourseAccess        EventType = "course_access"         // Ders içeriğine erişildi.
)

var eventTypes = []EventType{
	EventLessonView, EventLessonCompletion, EventQuizStarted, EventQuizCompleted,
	EventQuestionAnswered, EventAssignmentCompleted, EventLogin, EventCourseAccess,
}

type Activity struct {
	base.Model
	StudentID       uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_activities_student_occurred,priority:1,where:deleted_at IS NULL" json:"student_id"`
	OccurredAt      time.Time  `gorm:"not null;uniqueIndex:idx_activities_student_occurred,priority:2,where:deleted_at IS NULL" json:"occurred_at"`
	Type            EventType  `gorm:"size:40;not null;index" json:"type"`
	ClassID         *uuid.UUID `gorm:"type:char(36);index" json:"class_id"`
	CourseID        *uuid.UUID `gorm:"type:char(36);index" json:"course_id"`
	LessonID        *uuid.UUID `gorm:"type:char(36);index" json:"lesson_id"`
	QuizID          *uuid.UUID `gorm:"type:char(36);index" json:"quiz_id"`
	DurationSeconds int        `gorm:"not null;default:0" json:"duration_seconds"`
	Metadata        string     `gorm:"size:255" json:"metadata"`
}

type DailyAggregate struct {
	base.Model
	StudentID         uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_activity_daily,priority:1,where:deleted_at IS NULL;index" json:"student_id"`
	Date              string    `gorm:"size:10;not null;uniqueIndex:idx_activity_daily,priority:2,where:deleted_at IS NULL;index" json:"date"`
	CourseID          uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_activity_daily,priority:3,where:deleted_at IS NULL" json:"course_id"`
	Events            int       `gorm:"not null;default:0" json:"events"`
	LessonViews       int       `gorm:"not null;default:0" json:"lesson_views"`
	LessonCompletions int       `gorm:"not null;default:0" json:"lesson_completions"`
	QuizStarted       int       `gorm:"not null;default:0" json:"quiz_started"`
	QuizCompleted     int       `gorm:"not null;default:0" json:"quiz_completed"`
	QuestionAnswered  int       `gorm:"not null;default:0" json:"question_answered"`
	CourseAccess      int       `gorm:"not null;default:0" json:"course_access"`
	StudyMinutes      int       `gorm:"not null;default:0" json:"study_minutes"`
	Logins            int       `gorm:"not null;default:0" json:"logins"`
}

type ListParams struct {
	StudentID *uuid.UUID `json:"student_id"`
	CourseID  *uuid.UUID `json:"course_id"`
	LessonID  *uuid.UUID `json:"lesson_id"`
	QuizID    *uuid.UUID `json:"quiz_id"`
	Type      EventType  `json:"type"`
	From      *time.Time `json:"from"`
	To        *time.Time `json:"to"`
	Limit     int        `json:"limit"`
}

type DailyParams struct {
	StudentID uuid.UUID  `json:"student_id"`
	CourseID  *uuid.UUID `json:"course_id"`
	From      string     `json:"from"`
	To        string     `json:"to"`
}

type Repository interface {
	InsertBatch(ctx context.Context, events []Activity) error
	List(ctx context.Context, params ListParams) ([]Activity, error)
	RecordBatch(ctx context.Context, events []Activity, aggregates []DailyAggregate) error
	ListDaily(ctx context.Context, params DailyParams) ([]DailyAggregate, error)
}
