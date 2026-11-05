package exams

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/users"
)

// ---- sahte kalıcılık -------------------------------------------------------

type fakeStore struct {
	quizzes   map[uuid.UUID]*quizes.Quiz
	questions []questions.Question
	options   []options.Option
	attempts  map[uuid.UUID]int64
	failOn    string // "question:2" gibi: o noktada hata ver
	creates   int
}

func newFakeStore() *fakeStore {
	return &fakeStore{quizzes: map[uuid.UUID]*quizes.Quiz{}, attempts: map[uuid.UUID]int64{}}
}

type fakeTx struct{ store *fakeStore }

func (store *fakeStore) snapshot() *fakeStore {
	clone := &fakeStore{quizzes: map[uuid.UUID]*quizes.Quiz{}, attempts: store.attempts, failOn: store.failOn, creates: store.creates}
	for id, quiz := range store.quizzes {
		copy := *quiz
		clone.quizzes[id] = &copy
	}
	clone.questions = append([]questions.Question(nil), store.questions...)
	clone.options = append([]options.Option(nil), store.options...)
	return clone
}

func (store *fakeStore) InTx(_ context.Context, fn func(tx Tx) error) error {
	before := store.snapshot()
	if err := fn(&fakeTx{store: store}); err != nil {
		store.quizzes, store.questions, store.options, store.creates = before.quizzes, before.questions, before.options, before.creates
		return err
	}
	return nil
}

func (store *fakeStore) detail(quizID uuid.UUID) (*Detail, error) {
	quiz, found := store.quizzes[quizID]
	if !found {
		return nil, core.ErrNotFound
	}
	detail := &Detail{Quiz: *quiz, Questions: []QuestionDetail{}}
	for _, question := range store.questions {
		if question.QuizID != quizID {
			continue
		}
		item := QuestionDetail{Question: question, Options: []options.Option{}}
		for _, option := range store.options {
			if option.QuestionID == question.ID {
				item.Options = append(item.Options, option)
			}
		}
		sort.Slice(item.Options, func(i, j int) bool { return item.Options[i].Position < item.Options[j].Position })
		detail.Questions = append(detail.Questions, item)
	}
	sort.Slice(detail.Questions, func(i, j int) bool { return detail.Questions[i].Position < detail.Questions[j].Position })
	return detail, nil
}

func (store *fakeStore) LoadDetail(_ context.Context, quizID uuid.UUID) (*Detail, error) {
	return store.detail(quizID)
}
func (store *fakeStore) AttemptCount(_ context.Context, quizID uuid.UUID) (int64, error) {
	return store.attempts[quizID], nil
}
func (store *fakeStore) QuestionStats(context.Context, uuid.UUID) ([]QuestionStat, error) {
	return nil, nil
}

func (tx *fakeTx) CreateQuiz(quiz *quizes.Quiz) error {
	if quiz.ID == uuid.Nil {
		quiz.ID = uuid.New()
	}
	copy := *quiz
	tx.store.quizzes[quiz.ID] = &copy
	return nil
}
func (tx *fakeTx) SaveQuiz(quiz *quizes.Quiz) error {
	copy := *quiz
	tx.store.quizzes[quiz.ID] = &copy
	return nil
}
func (tx *fakeTx) GetQuiz(id uuid.UUID) (*quizes.Quiz, error) {
	quiz, found := tx.store.quizzes[id]
	if !found {
		return nil, core.ErrNotFound
	}
	copy := *quiz
	return &copy, nil
}
func (tx *fakeTx) AttemptCount(quizID uuid.UUID) (int64, error) {
	return tx.store.attempts[quizID], nil
}
func (tx *fakeTx) DeleteQuestions(quizID uuid.UUID) error {
	var keptQuestions []questions.Question
	removed := map[uuid.UUID]bool{}
	for _, question := range tx.store.questions {
		if question.QuizID == quizID {
			removed[question.ID] = true
			continue
		}
		keptQuestions = append(keptQuestions, question)
	}
	var keptOptions []options.Option
	for _, option := range tx.store.options {
		if !removed[option.QuestionID] {
			keptOptions = append(keptOptions, option)
		}
	}
	tx.store.questions, tx.store.options = keptQuestions, keptOptions
	return nil
}
func (tx *fakeTx) CreateQuestion(question *questions.Question) error {
	tx.store.creates++
	if tx.store.failOn == "question:2" && tx.store.creates >= 2 {
		return errors.New("boom")
	}
	question.ID = uuid.New()
	tx.store.questions = append(tx.store.questions, *question)
	return nil
}
func (tx *fakeTx) CreateOption(option *options.Option) error {
	option.ID = uuid.New()
	tx.store.options = append(tx.store.options, *option)
	return nil
}
func (tx *fakeTx) SetQuestionsState(quizID uuid.UUID, review approvals.Review) error {
	for index := range tx.store.questions {
		if tx.store.questions[index].QuizID == quizID {
			tx.store.questions[index].State = review.State
		}
	}
	return nil
}
func (tx *fakeTx) LoadDetail(quizID uuid.UUID) (*Detail, error) { return tx.store.detail(quizID) }

type fakeClasses struct{ missing bool }

func (lookup fakeClasses) Exists(context.Context, uuid.UUID) (bool, error) {
	return !lookup.missing, nil
}

type fakeOffers struct{ offered bool }

func (lookup fakeOffers) ClassCourseExists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return lookup.offered, nil
}

type fakeTopics struct{ course uuid.UUID }

func (lookup fakeTopics) GetByID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return lookup.course, nil
}

type fakeAttempts struct{ completed []attempts.QuizAttempt }

func (fakeAttempts) Start(context.Context, string, string) (*attempts.QuizAttempt, error) {
	return nil, nil
}
func (fakeAttempts) Answer(context.Context, string, attempts.AnswerInput) (*attempts.AttemptAnswer, error) {
	return nil, nil
}
func (fakeAttempts) Submit(context.Context, string) (*attempts.QuizAttempt, error) { return nil, nil }
func (fakeAttempts) GetByID(context.Context, string) (*attempts.QuizAttempt, error) {
	return nil, nil
}
func (service fakeAttempts) List(_ context.Context, params attempts.ListParams) ([]attempts.QuizAttempt, error) {
	if params.Status == attempts.StatusCompleted {
		return service.completed, nil
	}
	return nil, nil
}

// ---- düzenek ---------------------------------------------------------------

type env struct {
	store   *fakeStore
	service *service
	class   uuid.UUID
	course  uuid.UUID
	teacher Actor
	admin   Actor
}

func newEnv(mutate func(*env)) *env {
	e := &env{store: newFakeStore(), class: uuid.New(), course: uuid.New()}
	e.teacher = Actor{ID: uuid.New(), Role: users.RoleTeacher}
	e.admin = Actor{ID: uuid.New(), Role: users.RoleAdmin}
	e.service = NewService(e.store, fakeClasses{}, fakeOffers{offered: true}, fakeTopics{course: e.course}, fakeAttempts{}, nil).(*service)
	e.service.now = func() time.Time { return time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC) }
	if mutate != nil {
		mutate(e)
	}
	return e
}

func validInput(course uuid.UUID) Input {
	return Input{
		CourseID: course.String(), Name: "Vize 1", DurationMinutes: 40, MaxAttempts: 1,
		Questions: []QuestionInput{
			{Question: "2+2?", Points: 2, Options: []OptionInput{{Content: "3"}, {Content: "4", Answer: true}, {Content: "5"}}},
			{Question: "Başkent?", Points: 3, Options: []OptionInput{{Content: "Ankara", Answer: true}, {Content: "İzmir"}}},
		},
	}
}

func (e *env) compose(t *testing.T, input Input) *Detail {
	t.Helper()
	detail, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), input)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	return detail
}

// ---- testler ---------------------------------------------------------------

func TestComposeStoresADraftWithOrderedQuestionsAndTotalPoints(t *testing.T) {
	e := newEnv(nil)
	detail := e.compose(t, validInput(e.course))

	if detail.Quiz.State != approvals.StateDraft || detail.Quiz.CreatedByID != e.teacher.ID {
		t.Fatalf("expected a draft created by the teacher, got %s / %s", detail.Quiz.State, detail.Quiz.CreatedByID)
	}
	if detail.Quiz.Points != 5 {
		t.Fatalf("total points %.1f, want 5", detail.Quiz.Points)
	}
	if len(detail.Questions) != 2 || detail.Questions[0].Position != 0 || detail.Questions[1].Position != 1 {
		t.Fatalf("questions should keep their order: %+v", detail.Questions)
	}
	if len(detail.Questions[0].Options) != 3 || detail.Questions[0].Options[1].Content != "4" || !detail.Questions[0].Options[1].Answer {
		t.Fatalf("options/answer key not stored: %+v", detail.Questions[0].Options)
	}
	if detail.Questions[0].State != approvals.StateDraft {
		t.Fatalf("questions follow the quiz state, got %s", detail.Questions[0].State)
	}
	if len(detail.Issues) != 0 || !detail.QuestionsEditable || detail.AttemptCount != 0 {
		t.Fatalf("a valid, unattempted exam: issues=%v editable=%v attempts=%d", detail.Issues, detail.QuestionsEditable, detail.AttemptCount)
	}
}

func TestDetailInputRoundTrips(t *testing.T) {
	e := newEnv(nil)
	input := validInput(e.course)
	detail := e.compose(t, input)
	back := detail.Input()
	if back.Name != input.Name || len(back.Questions) != 2 || back.Questions[1].Options[0].Content != "Ankara" || !back.Questions[1].Options[0].Answer {
		t.Fatalf("round trip lost data: %+v", back)
	}
	if issues := Check(back); len(issues) != 0 {
		t.Fatalf("round-tripped exam should still pass the strict check: %v", issues)
	}
}

func TestIncompleteDraftsAreAllowedButCannotBeSubmitted(t *testing.T) {
	e := newEnv(nil)
	input := validInput(e.course)
	input.Questions[1].Options = []OptionInput{{Content: "Ankara"}} // tek seçenek, doğru yok
	detail := e.compose(t, input)
	if len(detail.Issues) == 0 {
		t.Fatal("an incomplete draft should list what is missing")
	}

	_, err := e.service.Review(context.Background(), e.teacher, detail.Quiz.ID.String(), approvals.ActionSubmit, "")
	var validation *ValidationError
	if !errors.As(err, &validation) || !errors.Is(err, core.ErrValidation) || len(validation.Issues) == 0 {
		t.Fatalf("submit should fail with detailed issues, got %v", err)
	}
	if e.store.quizzes[detail.Quiz.ID].State != approvals.StateDraft {
		t.Fatal("a failed submit must leave the exam a draft")
	}
}

func TestComposeRejectsBadInputWithoutStoringAnything(t *testing.T) {
	e := newEnv(nil)
	input := validInput(e.course)
	input.DurationMinutes = -5
	_, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), input)
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Issues[0].Path != "duration_minutes" {
		t.Fatalf("expected a duration issue, got %v", err)
	}
	if len(e.store.quizzes) != 0 {
		t.Fatal("nothing should be stored for invalid input")
	}
}

func TestComposeChecksClassCourseOfferAndTopicOwnership(t *testing.T) {
	e := newEnv(func(e *env) { e.service.classes = fakeClasses{missing: true} })
	if _, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), validInput(e.course)); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing class: got %v", err)
	}
	e = newEnv(func(e *env) { e.service.offers = fakeOffers{offered: false} })
	if _, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), validInput(e.course)); !errors.Is(err, core.ErrValidation) {
		t.Fatalf("course not offered: got %v", err)
	}
	e = newEnv(func(e *env) { e.service.topics = fakeTopics{course: uuid.New()} }) // başka dersin konusu
	input := validInput(e.course)
	input.Questions[0].TopicID = uuid.New().String()
	if _, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), input); !errors.Is(err, core.ErrValidation) {
		t.Fatalf("topic from another course: got %v", err)
	}
}

func TestComposeIsAtomic(t *testing.T) {
	e := newEnv(func(e *env) { e.store.failOn = "question:2" })
	_, err := e.service.Compose(context.Background(), e.teacher, e.class.String(), validInput(e.course))
	if err == nil {
		t.Fatal("expected the injected failure")
	}
	if len(e.store.quizzes) != 0 || len(e.store.questions) != 0 || len(e.store.options) != 0 {
		t.Fatalf("a failed compose must roll everything back: %d quizzes %d questions %d options", len(e.store.quizzes), len(e.store.questions), len(e.store.options))
	}
}

func TestReviewWorkflowAndRoles(t *testing.T) {
	e := newEnv(nil)
	id := e.compose(t, validInput(e.course)).Quiz.ID.String()
	ctx := context.Background()

	if _, err := e.service.Review(ctx, e.teacher, id, approvals.ActionApprove, ""); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("a teacher must not approve: got %v", err)
	}
	if _, err := e.service.Review(ctx, e.teacher, id, approvals.ActionReject, ""); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("a teacher must not reject: got %v", err)
	}
	pending, err := e.service.Review(ctx, e.teacher, id, approvals.ActionSubmit, "")
	if err != nil || pending.Quiz.State != approvals.StatePending {
		t.Fatalf("teacher submit: %v %v", pending, err)
	}
	approved, err := e.service.Review(ctx, e.admin, id, approvals.ActionApprove, "uygun")
	if err != nil || approved.Quiz.State != approvals.StateApproved || approved.Quiz.ReviewedByID == nil || *approved.Quiz.ReviewedByID != e.admin.ID {
		t.Fatalf("admin approve: %v %v", approved, err)
	}
	for _, question := range approved.Questions {
		if question.State != approvals.StateApproved {
			t.Fatalf("questions must follow the quiz to approved, got %s", question.State)
		}
	}
	if len(approved.Issues) != 0 {
		t.Fatalf("an approved valid exam has no issues: %v", approved.Issues)
	}
}

func TestAdminCanApproveADraftDirectly(t *testing.T) {
	e := newEnv(nil)
	id := e.compose(t, validInput(e.course)).Quiz.ID.String()
	approved, err := e.service.Review(context.Background(), e.admin, id, approvals.ActionApprove, "")
	if err != nil || approved.Quiz.State != approvals.StateApproved {
		t.Fatalf("direct approval: %v %v", approved, err)
	}
}

func TestReplaceRewritesQuestionsAndReturnsToDraft(t *testing.T) {
	e := newEnv(nil)
	first := e.compose(t, validInput(e.course))
	id := first.Quiz.ID.String()
	if _, err := e.service.Review(context.Background(), e.admin, id, approvals.ActionApprove, ""); err != nil {
		t.Fatalf("approve: %v", err)
	}

	next := validInput(e.course)
	next.Name = "Vize 1 (düzeltildi)"
	next.Questions = next.Questions[:1]
	next.Questions[0].Points = 10
	replaced, err := e.service.Replace(context.Background(), e.teacher, id, next)
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if replaced.Quiz.Name != "Vize 1 (düzeltildi)" || len(replaced.Questions) != 1 || replaced.Quiz.Points != 10 {
		t.Fatalf("replace did not apply: %+v", replaced.Quiz)
	}
	if replaced.Quiz.State != approvals.StateDraft || replaced.Quiz.ReviewedByID != nil {
		t.Fatalf("an edited exam returns to draft and loses its review, got %s", replaced.Quiz.State)
	}
	if len(e.store.questions) != 1 || len(e.store.options) != 3 {
		t.Fatalf("old questions/options must be removed: %d questions %d options", len(e.store.questions), len(e.store.options))
	}
}

func TestReplaceIsRefusedOnceAttemptsExistAndCannotChangeTheCourse(t *testing.T) {
	e := newEnv(nil)
	id := e.compose(t, validInput(e.course)).Quiz.ID
	other := validInput(uuid.New())
	if _, err := e.service.Replace(context.Background(), e.teacher, id.String(), other); !errors.Is(err, core.ErrValidation) {
		t.Fatalf("changing the course: got %v", err)
	}
	e.store.attempts[id] = 3
	if _, err := e.service.Replace(context.Background(), e.teacher, id.String(), validInput(e.course)); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("editing after attempts: got %v", err)
	}
	detail, _ := e.service.Detail(context.Background(), id.String())
	if detail.QuestionsEditable || detail.AttemptCount != 3 {
		t.Fatalf("detail should report the lock: editable=%v attempts=%d", detail.QuestionsEditable, detail.AttemptCount)
	}
}

func TestResultsSummarizeCompletedAttempts(t *testing.T) {
	e := newEnv(nil)
	id := e.compose(t, validInput(e.course)).Quiz.ID
	s1, s2 := uuid.New(), uuid.New()
	done := func(student uuid.UUID, score float64, at time.Time) attempts.QuizAttempt {
		return attempts.QuizAttempt{StudentID: student, QuizID: id, Status: attempts.StatusCompleted, Score: score, MaxScore: 10, CompletedAt: &at}
	}
	t0 := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	e.service.attempts = fakeAttempts{completed: []attempts.QuizAttempt{done(s1, 4, t0), done(s1, 9, t0.Add(time.Hour)), done(s2, 6, t0)}}

	summary, err := e.service.Results(context.Background(), id.String())
	if err != nil {
		t.Fatalf("results: %v", err)
	}
	if summary.Participants != 2 || summary.Highest != 90 || summary.Lowest != 60 || summary.Average != 75 {
		t.Fatalf("summary %+v", summary)
	}
	if summary.Students[0].StudentID != s1 || summary.Students[0].Attempts != 2 || summary.Students[0].BestScore != 9 || summary.Students[0].LastScore != 9 {
		t.Fatalf("best-first ordering / attempt counts wrong: %+v", summary.Students)
	}
}

func TestSummarizeQuestionRates(t *testing.T) {
	q1, q2 := uuid.New(), uuid.New()
	summary := Summarize(uuid.New(), nil, []QuestionStat{
		{QuestionID: q2, Position: 1, Answered: 0},
		{QuestionID: q1, Position: 0, Answered: 8, Correct: 6},
	})
	if summary.Participants != 0 || summary.Average != 0 {
		t.Fatalf("empty summary expected, got %+v", summary)
	}
	if summary.Questions[0].QuestionID != q1 || summary.Questions[0].Rate != 75 || summary.Questions[1].Rate != 0 {
		t.Fatalf("question stats wrong: %+v", summary.Questions)
	}
}

func TestStudentViewHidesTheAnswerKey(t *testing.T) {
	e := newEnv(nil)
	detail := e.compose(t, validInput(e.course))
	body, err := json.Marshal(detail.ForStudent())
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{`"answer"`, `"topic_id"`, `"correct"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("student view must not contain %s: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "Ankara") || !strings.Contains(text, `"options"`) {
		t.Fatalf("student view should still carry the questions and options: %s", text)
	}
}

func TestBuildResultHidesOrRevealsTheKey(t *testing.T) {
	e := newEnv(nil)
	detail := e.compose(t, validInput(e.course))
	first, second := detail.Questions[0], detail.Questions[1]
	selected := first.Options[1].ID // doğru ("4")
	wrong := second.Options[1].ID   // yanlış ("İzmir")
	completedAt := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	attempt := attempts.QuizAttempt{
		Status: attempts.StatusCompleted, Score: 2, MaxScore: 5, StartedAt: completedAt.Add(-time.Minute), CompletedAt: &completedAt,
		Answers: []attempts.AttemptAnswer{
			{QuestionID: first.ID, SelectedOptionID: &selected, Correct: true, Points: 2},
			{QuestionID: second.ID, SelectedOptionID: &wrong, Correct: false},
		},
	}

	hidden := BuildResult(attempt, *detail, false)
	if hidden.AnswersRevealed || hidden.Percentage != 40 {
		t.Fatalf("hidden result: %+v", hidden)
	}
	body, _ := json.Marshal(hidden)
	if strings.Contains(string(body), `"correct"`) || strings.Contains(string(body), "earned_points") {
		t.Fatalf("a hidden result must not leak correctness: %s", body)
	}
	if !hidden.Questions[0].Answered || hidden.Questions[0].SelectedOptionID == nil || !hidden.Questions[0].Options[1].Selected {
		t.Fatalf("the student's own selection must stay visible: %+v", hidden.Questions[0])
	}

	shown := BuildResult(attempt, *detail, true)
	if shown.Questions[0].Correct == nil || !*shown.Questions[0].Correct || *shown.Questions[0].EarnedPoints != 2 {
		t.Fatalf("revealed result should mark the first question correct: %+v", shown.Questions[0])
	}
	if shown.Questions[1].Correct == nil || *shown.Questions[1].Correct || *shown.Questions[1].EarnedPoints != 0 {
		t.Fatalf("revealed result should mark the second question wrong: %+v", shown.Questions[1])
	}
	if shown.Questions[1].Options[0].Correct == nil || !*shown.Questions[1].Options[0].Correct {
		t.Fatal("the correct option must be marked once revealed")
	}
}

func TestShouldReveal(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	cases := []struct {
		name      string
		closeAt   *time.Time
		max       int
		completed int
		want      bool
	}{
		{"still open, attempts left", &future, 2, 1, false},
		{"closed", &past, 2, 1, true},
		{"no close time, attempts used up", nil, 1, 1, true},
		{"no close time, attempts left", nil, 3, 1, false},
	}
	for _, tc := range cases {
		quiz := quizes.Quiz{CloseAt: tc.closeAt, MaxAttempts: tc.max}
		if got := ShouldReveal(quiz, tc.completed, now); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCheckRules(t *testing.T) {
	good := validInput(uuid.New())
	if issues := Check(good); len(issues) != 0 {
		t.Fatalf("a valid exam has no issues: %v", issues)
	}
	mutations := map[string]func(*Input){
		"no name":          func(in *Input) { in.Name = "  " },
		"no questions":     func(in *Input) { in.Questions = nil },
		"blank question":   func(in *Input) { in.Questions[0].Question = " " },
		"one option":       func(in *Input) { in.Questions[0].Options = in.Questions[0].Options[:1] },
		"no correct":       func(in *Input) { in.Questions[0].Options[1].Answer = false },
		"two correct":      func(in *Input) { in.Questions[0].Options[0].Answer = true },
		"blank option":     func(in *Input) { in.Questions[0].Options[0].Content = "" },
		"duplicate option": func(in *Input) { in.Questions[0].Options[0].Content = "4" },
		"points too high":  func(in *Input) { in.Questions[0].Points = 101 },
		"bad attempts":     func(in *Input) { in.MaxAttempts = 0 },
		"bad duration":     func(in *Input) { in.DurationMinutes = 601 },
		"close before open": func(in *Input) {
			open, close := time.Now(), time.Now().Add(-time.Hour)
			in.OpenAt, in.CloseAt = &open, &close
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			in := validInput(uuid.New())
			mutate(&in)
			if len(Check(in)) == 0 {
				t.Fatal("expected at least one issue")
			}
		})
	}

	// Taslakta eksik içerik serbest, sınırlar değil.
	draft := validInput(uuid.New())
	draft.Questions[0].Options = nil
	draft.Questions[1].Question = ""
	if issues := CheckDraft(draft); len(issues) != 0 {
		t.Fatalf("drafts may be incomplete: %v", issues)
	}
	draft.Questions[0].Points = -1
	if len(CheckDraft(draft)) == 0 {
		t.Fatal("drafts must still reject out-of-range values")
	}
	var many Input
	many.Name, many.MaxAttempts = "x", 1
	many.Questions = make([]QuestionInput, MaxQuestions+1)
	if len(CheckDraft(many)) == 0 {
		t.Fatal("too many questions must be rejected")
	}
}

func TestIssueMessagesAreInTurkishAndPointToQuestions(t *testing.T) {
	in := validInput(uuid.New())
	in.Questions[1].Options[0].Answer = false
	issues := Check(in)
	if len(issues) != 1 || issues[0].Path != "questions[1].options" || !strings.Contains(issues[0].Message, "Soru 2") {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}
