package database

import (
	"context"

	"github.com/google/uuid"

	"gorm.io/gorm"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/grades"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/notes"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/schedules"
	"lms/backend/pkg/domain/topics"
)

type GradingRepository struct {
	db *gorm.DB
}

func NewGradingRepository(db *gorm.DB) *GradingRepository {
	return &GradingRepository{db: db}
}

func (repository *GradingRepository) QuestionsForGrading(ctx context.Context, quizID uuid.UUID) ([]attempts.GradedQuestion, error) {
	type questionRow struct {
		QuestionID      uuid.UUID
		TopicID         *uuid.UUID
		Points          float64
		CorrectOptionID *uuid.UUID
		CorrectCount    int
	}
	var rows []questionRow
	err := repository.db.WithContext(ctx).
		Table("questions").
		Select("questions.id AS question_id, questions.topic_id AS topic_id, questions.points AS points, MAX(CASE WHEN options.answer = ? THEN options.id END) AS correct_option_id, COALESCE(SUM(CASE WHEN options.answer = ? THEN 1 ELSE 0 END), 0) AS correct_count", true, true).
		Joins("LEFT JOIN options ON options.question_id = questions.id AND options.deleted_at IS NULL").
		Where("questions.quiz_id = ? AND questions.deleted_at IS NULL", quizID).
		Group("questions.id, questions.topic_id, questions.points").
		Order("questions.position ASC").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make([]attempts.GradedQuestion, 0, len(rows))
	for _, item := range rows {
		optionIDs, err := repository.optionIDs(ctx, item.QuestionID)
		if err != nil {
			return nil, err
		}
		result = append(result, attempts.GradedQuestion{
			QuestionID:      item.QuestionID,
			TopicID:         item.TopicID,
			Points:          item.Points,
			OptionIDs:       optionIDs,
			CorrectOptionID: item.CorrectOptionID,
			CorrectCount:    item.CorrectCount,
		})
	}
	return result, nil
}

func (repository *GradingRepository) QuestionGrade(ctx context.Context, quizID uuid.UUID, questionID uuid.UUID) (*attempts.GradedQuestion, error) {
	type questionRow struct {
		QuestionID      uuid.UUID
		TopicID         *uuid.UUID
		Points          float64
		CorrectOptionID *uuid.UUID
		CorrectCount    int
	}
	var found questionRow
	err := repository.db.WithContext(ctx).
		Table("questions").
		Select("questions.id AS question_id, questions.topic_id AS topic_id, questions.points AS points, MAX(CASE WHEN options.answer = ? THEN options.id END) AS correct_option_id, COALESCE(SUM(CASE WHEN options.answer = ? THEN 1 ELSE 0 END), 0) AS correct_count", true, true).
		Joins("LEFT JOIN options ON options.question_id = questions.id AND options.deleted_at IS NULL").
		Where("questions.id = ? AND questions.quiz_id = ? AND questions.deleted_at IS NULL", questionID, quizID).
		Group("questions.id, questions.topic_id, questions.points").
		Take(&found).Error
	if err != nil {
		return nil, translate(err)
	}
	optionIDs, err := repository.optionIDs(ctx, questionID)
	if err != nil {
		return nil, err
	}
	return &attempts.GradedQuestion{
		QuestionID:      found.QuestionID,
		TopicID:         found.TopicID,
		Points:          found.Points,
		OptionIDs:       optionIDs,
		CorrectOptionID: found.CorrectOptionID,
		CorrectCount:    found.CorrectCount,
	}, nil
}

func (repository *GradingRepository) optionIDs(ctx context.Context, questionID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&options.Option{}).
		Where("question_id = ?", questionID).
		Order("position ASC").
		Pluck("id", &ids).Error
	return ids, translate(err)
}

type QuizScopeRepository struct {
	db *gorm.DB
}

func NewQuizScopeRepository(db *gorm.DB) *QuizScopeRepository {
	return &QuizScopeRepository{db: db}
}

func (repository *QuizScopeRepository) GetByID(ctx context.Context, id uuid.UUID) (*attempts.QuizScope, error) {
	var quiz quizes.Quiz
	err := repository.db.WithContext(ctx).First(&quiz, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	scope := &attempts.QuizScope{
		ID:              quiz.ID,
		ClassID:         quiz.ClassID,
		CourseID:        quiz.CourseID,
		DurationMinutes: quiz.DurationMinutes,
		MaxAttempts:     quiz.MaxAttempts,
		OpenAt:          quiz.OpenAt,
		CloseAt:         quiz.CloseAt,
		State:           string(quiz.State),
	}
	return scope, nil
}

type ScopeRepository struct {
	db *gorm.DB
}

func NewScopeRepository(db *gorm.DB) *ScopeRepository {
	return &ScopeRepository{db: db}
}

func (repository *ScopeRepository) CourseIDForQuestion(ctx context.Context, questionID uuid.UUID) (uuid.UUID, error) {
	quizID, err := repository.QuestionQuiz(ctx, questionID)
	if err != nil {
		return uuid.Nil, err
	}
	return repository.CourseIDForQuiz(ctx, quizID)
}

func (repository *ScopeRepository) CourseIDForQuiz(ctx context.Context, quizID uuid.UUID) (uuid.UUID, error) {
	var quiz quizes.Quiz
	if err := repository.db.WithContext(ctx).First(&quiz, "id = ?", quizID).Error; err != nil {
		return uuid.Nil, translate(err)
	}
	return quiz.CourseID, nil
}

func (repository *ScopeRepository) QuizScope(ctx context.Context, quizID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	var quiz quizes.Quiz
	if err := repository.db.WithContext(ctx).First(&quiz, "id = ?", quizID).Error; err != nil {
		return uuid.Nil, uuid.Nil, translate(err)
	}
	return quiz.ClassID, quiz.CourseID, nil
}

func (repository *ScopeRepository) QuestionQuiz(ctx context.Context, questionID uuid.UUID) (uuid.UUID, error) {
	var question questions.Question
	if err := repository.db.WithContext(ctx).First(&question, "id = ?", questionID).Error; err != nil {
		return uuid.Nil, translate(err)
	}
	return question.QuizID, nil
}

func (repository *ScopeRepository) OptionQuestion(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error) {
	var option options.Option
	if err := repository.db.WithContext(ctx).First(&option, "id = ?", optionID).Error; err != nil {
		return uuid.Nil, translate(err)
	}
	return option.QuestionID, nil
}

func (repository *ScopeRepository) CourseIDForLesson(ctx context.Context, lessonID uuid.UUID) (uuid.UUID, error) {
	var lesson lessons.Lesson
	if err := repository.db.WithContext(ctx).First(&lesson, "id = ?", lessonID).Error; err != nil {
		return uuid.Nil, translate(err)
	}
	return lesson.CourseID, nil
}

func (repository *ScopeRepository) TopicCourse(ctx context.Context, topicID uuid.UUID) (uuid.UUID, error) {
	var topic topics.Topic
	if err := repository.db.WithContext(ctx).First(&topic, "id = ?", topicID).Error; err != nil {
		return uuid.Nil, translate(err)
	}
	return topic.CourseID, nil
}

func (repository *ScopeRepository) OwnerIDForResource(ctx context.Context, resource string, id uuid.UUID) (uuid.UUID, error) {
	switch resource {
	case "attendance":
		var record attendance.Attendance
		if err := repository.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return record.StudentID, nil
	case "grade":
		var grade grades.Grade
		if err := repository.db.WithContext(ctx).First(&grade, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return grade.StudentID, nil
	case "schedule":
		var entry schedules.ScheduleEntry
		if err := repository.db.WithContext(ctx).First(&entry, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return entry.TeacherID, nil
	case "attempt":
		var attempt attempts.QuizAttempt
		if err := repository.db.WithContext(ctx).First(&attempt, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return attempt.StudentID, nil
	case "note":
		var note notes.Note
		if err := repository.db.WithContext(ctx).First(&note, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return note.StudentID, nil
	default:
		return uuid.Nil, core.ErrNotFound
	}
}

func (repository *ScopeRepository) ClassIDForResource(ctx context.Context, resource string, id uuid.UUID) (uuid.UUID, error) {
	switch resource {
	case "attendance":
		var record attendance.Attendance
		if err := repository.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return record.ClassID, nil
	case "grade":
		var grade grades.Grade
		if err := repository.db.WithContext(ctx).First(&grade, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return grade.ClassID, nil
	case "schedule":
		var entry schedules.ScheduleEntry
		if err := repository.db.WithContext(ctx).First(&entry, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return entry.ClassID, nil
	case "attempt":
		var attempt attempts.QuizAttempt
		if err := repository.db.WithContext(ctx).First(&attempt, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		return attempt.ClassID, nil
	case "note":
		var note notes.Note
		if err := repository.db.WithContext(ctx).First(&note, "id = ?", id).Error; err != nil {
			return uuid.Nil, translate(err)
		}
		if note.ClassID == nil {
			return uuid.Nil, core.ErrNotFound
		}
		return *note.ClassID, nil
	default:
		return uuid.Nil, core.ErrNotFound
	}
}
