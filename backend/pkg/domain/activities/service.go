package activities

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
)

const dateLayout = "2006-01-02"

const maxBatchSize = 50

type EventInput struct {
	Type            EventType  `json:"type"`
	ClassID         string     `json:"class_id"`
	CourseID        string     `json:"course_id"`
	LessonID        string     `json:"lesson_id"`
	QuizID          string     `json:"quiz_id"`
	DurationSeconds int        `json:"duration_seconds"`
	OccurredAt      *time.Time `json:"occurred_at"`
	Metadata        string     `json:"metadata"`
}

type Service interface {
	Record(ctx context.Context, studentID uuid.UUID, events []EventInput) error
	RecordLogin(ctx context.Context, studentID uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Activity, error)
	ListDaily(ctx context.Context, params DailyParams) ([]DailyAggregate, error)
}

type service struct {
	repository  Repository
	invalidator performance.Invalidator
}

func NewService(repository Repository, invalidator performance.Invalidator) Service {
	return &service{repository: repository, invalidator: invalidator}
}

func (service *service) Record(ctx context.Context, studentID uuid.UUID, events []EventInput) error {
	if studentID == uuid.Nil || len(events) == 0 || len(events) > maxBatchSize {
		return core.ErrValidation
	}
	now := time.Now().UTC()
	records := make([]Activity, 0, len(events))
	aggregates := make([]DailyAggregate, 0, len(events))
	index := make(map[string]int, len(events))

	for _, input := range events {
		entity, err := buildEvent(studentID, input, now)
		if err != nil {
			return err
		}
		records = append(records, *entity)

		courseID := uuid.Nil
		if entity.CourseID != nil {
			courseID = *entity.CourseID
		}
		key := entity.OccurredAt.Format(dateLayout) + "|" + courseID.String()
		position, ok := index[key]
		if !ok {
			index[key] = len(aggregates)
			aggregates = append(aggregates, newAggregate(studentID, entity.OccurredAt.Format(dateLayout), courseID))
			position = len(aggregates) - 1
		}
		applyEvent(&aggregates[position], *entity)
	}

	if err := service.repository.RecordBatch(ctx, records, aggregates); err != nil {
		return err
	}
	if service.invalidator != nil {
		_ = service.invalidator.InvalidateStudent(ctx, studentID)
	}
	return nil
}

func (service *service) RecordLogin(ctx context.Context, studentID uuid.UUID) error {
	if studentID == uuid.Nil {
		return core.ErrValidation
	}
	return service.Record(ctx, studentID, []EventInput{{Type: EventLogin}})
}

func (service *service) List(ctx context.Context, params ListParams) ([]Activity, error) {
	if params.Type != "" && !params.Type.IsValid() {
		return nil, core.ErrValidation
	}
	if params.Limit <= 0 || params.Limit > 500 {
		params.Limit = 100
	}
	return service.repository.List(ctx, params)
}

func (service *service) ListDaily(ctx context.Context, params DailyParams) ([]DailyAggregate, error) {
	if params.StudentID == uuid.Nil {
		return nil, core.ErrValidation
	}
	if params.From != "" {
		if _, err := time.Parse(dateLayout, params.From); err != nil {
			return nil, core.ErrValidation
		}
	}
	if params.To != "" {
		if _, err := time.Parse(dateLayout, params.To); err != nil {
			return nil, core.ErrValidation
		}
	}
	return service.repository.ListDaily(ctx, params)
}

func buildEvent(studentID uuid.UUID, input EventInput, now time.Time) (*Activity, error) {
	if !input.Type.IsValid() {
		return nil, core.ErrValidation
	}
	if input.DurationSeconds < 0 || input.DurationSeconds > 86400 {
		return nil, core.ErrValidation
	}
	occurredAt := now
	if input.OccurredAt != nil {
		if input.OccurredAt.After(now.Add(5 * time.Minute)) {
			return nil, core.ErrValidation
		}
		occurredAt = input.OccurredAt.UTC()
	}
	event := &Activity{
		StudentID:       studentID,
		OccurredAt:      occurredAt,
		Type:            input.Type,
		DurationSeconds: input.DurationSeconds,
		Metadata:        strings.TrimSpace(input.Metadata),
	}
	if len(event.Metadata) > 255 {
		return nil, core.ErrValidation
	}
	var err error
	if event.ClassID, err = parseOptionalID(input.ClassID); err != nil {
		return nil, err
	}
	if event.CourseID, err = parseOptionalID(input.CourseID); err != nil {
		return nil, err
	}
	if event.LessonID, err = parseOptionalID(input.LessonID); err != nil {
		return nil, err
	}
	if event.QuizID, err = parseOptionalID(input.QuizID); err != nil {
		return nil, err
	}
	return event, nil
}

func newAggregate(studentID uuid.UUID, date string, courseID uuid.UUID) DailyAggregate {
	return DailyAggregate{StudentID: studentID, Date: date, CourseID: courseID}
}

func applyEvent(aggregate *DailyAggregate, event Activity) {
	aggregate.Events++
	switch event.Type {
	case EventLessonView:
		aggregate.LessonViews++
		aggregate.StudyMinutes += event.DurationSeconds / 60
	case EventLessonCompletion:
		aggregate.LessonCompletions++
	case EventQuizStarted:
		aggregate.QuizStarted++
	case EventQuizCompleted:
		aggregate.QuizCompleted++
	case EventQuestionAnswered:
		aggregate.QuestionAnswered++
	case EventCourseAccess:
		aggregate.CourseAccess++
	case EventLogin:
		aggregate.Logins++
	}
}

func (eventType EventType) IsValid() bool {
	for _, candidate := range eventTypes {
		if eventType == candidate {
			return true
		}
	}
	return false
}

func parseOptionalID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return nil, core.ErrValidation
	}
	return &parsed, nil
}
