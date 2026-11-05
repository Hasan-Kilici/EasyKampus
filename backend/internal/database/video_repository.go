package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/video"
)

type videoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) video.Repository {
	return &videoRepository{db: db}
}

func (repository *videoRepository) Create(ctx context.Context, v *video.Video) error {
	return translate(repository.db.WithContext(ctx).Create(v).Error)
}

func (repository *videoRepository) GetByID(ctx context.Context, id uuid.UUID) (*video.Video, error) {
	var v video.Video
	err := repository.db.WithContext(ctx).First(&v, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &v, nil
}

func (repository *videoRepository) Update(ctx context.Context, v *video.Video) error {
	return translate(repository.db.WithContext(ctx).Save(v).Error)
}

func (repository *videoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&video.Video{}, "id = ?", id).Error)
}

func (repository *videoRepository) ListByLesson(ctx context.Context, params video.ListParams) ([]video.Video, error) {
	query := repository.db.WithContext(ctx).Where("lesson_id = ?", params.LessonID)
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	var result []video.Video
	err := query.Order("position ASC, created_at ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *videoRepository) UpsertCompletions(ctx context.Context, completions []video.VideoCompletion) error {
	if len(completions) == 0 {
		return nil
	}
	return translate(repository.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "student_id"}, {Name: "video_id"}},
		DoNothing: true,
	}).Create(&completions).Error)
}

func (repository *videoRepository) ListCompletions(ctx context.Context, studentID uuid.UUID, lessonID *uuid.UUID) ([]video.VideoCompletion, error) {
	query := repository.db.WithContext(ctx).Where("student_id = ?", studentID)
	if lessonID != nil {
		query = query.Where("lesson_id = ?", *lessonID)
	}
	var result []video.VideoCompletion
	err := query.Order("completed_at DESC").Find(&result).Error
	return result, translate(err)
}
