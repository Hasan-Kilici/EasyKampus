package attempts

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/performance"
)

type AnswerInput struct {
	QuestionID       string `json:"question_id"`
	SelectedOptionID string `json:"selected_option_id"`
}

type GradedQuestion struct {
	QuestionID      uuid.UUID
	TopicID         *uuid.UUID
	Points          float64
	OptionIDs       []uuid.UUID
	CorrectOptionID *uuid.UUID
	// CorrectCount, doğru işaretlenmiş seçenek sayısıdır; hazır bir sorunun tam 1 doğru seçeneği vardır.
	CorrectCount int
}

type QuizScope struct {
	ID              uuid.UUID
	ClassID         uuid.UUID
	CourseID        uuid.UUID
	DurationMinutes int
	MaxAttempts     int
	OpenAt          *time.Time
	CloseAt         *time.Time
	State           string
}

type Service interface {
	Start(ctx context.Context, studentID string, quizID string) (*QuizAttempt, error)
	Answer(ctx context.Context, attemptID string, input AnswerInput) (*AttemptAnswer, error)
	Submit(ctx context.Context, attemptID string) (*QuizAttempt, error)
	GetByID(ctx context.Context, id string) (*QuizAttempt, error)
	List(ctx context.Context, params ListParams) ([]QuizAttempt, error)
}

type QuizLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*QuizScope, error)
}

type GradingSource interface {
	QuestionsForGrading(ctx context.Context, quizID uuid.UUID) ([]GradedQuestion, error)
	QuestionGrade(ctx context.Context, quizID uuid.UUID, questionID uuid.UUID) (*GradedQuestion, error)
}

type EnrollmentLookup interface {
	Exists(ctx context.Context, classID uuid.UUID, studentID uuid.UUID, courseID *uuid.UUID) (bool, error)
}

type ActivityRecorder interface {
	Record(ctx context.Context, studentID uuid.UUID, events []activities.EventInput) error
}

type service struct {
	repository  Repository
	quizzes     QuizLookup
	grading     GradingSource
	enrollments EnrollmentLookup
	invalidator performance.Invalidator
	activity    ActivityRecorder
	now         func() time.Time
}

func NewService(repository Repository, quizzes QuizLookup, grading GradingSource, enrollments EnrollmentLookup, invalidator performance.Invalidator, recorders ...ActivityRecorder) Service {
	var activity ActivityRecorder
	if len(recorders) > 0 {
		activity = recorders[0]
	}
	return &service{repository: repository, quizzes: quizzes, grading: grading, enrollments: enrollments, invalidator: invalidator, activity: activity, now: func() time.Time { return time.Now().UTC() }}
}

func (service *service) Start(ctx context.Context, studentID string, quizID string) (*QuizAttempt, error) {
	parsedStudentID, err := parseID(studentID)
	if err != nil {
		return nil, core.ErrValidation
	}
	parsedQuizID, err := parseID(quizID)
	if err != nil {
		return nil, core.ErrValidation
	}
	quiz, err := service.quizzes.GetByID(ctx, parsedQuizID)
	if err != nil {
		return nil, err
	}
	if !quiz.IsPublished() {
		return nil, core.ErrForbidden
	}
	enrolled, err := service.enrollments.Exists(ctx, quiz.ClassID, parsedStudentID, &quiz.CourseID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, core.ErrForbidden
	}

	// Sürmekte olan deneme varsa yenisini açma: sayfa yenileme/ikinci sekme aynı
	// denemeye döner. Süresi dolmuşsa önce otomatik teslim edilir.
	active, err := service.repository.List(ctx, ListParams{StudentID: &parsedStudentID, QuizID: &parsedQuizID, Status: StatusInProgress, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(active) > 0 {
		attempt, err := service.repository.GetByID(ctx, active[0].ID)
		if err != nil {
			return nil, err
		}
		finalized, err := service.finalizeIfExpired(ctx, attempt, quiz)
		if err != nil {
			return nil, err
		}
		if !finalized {
			return attempt, nil
		}
	}

	now := service.now()
	if !quiz.isAvailableAt(now) {
		return nil, core.ErrForbidden
	}
	used, err := service.repository.CountByStudentAndQuiz(ctx, parsedStudentID, parsedQuizID, StatusCompleted)
	if err != nil {
		return nil, err
	}
	if int(used) >= quiz.MaxAttempts {
		return nil, core.ErrForbidden
	}
	if err := service.ensureReady(ctx, parsedQuizID); err != nil {
		return nil, err
	}
	attempt := &QuizAttempt{
		StudentID: parsedStudentID,
		QuizID:    parsedQuizID,
		ClassID:   quiz.ClassID,
		CourseID:  quiz.CourseID,
		Status:    StatusInProgress,
		StartedAt: now,
	}
	if err := service.repository.Create(ctx, attempt); err != nil {
		return nil, err
	}
	service.record(ctx, parsedStudentID, activities.EventInput{Type: activities.EventQuizStarted, ClassID: quiz.ClassID.String(), CourseID: quiz.CourseID.String(), QuizID: quiz.ID.String()})
	return attempt, nil
}

// ensureReady, yarım kalmış bir sınavın başlatılmasını engeller: en az bir soru
// olmalı ve her sorunun en az 2 seçeneği ile tam bir doğru seçeneği bulunmalı.
func (service *service) ensureReady(ctx context.Context, quizID uuid.UUID) error {
	graded, err := service.grading.QuestionsForGrading(ctx, quizID)
	if err != nil {
		return err
	}
	if len(graded) == 0 {
		return core.ErrValidation
	}
	for _, question := range graded {
		if len(question.OptionIDs) < 2 || question.CorrectCount != 1 {
			return core.ErrValidation
		}
	}
	return nil
}

func (service *service) Answer(ctx context.Context, attemptID string, input AnswerInput) (*AttemptAnswer, error) {
	parsedAttemptID, err := parseID(attemptID)
	if err != nil {
		return nil, core.ErrValidation
	}
	attempt, err := service.repository.GetByID(ctx, parsedAttemptID)
	if err != nil {
		return nil, err
	}
	if attempt.Status != StatusInProgress {
		return nil, core.ErrValidation
	}
	quiz, err := service.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	if finalized, err := service.finalizeIfExpired(ctx, attempt, quiz); err != nil {
		return nil, err
	} else if finalized {
		return nil, core.ErrForbidden
	}
	questionID, err := parseID(input.QuestionID)
	if err != nil {
		return nil, core.ErrValidation
	}
	selected, err := parseOptionalID(input.SelectedOptionID)
	if err != nil {
		return nil, core.ErrValidation
	}
	grade, err := service.grading.QuestionGrade(ctx, attempt.QuizID, questionID)
	if err != nil {
		return nil, err
	}
	if selected != nil && !grade.hasOption(*selected) {
		return nil, core.ErrValidation
	}
	correct := grade.isCorrect(selected)
	points := 0.0
	if correct {
		points = grade.Points
	}
	answer := &AttemptAnswer{
		AttemptID:        parsedAttemptID,
		QuestionID:       questionID,
		TopicID:          grade.TopicID,
		SelectedOptionID: selected,
		Correct:          correct,
		Points:           points,
		AnsweredAt:       service.now(),
	}
	if err := answer.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.SaveAnswers(ctx, []AttemptAnswer{*answer}); err != nil {
		return nil, err
	}
	return answer, nil
}

func (service *service) Submit(ctx context.Context, attemptID string) (*QuizAttempt, error) {
	parsedAttemptID, err := parseID(attemptID)
	if err != nil {
		return nil, core.ErrValidation
	}
	attempt, err := service.repository.GetByID(ctx, parsedAttemptID)
	if err != nil {
		return nil, err
	}
	// Süre dolunca deneme otomatik teslim edilmiş olabilir; tekrar teslim hata değil, sonucu döner.
	if attempt.Status == StatusCompleted {
		return attempt, nil
	}
	if attempt.Status != StatusInProgress {
		return nil, core.ErrValidation
	}
	quiz, err := service.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	at := service.now()
	if deadline, expired := quiz.deadline(attempt.StartedAt, at); expired {
		at = deadline // geç teslim, bitiş anına sabitlenir; süre dolduktan sonra cevap kaydedilemez
	}
	if err := service.complete(ctx, attempt, at); err != nil {
		return nil, err
	}
	return attempt, nil
}

// finalizeIfExpired, süresi dolmuş sürmekte olan denemeyi kayıtlı cevaplarıyla
// puanlayıp tamamlar (otomatik teslim). Vercel'de arka plan işi olmadığı için
// bu işlem tembel (lazy) olarak, denemeye dokunan her istekte yapılır.
func (service *service) finalizeIfExpired(ctx context.Context, attempt *QuizAttempt, quiz *QuizScope) (bool, error) {
	if attempt.Status != StatusInProgress {
		return false, nil
	}
	deadline, expired := quiz.deadline(attempt.StartedAt, service.now())
	if !expired {
		return false, nil
	}
	if err := service.complete(ctx, attempt, deadline); err != nil {
		return false, err
	}
	return true, nil
}

// complete, kayıtlı cevaplardan puanı hesaplar ve denemeyi `at` anında tamamlar.
func (service *service) complete(ctx context.Context, attempt *QuizAttempt, at time.Time) error {
	if attempt.Answers == nil {
		full, err := service.repository.GetByID(ctx, attempt.ID)
		if err != nil {
			return err
		}
		*attempt = *full
	}
	graded, err := service.grading.QuestionsForGrading(ctx, attempt.QuizID)
	if err != nil {
		return err
	}
	available := make(map[uuid.UUID]GradedQuestion, len(graded))
	total := 0.0
	for _, question := range graded {
		available[question.QuestionID] = question
		total += question.Points
	}
	score := 0.0
	for _, answer := range attempt.Answers {
		question, ok := available[answer.QuestionID]
		if !ok || !answer.Correct {
			continue
		}
		score += question.Points
	}
	if at.Before(attempt.StartedAt) {
		at = attempt.StartedAt
	}
	attempt.Status = StatusCompleted
	attempt.CompletedAt = &at
	attempt.Score = round(score)
	attempt.MaxScore = round(total)
	attempt.DurationSeconds = int(at.Sub(attempt.StartedAt).Seconds())
	if err := attempt.Validate(); err != nil {
		return err
	}
	if err := service.repository.Update(ctx, attempt); err != nil {
		return err
	}
	if service.invalidator != nil {
		_ = service.invalidator.InvalidateStudent(ctx, attempt.StudentID)
	}
	service.record(ctx, attempt.StudentID, activities.EventInput{Type: activities.EventQuizCompleted, ClassID: attempt.ClassID.String(), CourseID: attempt.CourseID.String(), QuizID: attempt.QuizID.String(), DurationSeconds: attempt.DurationSeconds})
	return nil
}

func (service *service) record(ctx context.Context, studentID uuid.UUID, event activities.EventInput) {
	if service.activity == nil {
		return
	}
	_ = service.activity.Record(ctx, studentID, []activities.EventInput{event})
}

func (service *service) GetByID(ctx context.Context, id string) (*QuizAttempt, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	attempt, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	if attempt.Status == StatusInProgress {
		quiz, err := service.quizzes.GetByID(ctx, attempt.QuizID)
		if err != nil {
			return nil, err
		}
		if _, err := service.finalizeIfExpired(ctx, attempt, quiz); err != nil {
			return nil, err
		}
	}
	return attempt, nil
}

func (service *service) List(ctx context.Context, params ListParams) ([]QuizAttempt, error) {
	if params.Limit <= 0 || params.Limit > 500 {
		params.Limit = 100
	}
	if params.Status != "" && !params.Status.IsValid() {
		return nil, core.ErrValidation
	}
	result, err := service.repository.List(ctx, params)
	if err != nil {
		return nil, err
	}
	// Listelenen denemelerden süresi dolanlar tembel olarak otomatik teslim edilir (en iyi çaba).
	quizzes := make(map[uuid.UUID]*QuizScope)
	for index := range result {
		if result[index].Status != StatusInProgress {
			continue
		}
		quiz, found := quizzes[result[index].QuizID]
		if !found {
			loaded, err := service.quizzes.GetByID(ctx, result[index].QuizID)
			if err != nil {
				continue
			}
			quiz = loaded
			quizzes[result[index].QuizID] = quiz
		}
		_, _ = service.finalizeIfExpired(ctx, &result[index], quiz)
	}
	return result, nil
}

func (grade GradedQuestion) isCorrect(selected *uuid.UUID) bool {
	if selected == nil || grade.CorrectOptionID == nil || !grade.hasOption(*selected) {
		return false
	}
	return *selected == *grade.CorrectOptionID
}

func (grade GradedQuestion) hasOption(optionID uuid.UUID) bool {
	for _, candidate := range grade.OptionIDs {
		if candidate == optionID {
			return true
		}
	}
	return false
}

func (quiz *QuizScope) IsPublished() bool {
	return quiz.State == "approved"
}

func (quiz *QuizScope) isAvailableAt(now time.Time) bool {
	if quiz.OpenAt != nil && now.Before(*quiz.OpenAt) {
		return false
	}
	if quiz.CloseAt != nil && now.After(*quiz.CloseAt) {
		return false
	}
	return true
}

// deadline, denemenin bitiş anını ve `now` itibarıyla süresinin dolup dolmadığını
// döner. Bitiş, süre (başlangıç + dakika) ile kapanış zamanından erken olanıdır;
// ikisi de yoksa sınav süresizdir ve asla dolmaz. (Önceki sürümde süre 0 iken
// kapanış zamanı dikkate alınmıyordu.)
func (quiz *QuizScope) deadline(startedAt time.Time, now time.Time) (time.Time, bool) {
	var limit *time.Time
	if quiz.DurationMinutes > 0 {
		byDuration := startedAt.Add(time.Duration(quiz.DurationMinutes) * time.Minute)
		limit = &byDuration
	}
	if quiz.CloseAt != nil && (limit == nil || quiz.CloseAt.Before(*limit)) {
		limit = quiz.CloseAt
	}
	if limit == nil {
		return time.Time{}, false
	}
	return *limit, now.After(*limit)
}

func round(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }

func parseOptionalID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseID(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
