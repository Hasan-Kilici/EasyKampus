package campuses

import (
	"strings"
	"unicode"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (campus Campus) Validate() error {
	if campus.SchoolID == uuid.Nil {
		return core.ErrValidation
	}
	name := strings.TrimSpace(campus.Name)
	if len(name) < 2 || len(name) > 200 {
		return core.ErrValidation
	}
	if len(campus.Code) < 2 || len(campus.Code) > 32 {
		return core.ErrValidation
	}
	for _, letter := range campus.Code {
		if !unicode.IsUpper(letter) && !unicode.IsDigit(letter) && letter != '-' && letter != '_' {
			return core.ErrValidation
		}
	}
	return nil
}
