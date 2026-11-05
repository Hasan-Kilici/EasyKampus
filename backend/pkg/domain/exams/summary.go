package exams

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/attempts"
)

// QuestionStat, bir sorunun tamamlanmış denemelerdeki başarısıdır.
type QuestionStat struct {
	QuestionID uuid.UUID `json:"question_id"`
	Question   string    `json:"question"`
	Position   int       `json:"position"`
	Answered   int64     `json:"answered"`
	Correct    int64     `json:"correct"`
	// Rate, doğru oranıdır (0–100); kimse cevaplamadıysa 0.
	Rate float64 `json:"rate"`
}

// StudentResult, bir öğrencinin tamamlanmış denemelerinin özetidir.
type StudentResult struct {
	StudentID       uuid.UUID  `json:"student_id"`
	Attempts        int        `json:"attempts"`
	BestScore       float64    `json:"best_score"`
	LastScore       float64    `json:"last_score"`
	MaxScore        float64    `json:"max_score"`
	BestPercentage  float64    `json:"best_percentage"`
	LastCompletedAt *time.Time `json:"last_completed_at"`
}

// Summary, öğretmenin göreceği sınav özetidir.
type Summary struct {
	QuizID       uuid.UUID       `json:"quiz_id"`
	Participants int             `json:"participants"`
	Average      float64         `json:"average_percentage"`
	Highest      float64         `json:"highest_percentage"`
	Lowest       float64         `json:"lowest_percentage"`
	Students     []StudentResult `json:"students"`
	Questions    []QuestionStat  `json:"questions"`
}

// Summarize, tamamlanmış denemelerden öğrenci bazlı özet üretir. Ortalama,
// öğrencilerin en iyi yüzdelerinin ortalamasıdır.
func Summarize(quizID uuid.UUID, completed []attempts.QuizAttempt, stats []QuestionStat) Summary {
	byStudent := make(map[uuid.UUID]*StudentResult)
	for _, attempt := range completed {
		if attempt.Status != attempts.StatusCompleted {
			continue
		}
		entry, found := byStudent[attempt.StudentID]
		if !found {
			entry = &StudentResult{StudentID: attempt.StudentID}
			byStudent[attempt.StudentID] = entry
		}
		entry.Attempts++
		percentage := attempt.Percentage()
		if percentage > entry.BestPercentage || entry.Attempts == 1 {
			entry.BestPercentage = percentage
			entry.BestScore = attempt.Score
		}
		if entry.LastCompletedAt == nil || (attempt.CompletedAt != nil && attempt.CompletedAt.After(*entry.LastCompletedAt)) {
			entry.LastCompletedAt = attempt.CompletedAt
			entry.LastScore = attempt.Score
		}
		if attempt.MaxScore > entry.MaxScore {
			entry.MaxScore = attempt.MaxScore
		}
	}

	summary := Summary{QuizID: quizID, Students: make([]StudentResult, 0, len(byStudent)), Questions: make([]QuestionStat, 0, len(stats))}
	var total float64
	for _, entry := range byStudent {
		summary.Students = append(summary.Students, *entry)
		total += entry.BestPercentage
		if summary.Participants == 0 || entry.BestPercentage > summary.Highest {
			summary.Highest = entry.BestPercentage
		}
		if summary.Participants == 0 || entry.BestPercentage < summary.Lowest {
			summary.Lowest = entry.BestPercentage
		}
		summary.Participants++
	}
	sort.Slice(summary.Students, func(i, j int) bool {
		if summary.Students[i].BestPercentage != summary.Students[j].BestPercentage {
			return summary.Students[i].BestPercentage > summary.Students[j].BestPercentage
		}
		return summary.Students[i].StudentID.String() < summary.Students[j].StudentID.String()
	})
	if summary.Participants > 0 {
		summary.Average = round1(total / float64(summary.Participants))
	}
	for _, stat := range stats {
		if stat.Answered > 0 {
			stat.Rate = round1(float64(stat.Correct) * 100 / float64(stat.Answered))
		}
		summary.Questions = append(summary.Questions, stat)
	}
	sort.Slice(summary.Questions, func(i, j int) bool { return summary.Questions[i].Position < summary.Questions[j].Position })
	return summary
}

func round1(value float64) float64 { return float64(int64(value*10+0.5)) / 10 }
