package attendance

import (
	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

func (record Attendance) Validate() error {
	if record.StudentID == uuid.Nil || record.ClassID == uuid.Nil || record.CourseID == uuid.Nil || record.TeacherID == uuid.Nil {
		return core.ErrValidation
	}
	if len(record.Date) != 10 {
		return core.ErrValidation
	}
	if !record.Status.IsValid() {
		return core.ErrValidation
	}
	if len(record.Note) > 255 {
		return core.ErrValidation
	}
	return nil
}

func (status Status) IsValid() bool {
	switch status {
	case StatusPresent, StatusAbsent, StatusLate, StatusExcused:
		return true
	default:
		return false
	}
}

func (status Status) IsAttended() bool {
	return status == StatusPresent || status == StatusLate
}

// AttendedRate, devam oranını 0-100 yüzde olarak döner; performans sözleşmesi
// tüm oran alanlarında yüzde ölçeği kullanır.
func (summary Summary) AttendedRate() float64 {
	if summary.Total <= 0 {
		return 0
	}
	attended := summary.Present + summary.Late
	return float64(int64(float64(attended)/float64(summary.Total)*10000+0.5)) / 100
}
