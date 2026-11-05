package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/approvals"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
)

type LessonRepository struct {
	db *gorm.DB
}

func NewLessonRepository(db *gorm.DB) *LessonRepository {
	return &LessonRepository{db: db}
}

func (repository *LessonRepository) Create(ctx context.Context, lesson *lessons.Lesson) error {
	return translate(repository.db.WithContext(ctx).Create(lesson).Error)
}

func (repository *LessonRepository) GetByID(ctx context.Context, id uuid.UUID) (*lessons.Lesson, error) {
	var lesson lessons.Lesson
	err := repository.db.WithContext(ctx).First(&lesson, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &lesson, nil
}

func (repository *LessonRepository) Update(ctx context.Context, lesson *lessons.Lesson) error {
	return translate(repository.db.WithContext(ctx).Save(lesson).Error)
}

func (repository *LessonRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&lessons.Lesson{}, "id = ?", id).Error)
}

func (repository *LessonRepository) ListByCourse(ctx context.Context, params lessons.ListParams) ([]lessons.Lesson, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Where("course_id = ?", params.CourseID)
	if params.OnlyPublished {
		query = query.Where("state = ?", "approved")
	}
	var result []lessons.Lesson
	err := query.Order("position ASC, created_at DESC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *LessonRepository) CountByCreatorAndState(ctx context.Context, creatorID uuid.UUID) (map[string]int64, error) {
	type row struct {
		State  string
		Amount int64
	}
	var rows []row
	err := repository.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Select("state, COUNT(*) AS amount").
		Where("created_by_id = ?", creatorID).
		Group("state").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.State] = item.Amount
	}
	return result, nil
}

func (repository *LessonRepository) CountInScope(ctx context.Context, courseIDs []uuid.UUID) (int64, error) {
	if len(courseIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := repository.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Where("course_id IN ?", courseIDs).
		Count(&count).Error
	return count, translate(err)
}

func (repository *LessonRepository) UpsertCompletions(ctx context.Context, completions []lessons.LessonCompletion) error {
	if len(completions) == 0 {
		return nil
	}
	entities := make([]lessons.LessonCompletion, 0, len(completions))
	for index := range completions {
		completions[index].ID = uuid.New()
		entities = append(entities, completions[index])
	}
	return translate(repository.db.WithContext(ctx).
		Clauses(liveRowConflict([]string{"student_id", "lesson_id"}, clause.AssignmentColumns([]string{"completed_at", "updated_at", "deleted_at"}))).
		Create(&entities).Error)
}

func (repository *LessonRepository) ListCompletions(ctx context.Context, studentID uuid.UUID, courseID *uuid.UUID) ([]lessons.LessonCompletion, error) {
	query := repository.db.WithContext(ctx).Model(&lessons.LessonCompletion{}).
		Where("student_id = ?", studentID)
	if courseID != nil {
		query = query.Where("course_id = ?", *courseID)
	}
	var result []lessons.LessonCompletion
	err := query.Order("completed_at DESC").Find(&result).Error
	return result, translate(err)
}

func (repository *LessonRepository) LessonIDsByCourse(ctx context.Context, courseIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Where("course_id IN ?", courseIDs).
		Pluck("id", &ids).Error
	return ids, translate(err)
}

func (repository *LessonRepository) PublishedLessonIDs(ctx context.Context, courseIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Where("course_id IN ? AND state = ?", courseIDs, "approved").
		Pluck("id", &ids).Error
	return ids, translate(err)
}

type QuizRepository struct {
	db *gorm.DB
}

func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

func (repository *QuizRepository) Create(ctx context.Context, quiz *quizes.Quiz) error {
	return translate(repository.db.WithContext(ctx).Create(quiz).Error)
}

func (repository *QuizRepository) GetByID(ctx context.Context, id uuid.UUID) (*quizes.Quiz, error) {
	var quiz quizes.Quiz
	err := repository.db.WithContext(ctx).First(&quiz, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &quiz, nil
}

func (repository *QuizRepository) Update(ctx context.Context, quiz *quizes.Quiz) error {
	return translate(repository.db.WithContext(ctx).Save(quiz).Error)
}

func (repository *QuizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&quizes.Quiz{}, "id = ?", id).Error)
}

func (repository *QuizRepository) ListByClass(ctx context.Context, params quizes.ListParams) ([]quizes.Quiz, error) {
	query := repository.db.WithContext(ctx).Model(&quizes.Quiz{}).Where("class_id = ?", params.ClassID)
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.OnlyPublished {
		query = query.Where("state = ?", approvals.StateApproved)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 200
	}
	var result []quizes.Quiz
	if err := query.Order("created_at DESC").Limit(limit).Find(&result).Error; err != nil {
		return nil, translate(err)
	}
	return result, nil
}

func (repository *QuizRepository) ListByClassCourse(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, params quizes.ListParams) ([]quizes.Quiz, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&quizes.Quiz{}).
		Where("class_id = ? AND course_id = ?", classID, courseID)
	if params.OnlyPublished {
		query = query.Where("state = ?", "approved")
	}
	var result []quizes.Quiz
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *QuizRepository) ListByCourse(ctx context.Context, courseIDs []uuid.UUID) ([]quizes.Quiz, error) {
	if len(courseIDs) == 0 {
		return []quizes.Quiz{}, nil
	}
	var result []quizes.Quiz
	err := repository.db.WithContext(ctx).
		Where("course_id IN ?", courseIDs).
		Order("created_at DESC").
		Find(&result).Error
	return result, translate(err)
}

func (repository *QuizRepository) CountByCreatorAndState(ctx context.Context, creatorID uuid.UUID) (map[string]int64, error) {
	type row struct {
		State  string
		Amount int64
	}
	var rows []row
	err := repository.db.WithContext(ctx).Model(&quizes.Quiz{}).
		Select("state, COUNT(*) AS amount").
		Where("created_by_id = ?", creatorID).
		Group("state").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.State] = item.Amount
	}
	return result, nil
}

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

func (repository *QuestionRepository) Create(ctx context.Context, question *questions.Question) error {
	return translate(repository.db.WithContext(ctx).Create(question).Error)
}

func (repository *QuestionRepository) GetByID(ctx context.Context, id uuid.UUID) (*questions.Question, error) {
	var question questions.Question
	err := repository.db.WithContext(ctx).First(&question, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &question, nil
}

func (repository *QuestionRepository) Update(ctx context.Context, question *questions.Question) error {
	return translate(repository.db.WithContext(ctx).Save(question).Error)
}

func (repository *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&questions.Question{}, "id = ?", id).Error)
}

func (repository *QuestionRepository) ListByQuiz(ctx context.Context, params questions.ListParams) ([]questions.Question, error) {
	query := repository.db.WithContext(ctx).Where("quiz_id = ?", params.QuizID)
	if params.OnlyPublished {
		query = query.Where("state = ?", approvals.StateApproved)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 200
	}
	var result []questions.Question
	err := query.Order("position ASC").Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *QuestionRepository) CountByCreator(ctx context.Context, creatorID uuid.UUID) (int64, error) {
	var count int64
	err := repository.db.WithContext(ctx).Model(&questions.Question{}).
		Where("created_by_id = ?", creatorID).
		Count(&count).Error
	return count, translate(err)
}

func (repository *QuestionRepository) CountInScope(ctx context.Context, classIDs []uuid.UUID) (int64, error) {
	if len(classIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := repository.db.WithContext(ctx).Model(&questions.Question{}).
		Joins("JOIN quizzes ON quizzes.id = questions.quiz_id AND quizzes.deleted_at IS NULL").
		Where("quizzes.class_id IN ?", classIDs).
		Count(&count).Error
	return count, translate(err)
}

func (repository *QuestionRepository) GetByIDInQuiz(ctx context.Context, id uuid.UUID, quizID uuid.UUID) (*questions.Question, error) {
	question, err := repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if question.QuizID != quizID {
		return nil, errNotFound
	}
	return question, nil
}

type OptionRepository struct {
	db *gorm.DB
}

func NewOptionRepository(db *gorm.DB) *OptionRepository {
	return &OptionRepository{db: db}
}

func (repository *OptionRepository) Create(ctx context.Context, option *options.Option) error {
	return translate(repository.db.WithContext(ctx).Create(option).Error)
}

func (repository *OptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*options.Option, error) {
	var option options.Option
	err := repository.db.WithContext(ctx).First(&option, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &option, nil
}

func (repository *OptionRepository) GetByIDInQuestion(ctx context.Context, id uuid.UUID, questionID uuid.UUID) (*options.Option, error) {
	option, err := repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if option.QuestionID != questionID {
		return nil, errNotFound
	}
	return option, nil
}

func (repository *OptionRepository) Update(ctx context.Context, option *options.Option) error {
	return translate(repository.db.WithContext(ctx).Save(option).Error)
}

func (repository *OptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&options.Option{}, "id = ?", id).Error)
}

func (repository *OptionRepository) ListByQuestion(ctx context.Context, questionID uuid.UUID, page int, limit int) ([]options.Option, error) {
	offset, size := window(page, limit, 50)
	var result []options.Option
	err := repository.db.WithContext(ctx).
		Where("question_id = ?", questionID).
		Order("position ASC").
		Offset(offset).Limit(size).
		Find(&result).Error
	return result, translate(err)
}
