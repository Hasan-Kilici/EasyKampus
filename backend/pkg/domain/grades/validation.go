package grades

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

const (
	maxScore = 100
	minScore = 0
	maxTerm  = 32
	maxNote  = 255
)

func (grade Grade) Validate() error {
	if grade.StudentID == uuid.Nil || grade.ClassID == uuid.Nil || grade.CourseID == uuid.Nil {
		return core.ErrValidation
	}
	term := strings.TrimSpace(grade.Term)
	if term == "" || len(term) > maxTerm {
		return core.ErrValidation
	}
	if !grade.Type.IsValid() {
		return core.ErrValidation
	}
	if grade.Score < minScore || grade.Score > maxScore {
		return core.ErrValidation
	}
	if len(grade.Note) > maxNote {
		return core.ErrValidation
	}
	return nil
}

func (gradeType GradeType) IsValid() bool {
	switch gradeType {
	case GradeTypeExam, GradeTypeAssignment, GradeTypeProject, GradeTypeParticipation:
		return true
	default:
		return false
	}
}
