package assignments

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (entity ClassCourse) Validate() error {
	if entity.ClassID == uuid.Nil || entity.CourseID == uuid.Nil {
		return core.ErrValidation
	}
	return validateTerm(entity.Term)
}

func (entity TeacherAssignment) Validate() error {
	if entity.TeacherID == uuid.Nil || entity.ClassID == uuid.Nil || entity.CourseID == uuid.Nil {
		return core.ErrValidation
	}
	return validateTerm(entity.Term)
}

func validateTerm(term string) error {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" || len(trimmed) > 32 {
		return core.ErrValidation
	}
	return nil
}
