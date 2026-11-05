package options

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (option Option) Validate() error {
	if option.QuestionID == uuid.Nil {
		return core.ErrValidation
	}
	if strings.TrimSpace(option.Content) == "" {
		return core.ErrValidation
	}
	if option.Position < 0 || option.Position > 1000 {
		return core.ErrValidation
	}
	return nil
}
