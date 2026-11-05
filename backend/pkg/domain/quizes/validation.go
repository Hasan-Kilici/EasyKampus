package quizes

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (quiz Quiz) Validate() error {
	if quiz.ClassID == uuid.Nil || quiz.CourseID == uuid.Nil || quiz.CreatedByID == uuid.Nil {
		return core.ErrValidation
	}
	if strings.TrimSpace(quiz.Name) == "" || len(quiz.Name) > 200 {
		return core.ErrValidation
	}
	if len(quiz.Description) > 2000 {
		return core.ErrValidation
	}
	if quiz.OpenAt != nil && quiz.CloseAt != nil && quiz.CloseAt.Before(*quiz.OpenAt) {
		return core.ErrValidation
	}
	if quiz.DurationMinutes < 0 || quiz.DurationMinutes > 600 {
		return core.ErrValidation
	}
	if quiz.Points < 0 || quiz.Points > 10000 {
		return core.ErrValidation
	}
	if quiz.MaxAttempts < 1 || quiz.MaxAttempts > 20 {
		return core.ErrValidation
	}
	if !quiz.State.IsValid() {
		return core.ErrValidation
	}
	return nil
}

func (quiz Quiz) IsOpenAt(now time.Time) bool {
	if quiz.OpenAt != nil && now.Before(*quiz.OpenAt) {
		return false
	}
	if quiz.CloseAt != nil && now.After(*quiz.CloseAt) {
		return false
	}
	return true
}

// Deadline, bir denemenin bitiş anını ve bir bitiş anı olup olmadığını döner:
// süre (başlangıç + dakika) ile kapanış zamanından erken olan geçerlidir; ikisi de
// yoksa sınav süresizdir. (Önceki sürüm süre 0 iken kapanış zamanını yok sayıyordu.)
func (quiz Quiz) Deadline(startedAt time.Time) (time.Time, bool) {
	var limit *time.Time
	if quiz.DurationMinutes > 0 {
		byDuration := startedAt.Add(time.Duration(quiz.DurationMinutes) * time.Minute)
		limit = &byDuration
	}
	if quiz.CloseAt != nil && (limit == nil || quiz.CloseAt.Before(*limit)) {
		limit = quiz.CloseAt
	}
	if limit == nil {
		return time.Time{}, false
	}
	return *limit, true
}
