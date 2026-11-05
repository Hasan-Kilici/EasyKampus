package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/lessons"
)

type ActivityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (repository *ActivityRepository) RecordBatch(ctx context.Context, events []activities.Activity, aggregates []activities.DailyAggregate) error {
	if len(events) == 0 {
		return nil
	}
	return translate(repository.transaction(ctx, func(tx *gorm.DB) error {
		entities := make([]activities.Activity, 0, len(events))
		completions := make([]lessons.LessonCompletion, 0, len(events))
		for _, event := range events {
			event.ID = uuid.New()
			event.OccurredAt = event.OccurredAt.UTC().Truncate(time.Second)
			entities = append(entities, event)
			if event.Type == activities.EventLessonCompletion && event.LessonID != nil && event.CourseID != nil {
				completions = append(completions, lessons.LessonCompletion{
					StudentID:   event.StudentID,
					LessonID:    *event.LessonID,
					CourseID:    *event.CourseID,
					CompletedAt: event.OccurredAt,
				})
			}
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:     []clause.Column{{Name: "student_id"}, {Name: "occurred_at"}},
			DoNothing:   true,
			TargetWhere: liveRowTarget(),
		}).Create(&entities).Error; err != nil {
			return err
		}
		if len(completions) > 0 {
			if err := upsertCompletions(tx, completions); err != nil {
				return err
			}
		}
		return upsertDaily(tx, aggregates)
	}))
}

func (repository *ActivityRepository) InsertBatch(ctx context.Context, events []activities.Activity) error {
	return repository.RecordBatch(ctx, events, nil)
}

func upsertCompletions(tx *gorm.DB, completions []lessons.LessonCompletion) error {
	return tx.Clauses(liveRowConflict(
		[]string{"student_id", "lesson_id"},
		clause.AssignmentColumns([]string{"completed_at", "updated_at", "deleted_at"}),
	)).Create(&completions).Error
}

// Daily sayaçları mevcut satır üzerine eklenir; ON CONFLICT hedefi partial
// unique index ile aynı olmalıdır, aksi halde SQLite eşleşen kısıt bulamaz.
func upsertDaily(tx *gorm.DB, aggregates []activities.DailyAggregate) error {
	if len(aggregates) == 0 {
		return nil
	}
	entities := make([]activities.DailyAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		aggregate.ID = uuid.New()
		entities = append(entities, aggregate)
	}
	return tx.Clauses(liveRowConflict(
		[]string{"student_id", "date", "course_id"},
		clause.Assignments(dailyIncrements),
	)).Create(&entities).Error
}

func (repository *ActivityRepository) List(ctx context.Context, params activities.ListParams) ([]activities.Activity, error) {
	if params.Limit < 1 || params.Limit > 500 {
		params.Limit = 100
	}
	query := repository.db.WithContext(ctx).Model(&activities.Activity{})
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.LessonID != nil {
		query = query.Where("lesson_id = ?", *params.LessonID)
	}
	if params.QuizID != nil {
		query = query.Where("quiz_id = ?", *params.QuizID)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.From != nil {
		query = query.Where("occurred_at >= ?", params.From.UTC())
	}
	if params.To != nil {
		query = query.Where("occurred_at <= ?", params.To.UTC())
	}
	var result []activities.Activity
	err := query.Order("occurred_at DESC").Limit(params.Limit).Find(&result).Error
	return result, translate(err)
}

func (repository *ActivityRepository) ListDaily(ctx context.Context, params activities.DailyParams) ([]activities.DailyAggregate, error) {
	query := repository.db.WithContext(ctx).Model(&activities.DailyAggregate{}).
		Where("student_id = ?", params.StudentID)
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.From != "" {
		query = query.Where("date >= ?", params.From)
	}
	if params.To != "" {
		query = query.Where("date <= ?", params.To)
	}
	var result []activities.DailyAggregate
	err := query.Order("date ASC").Find(&result).Error
	return result, translate(err)
}

func (repository *ActivityRepository) DailySeries(ctx context.Context, studentID uuid.UUID, courseID *uuid.UUID) ([]activities.DailyAggregate, error) {
	var result []activities.DailyAggregate
	type row struct {
		Date              string
		Events            int
		StudyMinutes      int
		LessonViews       int
		LessonCompletions int
		QuizStarted       int
		QuizCompleted     int
		QuestionAnswered  int
		CourseAccess      int
		Logins            int
	}
	query := repository.db.WithContext(ctx).Model(&activities.DailyAggregate{}).
		Select("date, SUM(events) AS events, SUM(study_minutes) AS study_minutes, SUM(lesson_views) AS lesson_views, SUM(lesson_completions) AS lesson_completions, SUM(quiz_started) AS quiz_started, SUM(quiz_completed) AS quiz_completed, SUM(question_answered) AS question_answered, SUM(course_access) AS course_access, SUM(logins) AS logins").
		Where("student_id = ?", studentID)
	if courseID != nil {
		query = query.Where("course_id = ?", *courseID)
	}
	var rows []row
	if err := query.Group("date").Order("date ASC").Find(&rows).Error; err != nil {
		return result, translate(err)
	}
	for _, item := range rows {
		result = append(result, activities.DailyAggregate{
			StudentID:         studentID,
			Date:              item.Date,
			Events:            item.Events,
			StudyMinutes:      item.StudyMinutes,
			LessonViews:       item.LessonViews,
			LessonCompletions: item.LessonCompletions,
			QuizStarted:       item.QuizStarted,
			QuizCompleted:     item.QuizCompleted,
			QuestionAnswered:  item.QuestionAnswered,
			CourseAccess:      item.CourseAccess,
			Logins:            item.Logins,
		})
	}
	return result, nil
}

func (repository *ActivityRepository) transaction(ctx context.Context, run func(tx *gorm.DB) error) error {
	return repository.db.WithContext(ctx).Transaction(run)
}

// dailyIncrements, mevcut günlük satırına ekleme yaparak paralel kayıtların
// sayaçları silmesini engeller.
var dailyIncrements = map[string]any{
	"events":             gorm.Expr("events + excluded.events"),
	"lesson_views":       gorm.Expr("lesson_views + excluded.lesson_views"),
	"lesson_completions": gorm.Expr("lesson_completions + excluded.lesson_completions"),
	"quiz_started":       gorm.Expr("quiz_started + excluded.quiz_started"),
	"quiz_completed":     gorm.Expr("quiz_completed + excluded.quiz_completed"),
	"question_answered":  gorm.Expr("question_answered + excluded.question_answered"),
	"course_access":      gorm.Expr("course_access + excluded.course_access"),
	"study_minutes":      gorm.Expr("study_minutes + excluded.study_minutes"),
	"logins":             gorm.Expr("logins + excluded.logins"),
	"updated_at":         gorm.Expr("CURRENT_TIMESTAMP"),
}
