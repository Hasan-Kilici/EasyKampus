package schools

import (
	"strings"
	"unicode"

	"lms/backend/pkg/core"
)

func (school School) Validate() error {
	name := strings.TrimSpace(school.Name)
	if len(name) < 2 || len(name) > 200 {
		return core.ErrValidation
	}
	if len(school.Code) < 2 || len(school.Code) > 32 {
		return core.ErrValidation
	}
	for _, letter := range school.Code {
		if !unicode.IsUpper(letter) && !unicode.IsDigit(letter) && letter != '-' && letter != '_' {
			return core.ErrValidation
		}
	}
	return nil
}

func (role StaffRole) IsValid() bool {
	switch role {
	case StaffRoleDirector, StaffRoleRegistrar:
		return true
	default:
		return false
	}
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
