package attempts

import (
	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

const maxDurationSeconds = 86400

func (attempt QuizAttempt) Validate() error {
	if attempt.StudentID == uuid.Nil || attempt.QuizID == uuid.Nil || attempt.ClassID == uuid.Nil || attempt.CourseID == uuid.Nil {
		return core.ErrValidation
	}
	if !attempt.Status.IsValid() {
		return core.ErrValidation
	}
	if attempt.StartedAt.IsZero() {
		return core.ErrValidation
	}
	if attempt.Score < 0 || attempt.MaxScore < 0 || attempt.Score > attempt.MaxScore {
		return core.ErrValidation
	}
	if attempt.DurationSeconds < 0 || attempt.DurationSeconds > maxDurationSeconds {
		return core.ErrValidation
	}
	return nil
}

func (answer AttemptAnswer) Validate() error {
	if answer.AttemptID == uuid.Nil || answer.QuestionID == uuid.Nil {
		return core.ErrValidation
	}
	if answer.Points < 0 {
		return core.ErrValidation
	}
	if answer.Correct && answer.SelectedOptionID == nil {
		return core.ErrValidation
	}
	if answer.AnsweredAt.IsZero() {
		return core.ErrValidation
	}
	return nil
}

func (status Status) IsValid() bool {
	switch status {
	case StatusInProgress, StatusCompleted, StatusAbandoned:
		return true
	default:
		return false
	}
}

func (attempt QuizAttempt) Percentage() float64 {
	if attempt.MaxScore <= 0 {
		return 0
	}
	return round(attempt.Score * 100 / attempt.MaxScore)
}

func (attempt QuizAttempt) IsCompleted() bool { return attempt.Status == StatusCompleted }
