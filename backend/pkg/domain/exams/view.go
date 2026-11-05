package exams

import (
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
)

// QuestionDetail, cevap anahtarıyla birlikte bir sorudur (yalnızca personel görür).
type QuestionDetail struct {
	questions.Question
	Options []options.Option `json:"options"`
}

// Detail, bir sınavın personel görünümüdür.
type Detail struct {
	Quiz         quizes.Quiz      `json:"quiz"`
	Questions    []QuestionDetail `json:"questions"`
	Issues       []Issue          `json:"issues"`
	AttemptCount int64            `json:"attempt_count"`
	// QuestionsEditable: sınav çözülmeye başlandıysa sorular değiştirilemez
	// (geçmiş cevaplar soru kimliklerine bağlıdır).
	QuestionsEditable bool `json:"questions_editable"`
}

// Input, sınavı yeniden kaydedilebilir giriş biçimine çevirir.
func (detail Detail) Input() Input {
	in := Input{
		CourseID:        detail.Quiz.CourseID.String(),
		Name:            detail.Quiz.Name,
		Description:     detail.Quiz.Description,
		OpenAt:          detail.Quiz.OpenAt,
		CloseAt:         detail.Quiz.CloseAt,
		DurationMinutes: detail.Quiz.DurationMinutes,
		MaxAttempts:     detail.Quiz.MaxAttempts,
		Questions:       make([]QuestionInput, 0, len(detail.Questions)),
	}
	for _, question := range detail.Questions {
		item := QuestionInput{Question: question.Question.Question, Points: question.Points}
		if question.TopicID != nil {
			item.TopicID = question.TopicID.String()
		}
		for _, option := range question.Options {
			item.Options = append(item.Options, OptionInput{Content: option.Content, Answer: option.Answer})
		}
		in.Questions = append(in.Questions, item)
	}
	return in
}

// TakeOption ve TakeQuestion, öğrencinin gördüğü cevap anahtarsız görünümdür.
type TakeOption struct {
	ID       uuid.UUID `json:"id"`
	Content  string    `json:"content"`
	Position int       `json:"position"`
}

type TakeQuestion struct {
	ID       uuid.UUID    `json:"id"`
	Question string       `json:"question"`
	Points   float64      `json:"points"`
	Position int          `json:"position"`
	Options  []TakeOption `json:"options"`
}

// ForStudent, doğru cevap bayrağını ve konu bilgisini çıkarır.
func (detail Detail) ForStudent() []TakeQuestion {
	result := make([]TakeQuestion, 0, len(detail.Questions))
	for _, question := range detail.Questions {
		item := TakeQuestion{
			ID:       question.ID,
			Question: question.Question.Question,
			Points:   question.Points,
			Position: question.Position,
			Options:  make([]TakeOption, 0, len(question.Options)),
		}
		for _, option := range question.Options {
			item.Options = append(item.Options, TakeOption{ID: option.ID, Content: option.Content, Position: option.Position})
		}
		result = append(result, item)
	}
	return result
}

// ShouldReveal: doğru cevaplar yalnızca sınav kapandıysa ya da öğrencinin
// deneme hakkı bittiyse gösterilir; aksi halde bir sonraki denemede veya
// henüz çözmeyen arkadaşlara sızabilir.
func ShouldReveal(quiz quizes.Quiz, completedAttempts int, now time.Time) bool {
	if quiz.CloseAt != nil && !now.Before(*quiz.CloseAt) {
		return true
	}
	return quiz.MaxAttempts > 0 && completedAttempts >= quiz.MaxAttempts
}

// ResultOption, sonuç ekranındaki bir seçenektir. Correct yalnızca anahtar
// açıldığında (reveal) doludur.
type ResultOption struct {
	ID       uuid.UUID `json:"id"`
	Content  string    `json:"content"`
	Selected bool      `json:"selected"`
	Correct  *bool     `json:"correct,omitempty"`
}

type QuestionResult struct {
	QuestionID       uuid.UUID      `json:"question_id"`
	Question         string         `json:"question"`
	Points           float64        `json:"points"`
	Position         int            `json:"position"`
	Answered         bool           `json:"answered"`
	SelectedOptionID *uuid.UUID     `json:"selected_option_id"`
	Correct          *bool          `json:"correct,omitempty"`
	EarnedPoints     *float64       `json:"earned_points,omitempty"`
	Options          []ResultOption `json:"options"`
}

// Result, tamamlanmış bir denemenin sonucudur.
type Result struct {
	AttemptID       uuid.UUID        `json:"attempt_id"`
	QuizID          uuid.UUID        `json:"quiz_id"`
	QuizName        string           `json:"quiz_name"`
	Status          attempts.Status  `json:"status"`
	Score           float64          `json:"score"`
	MaxScore        float64          `json:"max_score"`
	Percentage      float64          `json:"percentage"`
	StartedAt       time.Time        `json:"started_at"`
	CompletedAt     *time.Time       `json:"completed_at"`
	DurationSeconds int              `json:"duration_seconds"`
	AnswersRevealed bool             `json:"answers_revealed"`
	Questions       []QuestionResult `json:"questions"`
}

// BuildResult, denemenin sonucunu üretir. reveal false ise doğru seçenek ve
// soru bazında doğruluk gizlenir; öğrencinin kendi seçimi her zaman görünür.
func BuildResult(attempt attempts.QuizAttempt, detail Detail, reveal bool) Result {
	selected := make(map[uuid.UUID]attempts.AttemptAnswer, len(attempt.Answers))
	for _, answer := range attempt.Answers {
		selected[answer.QuestionID] = answer
	}
	result := Result{
		AttemptID:       attempt.ID,
		QuizID:          attempt.QuizID,
		QuizName:        detail.Quiz.Name,
		Status:          attempt.Status,
		Score:           attempt.Score,
		MaxScore:        attempt.MaxScore,
		Percentage:      attempt.Percentage(),
		StartedAt:       attempt.StartedAt,
		CompletedAt:     attempt.CompletedAt,
		DurationSeconds: attempt.DurationSeconds,
		AnswersRevealed: reveal,
		Questions:       make([]QuestionResult, 0, len(detail.Questions)),
	}
	for _, question := range detail.Questions {
		item := QuestionResult{
			QuestionID: question.ID,
			Question:   question.Question.Question,
			Points:     question.Points,
			Position:   question.Position,
			Options:    make([]ResultOption, 0, len(question.Options)),
		}
		answer, answered := selected[question.ID]
		if answered && answer.SelectedOptionID != nil {
			item.Answered = true
			item.SelectedOptionID = answer.SelectedOptionID
		}
		if reveal {
			correct := item.Answered && answer.Correct
			earned := 0.0
			if correct {
				earned = question.Points
			}
			item.Correct = &correct
			item.EarnedPoints = &earned
		}
		for _, option := range question.Options {
			entry := ResultOption{ID: option.ID, Content: option.Content}
			if item.SelectedOptionID != nil && *item.SelectedOptionID == option.ID {
				entry.Selected = true
			}
			if reveal {
				isCorrect := option.Answer
				entry.Correct = &isCorrect
			}
			item.Options = append(item.Options, entry)
		}
		result.Questions = append(result.Questions, item)
	}
	return result
}
