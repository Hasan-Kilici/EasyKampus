package topics

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

const (
	// MinDescriptionLength: konunun ne anlattığını yazmak zorunludur; çok kısa
	// açıklamalar TF-IDF puanı için anlamlı bir metin sağlamaz.
	MinDescriptionLength = 10
	MaxDescriptionLength = 4000
	MaxStudyHours        = 500.0
)

func (topic Topic) Validate() error {
	if topic.CourseID == uuid.Nil {
		return core.ErrValidation
	}
	if len(strings.TrimSpace(topic.Name)) < 2 || len(topic.Name) > 150 {
		return core.ErrValidation
	}
	if len(topic.Code) < 1 || len(topic.Code) > 32 {
		return core.ErrValidation
	}
	if topic.Weight < 0 || topic.Weight > 10 {
		return core.ErrValidation
	}
	description := strings.TrimSpace(topic.Description)
	if utf8.RuneCountInString(description) < MinDescriptionLength || utf8.RuneCountInString(description) > MaxDescriptionLength {
		return core.ErrValidation
	}
	if math.IsNaN(topic.StudyHours) || topic.StudyHours < 0 || topic.StudyHours > MaxStudyHours {
		return core.ErrValidation
	}
	return nil
}
