package lessons

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (lesson Lesson) Validate() error {
	if lesson.CourseID == uuid.Nil || lesson.CreatedByID == uuid.Nil {
		return core.ErrValidation
	}
	if strings.TrimSpace(lesson.Title) == "" || len(lesson.Title) > 200 {
		return core.ErrValidation
	}
	if len(lesson.Description) > 2000 {
		return core.ErrValidation
	}
	if len(lesson.MarkdownContent) > 400000 {
		return core.ErrValidation
	}
	if lesson.Position < 0 || lesson.Position > 10000 {
		return core.ErrValidation
	}
	if lesson.DurationMinutes < 0 || lesson.DurationMinutes > 600 {
		return core.ErrValidation
	}
	if !lesson.State.IsValid() {
		return core.ErrValidation
	}
	return nil
}
