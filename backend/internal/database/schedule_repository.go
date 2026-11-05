package database

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"gorm.io/gorm"

	"lms/backend/pkg/domain/schedules"
)

type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (repository *ScheduleRepository) Create(ctx context.Context, entry *schedules.ScheduleEntry) error {
	return translate(repository.db.WithContext(ctx).Create(entry).Error)
}

func (repository *ScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*schedules.ScheduleEntry, error) {
	var entry schedules.ScheduleEntry
	err := repository.db.WithContext(ctx).First(&entry, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &entry, nil
}

func (repository *ScheduleRepository) Update(ctx context.Context, entry *schedules.ScheduleEntry) error {
	return translate(repository.db.WithContext(ctx).Save(entry).Error)
}

func (repository *ScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&schedules.ScheduleEntry{}, "id = ?", id).Error)
}

func (repository *ScheduleRepository) List(ctx context.Context, params schedules.ListParams, limit int) ([]schedules.ScheduleEntry, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	query := repository.db.WithContext(ctx).Model(&schedules.ScheduleEntry{})
	if params.ScopeApplied && len(params.ClassIDs) == 0 {
		return []schedules.ScheduleEntry{}, nil
	}
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if len(params.ClassIDs) > 0 {
		query = query.Where("class_id IN ?", params.ClassIDs)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.TeacherID != nil {
		query = query.Where("teacher_id = ?", *params.TeacherID)
	}
	if term := strings.TrimSpace(params.Term); term != "" {
		query = query.Where("term = ?", term)
	}
	if params.DayOfWeek != nil {
		query = query.Where("day_of_week = ?", *params.DayOfWeek)
	}
	if params.ActiveOnly {
		query = query.Where("active = ?", true)
	}
	var result []schedules.ScheduleEntry
	err := query.Order("day_of_week ASC, start_minute ASC").Limit(limit).Find(&result).Error
	return result, translate(err)
}
