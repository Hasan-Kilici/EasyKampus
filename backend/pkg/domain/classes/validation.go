package classes

import (
	"strings"
	"unicode"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (class Class) Validate() error {
	if class.SchoolID == uuid.Nil || class.CampusID == uuid.Nil {
		return core.ErrValidation
	}
	if strings.TrimSpace(class.Name) == "" || strings.TrimSpace(class.Level) == "" {
		return core.ErrValidation
	}
	if len(class.No) != 1 {
		return core.ErrValidation
	}
	letter := rune(class.No[0])
	if !unicode.IsUpper(letter) || letter < 'A' || letter > 'Z' {
		return core.ErrValidation
	}
	return nil
}
