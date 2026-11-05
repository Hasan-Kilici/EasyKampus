package attempts

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/activities"
)

// ---- sahte bağımlılıklar --------------------------------------------------

type fakeRepository struct {
	attempts map[uuid.UUID]*QuizAttempt
	answers  map[uuid.UUID]map[uuid.UUID]AttemptAnswer
	creates  int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{attempts: map[uuid.UUID]*QuizAttempt{}, answers: map[uuid.UUID]map[uuid.UUID]AttemptAnswer{}}
}

func (repository *fakeRepository) Create(_ context.Context, attempt *QuizAttempt) error {
	for _, existing := range repository.attempts {
		// gerçek veritabanındaki benzersiz "tek aktif deneme" indeksi
		if existing.StudentID == attempt.StudentID && existing.QuizID == attempt.QuizID && existing.Status == StatusInProgress {
			return core.ErrAlreadyExists
		}
	}
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	stored := *attempt
	stored.Answers = nil
	repository.attempts[attempt.ID] = &stored
	repository.creates++
	return nil
}

func (repository *fakeRepository) GetByID(_ context.Context, id uuid.UUID) (*QuizAttempt, error) {
	stored, found := repository.attempts[id]
	if !found {
		return nil, core.ErrNotFound
	}
	copy := *stored
	copy.Answers = []AttemptAnswer{}
	for _, answer := range repository.answers[id] {
		copy.Answers = append(copy.Answers, answer)
	}
	return &copy, nil
}

func (repository *fakeRepository) Update(_ context.Context, attempt *QuizAttempt) error {
	stored := *attempt
	stored.Answers = nil
	repository.attempts[attempt.ID] = &stored
	return nil
}

func (repository *fakeRepository) SaveAnswers(_ context.Context, answers []AttemptAnswer) error {
	for _, answer := range answers {
		if repository.answers[answer.AttemptID] == nil {
			repository.answers[answer.AttemptID] = map[uuid.UUID]AttemptAnswer{}
		}
		repository.answers[answer.AttemptID][answer.QuestionID] = answer
	}
	return nil
}

func (repository *fakeRepository) List(_ context.Context, params ListParams) ([]QuizAttempt, error) {
	var result []QuizAttempt
	for _, attempt := range repository.attempts {
		if params.StudentID != nil && attempt.StudentID != *params.StudentID {
			continue
		}
		if params.QuizID != nil && attempt.QuizID != *params.QuizID {
			continue
		}
		if params.Status != "" && attempt.Status != params.Status {
			continue
		}
		copy := *attempt
		result = append(result, copy)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.After(result[j].StartedAt) })
	if params.Limit > 0 && len(result) > params.Limit {
		result = result[:params.Limit]
	}
	return result, nil
}

func (repository *fakeRepository) CountByStudentAndQuiz(_ context.Context, studentID uuid.UUID, quizID uuid.UUID, status Status) (int64, error) {
	var count int64
	for _, attempt := range repository.attempts {
		if attempt.StudentID == studentID && attempt.QuizID == quizID && attempt.Status == status {
			count++
		}
	}
	return count, nil
}

type fakeQuizzes struct{ quiz QuizScope }

func (source fakeQuizzes) GetByID(_ context.Context, id uuid.UUID) (*QuizScope, error) {
	if id != source.quiz.ID {
		return nil, core.ErrNotFound
	}
	copy := source.quiz
	return &copy, nil
}

type fakeGrading struct{ questions []GradedQuestion }

func (source fakeGrading) QuestionsForGrading(_ context.Context, _ uuid.UUID) ([]GradedQuestion, error) {
	return source.questions, nil
}

func (source fakeGrading) QuestionGrade(_ context.Context, _ uuid.UUID, questionID uuid.UUID) (*GradedQuestion, error) {
	for _, question := range source.questions {
		if question.QuestionID == questionID {
			copy := question
			return &copy, nil
		}
	}
	return nil, core.ErrNotFound
}

type fakeEnrollments struct{ enrolled bool }

func (source fakeEnrollments) Exists(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) (bool, error) {
	return source.enrolled, nil
}

type recorder struct{ events []activities.EventType }

func (r *recorder) Record(_ context.Context, _ uuid.UUID, events []activities.EventInput) error {
	for _, event := range events {
		r.events = append(r.events, event.Type)
	}
	return nil
}

// ---- test düzeneği --------------------------------------------------------

type fixture struct {
	service   *service
	repo      *fakeRepository
	clock     time.Time
	student   uuid.UUID
	quiz      QuizScope
	questions []GradedQuestion
	events    *recorder
}

// 3 soru: 1 + 1 + 2 puan = 4. Her sorunun ilk seçeneği doğrudur.
func newFixture(t *testing.T, mutate func(*fixture)) *fixture {
	t.Helper()
	f := &fixture{
		repo:    newFakeRepository(),
		clock:   time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC),
		student: uuid.New(),
		events:  &recorder{},
	}
	f.quiz = QuizScope{ID: uuid.New(), ClassID: uuid.New(), CourseID: uuid.New(), DurationMinutes: 30, MaxAttempts: 2, State: "approved"}
	for _, points := range []float64{1, 1, 2} {
		first, second := uuid.New(), uuid.New()
		f.questions = append(f.questions, GradedQuestion{QuestionID: uuid.New(), Points: points, OptionIDs: []uuid.UUID{first, second}, CorrectOptionID: &first, CorrectCount: 1})
	}
	if mutate != nil {
		mutate(f)
	}
	f.service = NewService(f.repo, fakeQuizzes{quiz: f.quiz}, fakeGrading{questions: f.questions}, fakeEnrollments{enrolled: true}, nil, f.events).(*service)
	f.service.now = func() time.Time { return f.clock }
	return f
}

func (f *fixture) rebuild() {
	f.service.quizzes = fakeQuizzes{quiz: f.quiz}
	f.service.grading = fakeGrading{questions: f.questions}
}

func (f *fixture) start(t *testing.T) *QuizAttempt {
	t.Helper()
	attempt, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return attempt
}

func (f *fixture) answer(t *testing.T, attempt *QuizAttempt, question int, option int) {
	t.Helper()
	_, err := f.service.Answer(context.Background(), attempt.ID.String(), AnswerInput{
		QuestionID:       f.questions[question].QuestionID.String(),
		SelectedOptionID: f.questions[question].OptionIDs[option].String(),
	})
	if err != nil {
		t.Fatalf("answer q%d: %v", question, err)
	}
}

// ---- testler --------------------------------------------------------------

func TestStartResumesTheActiveAttemptInsteadOfCreatingAnother(t *testing.T) {
	f := newFixture(t, nil)
	first := f.start(t)
	f.clock = f.clock.Add(5 * time.Minute)
	f.answer(t, first, 0, 0)

	second, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String())
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if second.ID != first.ID || f.repo.creates != 1 {
		t.Fatalf("expected the same attempt (creates=%d)", f.repo.creates)
	}
	if len(second.Answers) != 1 {
		t.Fatalf("resumed attempt should carry the saved answer, got %d", len(second.Answers))
	}
}

func TestSubmitScoresSavedAnswers(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0) // doğru (1)
	f.answer(t, attempt, 1, 1) // yanlış
	f.answer(t, attempt, 2, 0) // doğru (2)
	f.clock = f.clock.Add(10 * time.Minute)

	done, err := f.service.Submit(context.Background(), attempt.ID.String())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if done.Status != StatusCompleted || done.Score != 3 || done.MaxScore != 4 {
		t.Fatalf("status %s score %.1f/%.1f, want completed 3/4", done.Status, done.Score, done.MaxScore)
	}
	if done.DurationSeconds != 600 {
		t.Fatalf("duration %d, want 600", done.DurationSeconds)
	}
	if done.Percentage() != 75 {
		t.Fatalf("percentage %.1f, want 75", done.Percentage())
	}
}

func TestChangingAnAnswerReplacesTheEarlierOne(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 1) // yanlış
	f.answer(t, attempt, 0, 0) // düzeltildi
	done, _ := f.service.Submit(context.Background(), attempt.ID.String())
	if done.Score != 1 {
		t.Fatalf("score %.1f, want 1 (last answer counts)", done.Score)
	}
}

func TestClearingAnAnswerScoresZero(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0)
	if _, err := f.service.Answer(context.Background(), attempt.ID.String(), AnswerInput{QuestionID: f.questions[0].QuestionID.String()}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	done, _ := f.service.Submit(context.Background(), attempt.ID.String())
	if done.Score != 0 {
		t.Fatalf("score %.1f, want 0 after clearing", done.Score)
	}
}

func TestAnswerAfterDeadlineAutoSubmitsAndIsRejected(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0) // 1 puan, süre içinde
	f.clock = f.clock.Add(31 * time.Minute)

	_, err := f.service.Answer(context.Background(), attempt.ID.String(), AnswerInput{QuestionID: f.questions[1].QuestionID.String(), SelectedOptionID: f.questions[1].OptionIDs[0].String()})
	if !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("late answer: got %v, want ErrForbidden", err)
	}
	stored, _ := f.repo.GetByID(context.Background(), attempt.ID)
	if stored.Status != StatusCompleted || stored.Score != 1 {
		t.Fatalf("expired attempt should be auto-completed with the saved score, got %s %.1f", stored.Status, stored.Score)
	}
	if stored.CompletedAt == nil || !stored.CompletedAt.Equal(attempt.StartedAt.Add(30*time.Minute)) {
		t.Fatalf("completed_at should be the deadline, got %v", stored.CompletedAt)
	}
	if stored.DurationSeconds != 1800 {
		t.Fatalf("duration %d, want 1800", stored.DurationSeconds)
	}
}

func TestLateSubmitIsCappedAtTheDeadlineAndSubmitIsIdempotent(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 2, 0)
	f.clock = f.clock.Add(2 * time.Hour)

	done, err := f.service.Submit(context.Background(), attempt.ID.String())
	if err != nil {
		t.Fatalf("late submit: %v", err)
	}
	if done.DurationSeconds != 1800 || done.Score != 2 {
		t.Fatalf("duration %d score %.1f, want 1800 and 2", done.DurationSeconds, done.Score)
	}
	again, err := f.service.Submit(context.Background(), attempt.ID.String())
	if err != nil || again.Status != StatusCompleted {
		t.Fatalf("second submit should return the finished attempt, got %v %v", again, err)
	}
	completedEvents := 0
	for _, event := range f.events.events {
		if event == activities.EventQuizCompleted {
			completedEvents++
		}
	}
	if completedEvents != 1 {
		t.Fatalf("quiz_completed recorded %d times, want 1", completedEvents)
	}
}

func TestStartAfterExpiryFinalizesTheOldAttemptAndHonoursMaxAttempts(t *testing.T) {
	f := newFixture(t, nil)
	old := f.start(t)
	f.answer(t, old, 0, 0)
	f.clock = f.clock.Add(45 * time.Minute)

	fresh, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String())
	if err != nil {
		t.Fatalf("second attempt should be allowed (max 2): %v", err)
	}
	if fresh.ID == old.ID {
		t.Fatal("an expired attempt must not be resumed")
	}
	closed, _ := f.repo.GetByID(context.Background(), old.ID)
	if closed.Status != StatusCompleted {
		t.Fatalf("old attempt should have been auto-submitted, got %s", closed.Status)
	}

	// ikinci deneme de bitip üçüncü istek: hak kalmadı
	f.clock = f.clock.Add(time.Hour)
	if _, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String()); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("third attempt: got %v, want ErrForbidden", err)
	}
}

func TestStartRefusesExamsThatAreNotReady(t *testing.T) {
	cases := map[string]func(*fixture){
		"no questions":       func(f *fixture) { f.questions = nil },
		"no correct option":  func(f *fixture) { f.questions[1].CorrectOptionID, f.questions[1].CorrectCount = nil, 0 },
		"two correct":        func(f *fixture) { f.questions[1].CorrectCount = 2 },
		"single option only": func(f *fixture) { f.questions[1].OptionIDs = f.questions[1].OptionIDs[:1] },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, mutate)
			if _, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String()); !errors.Is(err, core.ErrValidation) {
				t.Fatalf("got %v, want ErrValidation", err)
			}
			if f.repo.creates != 0 {
				t.Fatal("no attempt must be created for an unready exam")
			}
		})
	}
}

func TestStartChecksPublicationWindowAndEnrollment(t *testing.T) {
	f := newFixture(t, nil)
	open := f.clock.Add(time.Hour)
	f.quiz.OpenAt = &open
	f.rebuild()
	if _, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String()); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("before open_at: got %v", err)
	}
	f.quiz.OpenAt = nil
	f.quiz.State = "draft"
	f.rebuild()
	if _, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String()); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("draft exam: got %v", err)
	}
	f.quiz.State = "approved"
	f.rebuild()
	f.service.enrollments = fakeEnrollments{enrolled: false}
	if _, err := f.service.Start(context.Background(), f.student.String(), f.quiz.ID.String()); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("not enrolled: got %v", err)
	}
}

func TestDeadlineUsesCloseAtEvenWhenTheExamIsUntimed(t *testing.T) {
	started := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	closeAt := started.Add(2 * time.Hour)

	untimed := QuizScope{DurationMinutes: 0, CloseAt: &closeAt}
	if _, expired := untimed.deadline(started, started.Add(time.Hour)); expired {
		t.Fatal("untimed exam must not expire before close_at")
	}
	if deadline, expired := untimed.deadline(started, started.Add(3*time.Hour)); !expired || !deadline.Equal(closeAt) {
		t.Fatalf("untimed exam must expire at close_at, got %v %v", deadline, expired)
	}

	timed := QuizScope{DurationMinutes: 30, CloseAt: &closeAt}
	if deadline, _ := timed.deadline(started, started); !deadline.Equal(started.Add(30 * time.Minute)) {
		t.Fatalf("the shorter limit wins, got %v", deadline)
	}
	closingSoon := started.Add(10 * time.Minute)
	timed.CloseAt = &closingSoon
	if deadline, _ := timed.deadline(started, started); !deadline.Equal(closingSoon) {
		t.Fatalf("close_at earlier than the duration wins, got %v", deadline)
	}

	open := QuizScope{}
	if _, expired := open.deadline(started, started.Add(1000*time.Hour)); expired {
		t.Fatal("an exam with neither duration nor close_at never expires")
	}
}

func TestListAutoSubmitsExpiredAttempts(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0)
	f.clock = f.clock.Add(time.Hour)

	studentID := f.student
	items, err := f.service.List(context.Background(), ListParams{StudentID: &studentID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Status != StatusCompleted || items[0].Score != 1 {
		t.Fatalf("listing should auto-submit the expired attempt, got %+v", items)
	}
}

func TestViewHidesScoreAndCorrectnessWhileInProgress(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0)
	loaded, _ := f.service.GetByID(context.Background(), attempt.ID.String())

	view := loaded.View(true) // anahtar açık olsa bile sürerken gizli
	if view.Score != nil || view.MaxScore != nil {
		t.Fatal("score must stay hidden while the attempt is in progress")
	}
	if len(view.Answers) != 1 || view.Answers[0].Correct != nil || view.Answers[0].Points != nil {
		t.Fatal("correctness must stay hidden while the attempt is in progress")
	}
	if view.Answers[0].SelectedOptionID == nil {
		t.Fatal("the student's own selection must be visible so the page can resume")
	}
}

func TestViewRevealsCorrectnessOnlyWhenAllowed(t *testing.T) {
	f := newFixture(t, nil)
	attempt := f.start(t)
	f.answer(t, attempt, 0, 0)
	done, _ := f.service.Submit(context.Background(), attempt.ID.String())

	hidden := done.View(false)
	if hidden.Score == nil || *hidden.Score != 1 {
		t.Fatal("the score is visible after submission")
	}
	if hidden.Answers[0].Correct != nil {
		t.Fatal("per-question correctness must stay hidden until reveal")
	}
	shown := done.View(true)
	if shown.Answers[0].Correct == nil || !*shown.Answers[0].Correct {
		t.Fatal("per-question correctness must be visible after reveal")
	}
}

func TestAnswerViewNeverLeaksCorrectnessUnlessRevealed(t *testing.T) {
	selected := uuid.New()
	answer := AttemptAnswer{QuestionID: uuid.New(), SelectedOptionID: &selected, Correct: true, Points: 3}
	if view := answer.View(false); view.Correct != nil || view.Points != nil || view.SelectedOptionID == nil {
		t.Fatalf("hidden view leaked or lost data: %+v", view)
	}
	if view := answer.View(true); view.Correct == nil || !*view.Correct || *view.Points != 3 {
		t.Fatalf("revealed view wrong: %+v", view)
	}
}
