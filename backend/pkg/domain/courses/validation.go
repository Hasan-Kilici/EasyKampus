package courses

import (
	"math"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (course Course) Validate() error {
	if course.SchoolID == uuid.Nil {
		return core.ErrValidation
	}
	if len(strings.TrimSpace(course.Name)) < 2 || len(course.Name) > 150 {
		return core.ErrValidation
	}
	if len(course.Code) < 2 || len(course.Code) > 32 {
		return core.ErrValidation
	}
	for _, letter := range course.Code {
		if !unicode.IsUpper(letter) && !unicode.IsDigit(letter) && letter != '-' && letter != '_' {
			return core.ErrValidation
		}
	}
	if len(course.SubjectArea) > 100 || len(course.Description) > 4000 {
		return core.ErrValidation
	}
	if len(course.Department) > 150 || len(course.DepartmentKeywords) > 2000 {
		return core.ErrValidation
	}
	for _, hours := range []float64{course.HomeworkHours, course.ProjectHours} {
		if math.IsNaN(hours) || hours < 0 || hours > MaxCourseHours {
			return core.ErrValidation
		}
	}
	return nil
}
