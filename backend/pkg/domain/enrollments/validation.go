package enrollments

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (enrollment Enrollment) Validate() error {
	if enrollment.StudentID == uuid.Nil || enrollment.ClassID == uuid.Nil {
		return core.ErrValidation
	}
	term := strings.TrimSpace(enrollment.Term)
	if term == "" || len(term) > 32 {
		return core.ErrValidation
	}
	if !enrollment.Status.IsValid() {
		return core.ErrValidation
	}
	if len(enrollment.Note) > 2000 {
		return core.ErrValidation
	}
	return nil
}

func (status Status) IsValid() bool {
	switch status {
	case StatusActive, StatusWithdrawn, StatusCompleted:
		return true
	default:
		return false
	}
}
