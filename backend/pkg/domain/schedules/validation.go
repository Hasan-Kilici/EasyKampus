package schedules

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (entry ScheduleEntry) Validate() error {
	if entry.ClassID == uuid.Nil || entry.CourseID == uuid.Nil || entry.TeacherID == uuid.Nil {
		return core.ErrValidation
	}
	term := strings.TrimSpace(entry.Term)
	if term == "" || len(term) > 32 {
		return core.ErrValidation
	}
	if entry.DayOfWeek < 0 || entry.DayOfWeek > 6 {
		return core.ErrValidation
	}
	if entry.StartMinute < 0 || entry.StartMinute >= entry.EndMinute || entry.EndMinute > minutesInDay {
		return core.ErrValidation
	}
	return nil
}

func (entry ScheduleEntry) StartText() string {
	return formatMinute(entry.StartMinute)
}

func (entry ScheduleEntry) EndText() string {
	return formatMinute(entry.EndMinute)
}

func formatMinute(minute int) string {
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}
