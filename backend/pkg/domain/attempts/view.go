package attempts

import (
	"time"

	"github.com/google/uuid"
)

// AnswerView ve View, öğrenciye dönen güvenli deneme görünümüdür. Sınav sürerken
// puan ve doğruluk bilgisi verilmez: aksi halde öğrenci bir seçeneği işaretleyip
// anında doğru olup olmadığını öğrenebilir.
type AnswerView struct {
	QuestionID       uuid.UUID  `json:"question_id"`
	SelectedOptionID *uuid.UUID `json:"selected_option_id"`
	AnsweredAt       time.Time  `json:"answered_at"`
	Correct          *bool      `json:"correct,omitempty"`
	Points           *float64   `json:"points,omitempty"`
}

type View struct {
	ID              uuid.UUID    `json:"id"`
	StudentID       uuid.UUID    `json:"student_id"`
	QuizID          uuid.UUID    `json:"quiz_id"`
	ClassID         uuid.UUID    `json:"class_id"`
	CourseID        uuid.UUID    `json:"course_id"`
	Status          Status       `json:"status"`
	StartedAt       time.Time    `json:"started_at"`
	CompletedAt     *time.Time   `json:"completed_at"`
	Score           *float64     `json:"score,omitempty"`
	MaxScore        *float64     `json:"max_score,omitempty"`
	DurationSeconds int          `json:"duration_seconds"`
	Answers         []AnswerView `json:"answers"`
}

// View, öğrenci için güvenli görünümü üretir. Sürmekte olan denemede puan ve
// doğruluk gizlenir; tamamlanmışta puan görünür, soru bazında doğruluk yalnızca
// revealKeys true ise (bkz. exams.ShouldReveal) görünür.
func (attempt QuizAttempt) View(revealKeys bool) View {
	view := View{
		ID:              attempt.ID,
		StudentID:       attempt.StudentID,
		QuizID:          attempt.QuizID,
		ClassID:         attempt.ClassID,
		CourseID:        attempt.CourseID,
		Status:          attempt.Status,
		StartedAt:       attempt.StartedAt,
		CompletedAt:     attempt.CompletedAt,
		DurationSeconds: attempt.DurationSeconds,
		Answers:         make([]AnswerView, 0, len(attempt.Answers)),
	}
	completed := attempt.Status == StatusCompleted
	if completed {
		score, max := attempt.Score, attempt.MaxScore
		view.Score, view.MaxScore = &score, &max
	}
	for _, answer := range attempt.Answers {
		view.Answers = append(view.Answers, answer.View(completed && revealKeys))
	}
	return view
}

// View, tek bir cevabın güvenli görünümüdür; doğruluk ve puan yalnızca reveal ise döner.
func (answer AttemptAnswer) View(reveal bool) AnswerView {
	item := AnswerView{QuestionID: answer.QuestionID, SelectedOptionID: answer.SelectedOptionID, AnsweredAt: answer.AnsweredAt}
	if reveal {
		correct, points := answer.Correct, answer.Points
		item.Correct, item.Points = &correct, &points
	}
	return item
}
