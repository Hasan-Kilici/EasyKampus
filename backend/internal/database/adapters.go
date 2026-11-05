package database

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/grades"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
)

// AttemptQuizSource, attempts alanının beklediği sınav kapsamı arayüzünü karşılar.
type AttemptQuizSource struct {
	repository *QuizRepository
}

func NewAttemptQuizSource(repository *QuizRepository) *AttemptQuizSource {
	return &AttemptQuizSource{repository: repository}
}

func (source *AttemptQuizSource) GetByID(ctx context.Context, id uuid.UUID) (*attempts.QuizScope, error) {
	quiz, err := source.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &attempts.QuizScope{
		ID:              quiz.ID,
		ClassID:         quiz.ClassID,
		CourseID:        quiz.CourseID,
		DurationMinutes: quiz.DurationMinutes,
		MaxAttempts:     quiz.MaxAttempts,
		OpenAt:          quiz.OpenAt,
		CloseAt:         quiz.CloseAt,
		State:           string(quiz.State),
	}, nil
}

// QuestionQuizSource, questions alanının beklediği sınav kapsamı arayüzünü karşılar.
type QuestionQuizSource struct {
	repository *QuizRepository
}

func NewQuestionQuizSource(repository *QuizRepository) *QuestionQuizSource {
	return &QuestionQuizSource{repository: repository}
}

func (source *QuestionQuizSource) GetByID(ctx context.Context, id uuid.UUID) (*questions.QuizScope, error) {
	quiz, err := source.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &questions.QuizScope{ID: quiz.ID, ClassID: quiz.ClassID, CourseID: quiz.CourseID}, nil
}

// TopicCourseSource, soru alanının konu kapsamı çözümleyicisini karşılar.
type TopicCourseSource struct {
	repository *TopicRepository
}

func NewTopicCourseSource(repository *TopicRepository) *TopicCourseSource {
	return &TopicCourseSource{repository: repository}
}

func (source *TopicCourseSource) GetByID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	topic, err := source.repository.GetByID(ctx, id)
	if err != nil {
		return uuid.Nil, err
	}
	return topic.CourseID, nil
}

// EnrollmentScopeSource, attendance ve grades alanlarının beklediği kayıt kontrolünü dönem bağımsız karşılar.
type EnrollmentScopeSource struct {
	repository *EnrollmentRepository
}

func NewEnrollmentScopeSource(repository *EnrollmentRepository) *EnrollmentScopeSource {
	return &EnrollmentScopeSource{repository: repository}
}

func (source *EnrollmentScopeSource) Exists(ctx context.Context, classID uuid.UUID, studentID uuid.UUID, courseID *uuid.UUID) (bool, error) {
	return source.repository.Exists(ctx, classID, studentID, courseID, "")
}

var (
	_ attempts.QuizLookup         = (*AttemptQuizSource)(nil)
	_ questions.QuizLookup        = (*QuestionQuizSource)(nil)
	_ questions.TopicLookup       = (*TopicCourseSource)(nil)
	_ quizes.Repository           = (*QuizRepository)(nil)
	_ attendance.EnrollmentLookup = (*EnrollmentScopeSource)(nil)
	_ grades.EnrollmentLookup     = (*EnrollmentScopeSource)(nil)
	_ attempts.EnrollmentLookup   = (*EnrollmentScopeSource)(nil)
)
