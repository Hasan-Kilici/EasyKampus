package questions

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (question Question) Validate() error {
	if question.QuizID == uuid.Nil || question.CreatedByID == uuid.Nil {
		return core.ErrValidation
	}
	if strings.TrimSpace(question.Question) == "" {
		return core.ErrValidation
	}
	if question.Points < 0 || question.Points > 100 {
		return core.ErrValidation
	}
	if question.Position < 0 || question.Position > 10000 {
		return core.ErrValidation
	}
	if !question.State.IsValid() {
		return core.ErrValidation
	}
	return nil
}
