package notes

import (
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (note Note) Validate() error {
	if note.AuthorID == uuid.Nil || note.StudentID == uuid.Nil {
		return core.ErrValidation
	}
	body := strings.TrimSpace(note.Body)
	if body == "" || len(body) > maxBodyLength {
		return core.ErrValidation
	}
	if note.CourseID != nil && note.ClassID == nil {
		return core.ErrValidation
	}
	return nil
}

func (note Note) IsShared() bool { return note.VisibleToStudent }
