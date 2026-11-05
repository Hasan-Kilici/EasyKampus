package database

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"

	"lms/backend/pkg/domain/enrollments"
)

type EnrollmentRepository struct {
	db *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{db: db}
}

func (repository *EnrollmentRepository) Create(ctx context.Context, enrollment *enrollments.Enrollment) error {
	return translate(repository.db.WithContext(ctx).Create(enrollment).Error)
}

func (repository *EnrollmentRepository) CreateBulk(ctx context.Context, items []*enrollments.Enrollment) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := translate(tx.Create(item).Error); err != nil {
				return err
			}
		}
		return nil
	})
}

func (repository *EnrollmentRepository) Transfer(ctx context.Context, fromClassID uuid.UUID, studentID uuid.UUID, enrollment *enrollments.Enrollment) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if err := tx.Model(&enrollments.Enrollment{}).
			Where("class_id = ? AND student_id = ? AND status = ?", fromClassID, studentID, enrollments.StatusActive).
			Updates(map[string]any{"status": enrollments.StatusWithdrawn, "finished_at": &now}).Error; err != nil {
			return translate(err)
		}
		return translate(tx.Create(enrollment).Error)
	})
}

func (repository *EnrollmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*enrollments.Enrollment, error) {
	var enrollment enrollments.Enrollment
	err := repository.db.WithContext(ctx).First(&enrollment, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &enrollment, nil
}

func (repository *EnrollmentRepository) Update(ctx context.Context, enrollment *enrollments.Enrollment) error {
	return translate(repository.db.WithContext(ctx).Save(enrollment).Error)
}

func (repository *EnrollmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&enrollments.Enrollment{}, "id = ?", id).Error)
}

func (repository *EnrollmentRepository) List(ctx context.Context, params enrollments.ListParams) ([]enrollments.Enrollment, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{})
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	var result []enrollments.Enrollment
	err := query.Order("class_id ASC, student_id ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *EnrollmentRepository) CountByClass(ctx context.Context, classID uuid.UUID) (int64, error) {
	var count int64
	err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("class_id = ? AND status = ?", classID, enrollments.StatusActive).
		Count(&count).Error
	return count, translate(err)
}

func (repository *EnrollmentRepository) Exists(ctx context.Context, classID uuid.UUID, studentID uuid.UUID, courseID *uuid.UUID, term string) (bool, error) {
	query := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("class_id = ? AND student_id = ?", classID, studentID).
		Where("status = ?", enrollments.StatusActive)
	if courseID != nil {
		query = query.Where("(course_id = ? OR course_id IS NULL)", *courseID)
	} else {
		query = query.Where("course_id IS NULL")
	}
	if term = strings.TrimSpace(term); term != "" {
		query = query.Where("term = ?", term)
	}
	return existsQuery(ctx, query)
}

func (repository *EnrollmentRepository) StudentIDsForClass(ctx context.Context, classID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("class_id = ? AND status = ?", classID, enrollments.StatusActive).
		Distinct().
		Pluck("student_id", &ids).Error
	return ids, translate(err)
}

func (repository *EnrollmentRepository) StudentIDsForCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("course_id = ? AND status = ?", courseID, enrollments.StatusActive).
		Distinct().
		Pluck("student_id", &ids).Error
	return ids, translate(err)
}
