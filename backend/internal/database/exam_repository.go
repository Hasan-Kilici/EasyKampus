package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/exams"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
)

// ExamRepository, exams.Store'un GORM uygulamasıdır. Sınav + sorular + seçenekler
// tek bir veritabanı işleminde yazılır; hata olursa hiçbiri kalmaz.
type ExamRepository struct {
	db *gorm.DB
}

func NewExamRepository(db *gorm.DB) *ExamRepository {
	return &ExamRepository{db: db}
}

func (repository *ExamRepository) InTx(ctx context.Context, fn func(tx exams.Tx) error) error {
	return translate(repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&examTx{db: tx})
	}))
}

func (repository *ExamRepository) LoadDetail(ctx context.Context, quizID uuid.UUID) (*exams.Detail, error) {
	return loadExamDetail(repository.db.WithContext(ctx), quizID)
}

func (repository *ExamRepository) AttemptCount(ctx context.Context, quizID uuid.UUID) (int64, error) {
	return countQuizAttempts(repository.db.WithContext(ctx), quizID)
}

// QuestionStats, yalnızca tamamlanmış denemelerdeki boş olmayan cevapları sayar.
func (repository *ExamRepository) QuestionStats(ctx context.Context, quizID uuid.UUID) ([]exams.QuestionStat, error) {
	type row struct {
		QuestionID uuid.UUID
		Question   string
		Position   int
		Answered   int64
		Correct    int64
	}
	var rows []row
	err := repository.db.WithContext(ctx).Raw(`
SELECT questions.id AS question_id, questions.question AS question, questions.position AS position,
 COUNT(answers.question_id) AS answered,
 COALESCE(SUM(CASE WHEN answers.correct = 1 THEN 1 ELSE 0 END), 0) AS correct
FROM questions
LEFT JOIN (
 SELECT attempt_answers.question_id AS question_id, attempt_answers.correct AS correct
 FROM attempt_answers
 JOIN quiz_attempts ON quiz_attempts.id = attempt_answers.attempt_id
 WHERE quiz_attempts.quiz_id = ? AND quiz_attempts.status = 'completed' AND quiz_attempts.deleted_at IS NULL
  AND attempt_answers.deleted_at IS NULL AND attempt_answers.selected_option_id IS NOT NULL
) answers ON answers.question_id = questions.id
WHERE questions.quiz_id = ? AND questions.deleted_at IS NULL
GROUP BY questions.id, questions.question, questions.position
ORDER BY questions.position ASC`, quizID, quizID).Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	stats := make([]exams.QuestionStat, 0, len(rows))
	for _, item := range rows {
		stats = append(stats, exams.QuestionStat{QuestionID: item.QuestionID, Question: item.Question, Position: item.Position, Answered: item.Answered, Correct: item.Correct})
	}
	return stats, nil
}

type examTx struct {
	db *gorm.DB
}

func (tx *examTx) CreateQuiz(quiz *quizes.Quiz) error { return translate(tx.db.Create(quiz).Error) }

func (tx *examTx) SaveQuiz(quiz *quizes.Quiz) error { return translate(tx.db.Save(quiz).Error) }

func (tx *examTx) GetQuiz(id uuid.UUID) (*quizes.Quiz, error) {
	var quiz quizes.Quiz
	if err := tx.db.First(&quiz, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return &quiz, nil
}

func (tx *examTx) AttemptCount(quizID uuid.UUID) (int64, error) {
	return countQuizAttempts(tx.db, quizID)
}

// DeleteQuestions, sınavın sorularını ve seçeneklerini yumuşak siler (soft delete).
func (tx *examTx) DeleteQuestions(quizID uuid.UUID) error {
	var ids []uuid.UUID
	if err := tx.db.Model(&questions.Question{}).Where("quiz_id = ?", quizID).Pluck("id", &ids).Error; err != nil {
		return translate(err)
	}
	if len(ids) > 0 {
		if err := tx.db.Where("question_id IN ?", ids).Delete(&options.Option{}).Error; err != nil {
			return translate(err)
		}
	}
	return translate(tx.db.Where("quiz_id = ?", quizID).Delete(&questions.Question{}).Error)
}

func (tx *examTx) CreateQuestion(question *questions.Question) error {
	return translate(tx.db.Create(question).Error)
}

func (tx *examTx) CreateOption(option *options.Option) error {
	return translate(tx.db.Create(option).Error)
}

func (tx *examTx) SetQuestionsState(quizID uuid.UUID, review approvals.Review) error {
	return translate(tx.db.Model(&questions.Question{}).Where("quiz_id = ?", quizID).Updates(map[string]any{
		"state":          review.State,
		"reviewed_by_id": review.ReviewedByID,
		"reviewed_at":    review.ReviewedAt,
		"note":           review.Note,
	}).Error)
}

func (tx *examTx) LoadDetail(quizID uuid.UUID) (*exams.Detail, error) {
	return loadExamDetail(tx.db, quizID)
}

func countQuizAttempts(db *gorm.DB, quizID uuid.UUID) (int64, error) {
	var count int64
	err := db.Model(&attempts.QuizAttempt{}).Where("quiz_id = ?", quizID).Count(&count).Error
	return count, translate(err)
}

func loadExamDetail(db *gorm.DB, quizID uuid.UUID) (*exams.Detail, error) {
	var quiz quizes.Quiz
	if err := db.First(&quiz, "id = ?", quizID).Error; err != nil {
		return nil, translate(err)
	}
	var questionRows []questions.Question
	if err := db.Where("quiz_id = ?", quizID).Order("position ASC").Find(&questionRows).Error; err != nil {
		return nil, translate(err)
	}
	detail := &exams.Detail{Quiz: quiz, Questions: make([]exams.QuestionDetail, 0, len(questionRows))}
	if len(questionRows) == 0 {
		return detail, nil
	}
	ids := make([]uuid.UUID, 0, len(questionRows))
	for _, question := range questionRows {
		ids = append(ids, question.ID)
	}
	var optionRows []options.Option
	if err := db.Where("question_id IN ?", ids).Order("position ASC").Find(&optionRows).Error; err != nil {
		return nil, translate(err)
	}
	byQuestion := make(map[uuid.UUID][]options.Option, len(questionRows))
	for _, option := range optionRows {
		byQuestion[option.QuestionID] = append(byQuestion[option.QuestionID], option)
	}
	for _, question := range questionRows {
		items := byQuestion[question.ID]
		if items == nil {
			items = []options.Option{}
		}
		detail.Questions = append(detail.Questions, exams.QuestionDetail{Question: question, Options: items})
	}
	return detail, nil
}
