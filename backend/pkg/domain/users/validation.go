package users

import (
	"net/mail"
	"strings"

	"lms/backend/pkg/core"
)

func (user User) Validate() error {
	if strings.TrimSpace(user.Name) == "" || strings.TrimSpace(user.Surname) == "" {
		return core.ErrValidation
	}
	if _, err := mail.ParseAddress(user.Email); err != nil || strings.TrimSpace(user.Email) != user.Email {
		return core.ErrValidation
	}
	if user.Age < 0 || user.Age > 150 {
		return core.ErrValidation
	}
	if !user.Role.IsValid() {
		return core.ErrValidation
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return core.ErrValidation
	}
	return nil
}

func (role Role) IsValid() bool {
	switch role {
	case RoleSuperAdmin, RoleAdmin, RoleTeacher, RoleStudent:
		return true
	default:
		return false
	}
}
