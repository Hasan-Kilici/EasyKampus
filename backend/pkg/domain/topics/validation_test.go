package topics

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func validTopic() Topic {
	return Topic{
		CourseID:    uuid.New(),
		Name:        "Değişkenler",
		Code:        "degiskenler",
		Weight:      1,
		Description: "Programlamada değişken tanımlama ve veri tipleri anlatılır.",
		StudyHours:  8,
	}
}

func TestTopicRequiresADescription(t *testing.T) {
	if err := validTopic().Validate(); err != nil {
		t.Fatalf("valid topic rejected: %v", err)
	}
	for _, description := range []string{"", "   ", "kısa", strings.Repeat("a", MaxDescriptionLength+1)} {
		topic := validTopic()
		topic.Description = description
		if err := topic.Validate(); !errors.Is(err, core.ErrValidation) {
			t.Errorf("description %q should be rejected, got %v", description, err)
		}
	}
}

func TestTopicStudyHoursBounds(t *testing.T) {
	for _, hours := range []float64{-1, MaxStudyHours + 1} {
		topic := validTopic()
		topic.StudyHours = hours
		if err := topic.Validate(); !errors.Is(err, core.ErrValidation) {
			t.Errorf("study hours %.1f should be rejected, got %v", hours, err)
		}
	}
	for _, hours := range []float64{0, 2.5, MaxStudyHours} {
		topic := validTopic()
		topic.StudyHours = hours
		if err := topic.Validate(); err != nil {
			t.Errorf("study hours %.1f should be accepted, got %v", hours, err)
		}
	}
}
