package database

import (
	"context"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/attempts"
)

type AttemptRepository struct {
	db *gorm.DB
}

func NewAttemptRepository(db *gorm.DB) *AttemptRepository {
	return &AttemptRepository{db: db}
}

func (repository *AttemptRepository) Create(ctx context.Context, attempt *attempts.QuizAttempt) error {
	return translate(repository.db.WithContext(ctx).Omit("Answers").Create(attempt).Error)
}

func (repository *AttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*attempts.QuizAttempt, error) {
	var attempt attempts.QuizAttempt
	err := repository.db.WithContext(ctx).
		Preload("Answers").
		First(&attempt, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &attempt, nil
}

func (repository *AttemptRepository) Update(ctx context.Context, attempt *attempts.QuizAttempt) error {
	return translate(repository.db.WithContext(ctx).Omit("Answers").Save(attempt).Error)
}

func (repository *AttemptRepository) SaveAnswers(ctx context.Context, answers []attempts.AttemptAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	entities := make([]attempts.AttemptAnswer, 0, len(answers))
	for _, answer := range answers {
		answer.ID = uuid.New()
		entities = append(entities, answer)
	}
	return translate(repository.db.WithContext(ctx).
		Clauses(liveRowConflict(
			[]string{"attempt_id", "question_id"},
			clause.AssignmentColumns([]string{"topic_id", "selected_option_id", "correct", "points", "answered_at", "updated_at", "deleted_at"}),
		)).
		Create(&entities).Error)
}

func (repository *AttemptRepository) List(ctx context.Context, params attempts.ListParams) ([]attempts.QuizAttempt, error) {
	if params.Limit < 1 || params.Limit > 500 {
		params.Limit = 100
	}
	query := repository.db.WithContext(ctx).Model(&attempts.QuizAttempt{})
	if params.ScopeApplied && len(params.ClassIDs) == 0 {
		return []attempts.QuizAttempt{}, nil
	}
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.QuizID != nil {
		query = query.Where("quiz_id = ?", *params.QuizID)
	}
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if len(params.ClassIDs) > 0 {
		query = query.Where("class_id IN ?", params.ClassIDs)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	var result []attempts.QuizAttempt
	err := query.Order("started_at DESC").Limit(params.Limit).Find(&result).Error
	return result, translate(err)
}

func (repository *AttemptRepository) CountByStudentAndQuiz(ctx context.Context, studentID uuid.UUID, quizID uuid.UUID, status attempts.Status) (int64, error) {
	var count int64
	err := repository.db.WithContext(ctx).Model(&attempts.QuizAttempt{}).
		Where("student_id = ? AND quiz_id = ? AND status = ?", studentID, quizID, status).
		Count(&count).Error
	return count, translate(err)
}

func (repository *AttemptRepository) CompletedAttempts(ctx context.Context, studentID uuid.UUID) ([]attempts.QuizAttempt, error) {
	var result []attempts.QuizAttempt
	err := repository.db.WithContext(ctx).
		Where("student_id = ? AND status = ?", studentID, attempts.StatusCompleted).
		Order("completed_at DESC").
		Find(&result).Error
	return result, translate(err)
}
