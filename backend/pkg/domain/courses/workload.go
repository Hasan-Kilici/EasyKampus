package courses

import (
	"math"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/topics"
)

// EFOR / AKTS: bir dersin iş yükü; konuların, ödevin ve projenin saat toplamıdır.
// AKTS = toplam saat / (bir AKTS'nin saat karşılığı). Avrupa Kredi Transfer
// Sistemi'nde bir AKTS genellikle 25–30 saatlik iş yüküne denk gelir; varsayılan
// 30 saattir ve istek başına 25–30 aralığında değiştirilebilir.

const (
	DefaultHoursPerECTS = 30.0
	MinHoursPerECTS     = 25.0
	MaxHoursPerECTS     = 30.0
	// MaxCourseHours: ödev/proje saatleri için üst sınır (yanlış girişe karşı).
	MaxCourseHours = 1000.0
)

// TopicHours, tek bir konunun saat katkısıdır.
type TopicHours struct {
	TopicID string  `json:"topic_id"`
	Name    string  `json:"name"`
	Hours   float64 `json:"hours"`
}

// Workload, bir dersin EFOR özetidir.
type Workload struct {
	CourseID      string  `json:"course_id"`
	TopicHours    float64 `json:"topic_hours"`
	HomeworkHours float64 `json:"homework_hours"`
	ProjectHours  float64 `json:"project_hours"`
	TotalHours    float64 `json:"total_hours"`
	HoursPerECTS  float64 `json:"hours_per_ects"`
	// ECTS, en yakın 0,5'e yuvarlanmış AKTS'dir; ECTSExact ham değerdir.
	ECTS      float64      `json:"ects"`
	ECTSExact float64      `json:"ects_exact"`
	Topics    []TopicHours `json:"topics"`
}

// ComputeWorkload, yalnızca aktif konuların saatlerini toplar. hoursPerECTS 0
// ise varsayılan kullanılır; 25–30 dışındaki değerler ErrValidation döner.
func ComputeWorkload(course Course, courseTopics []topics.Topic, hoursPerECTS float64) (Workload, error) {
	if hoursPerECTS == 0 {
		hoursPerECTS = DefaultHoursPerECTS
	}
	if math.IsNaN(hoursPerECTS) || hoursPerECTS < MinHoursPerECTS || hoursPerECTS > MaxHoursPerECTS {
		return Workload{}, core.ErrValidation
	}
	workload := Workload{
		CourseID:      uuidString(course.ID),
		HomeworkHours: course.HomeworkHours,
		ProjectHours:  course.ProjectHours,
		HoursPerECTS:  hoursPerECTS,
		Topics:        make([]TopicHours, 0, len(courseTopics)),
	}
	for _, topic := range courseTopics {
		if !topic.Active {
			continue
		}
		workload.TopicHours += topic.StudyHours
		workload.Topics = append(workload.Topics, TopicHours{TopicID: uuidString(topic.ID), Name: topic.Name, Hours: topic.StudyHours})
	}
	workload.TotalHours = round2(workload.TopicHours + workload.HomeworkHours + workload.ProjectHours)
	workload.TopicHours = round2(workload.TopicHours)
	workload.ECTSExact = round2(workload.TotalHours / hoursPerECTS)
	workload.ECTS = math.Round(workload.TotalHours/hoursPerECTS*2) / 2
	return workload, nil
}

func round2(value float64) float64 { return math.Round(value*100) / 100 }

func uuidString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
