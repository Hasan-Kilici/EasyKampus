package database

import (
	"context"
	"lms/backend/internal/database/listscope"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/schools"
)

type CourseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func (repository *CourseRepository) Create(ctx context.Context, course *courses.Course) error {
	return translate(repository.db.WithContext(ctx).Create(course).Error)
}

func (repository *CourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*courses.Course, error) {
	var course courses.Course
	err := repository.db.WithContext(ctx).First(&course, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &course, nil
}

func (repository *CourseRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &courses.Course{}, "id = ?", id)
}

func (repository *CourseRepository) SchoolExists(ctx context.Context, schoolID uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &schools.School{}, "id = ?", schoolID)
}

func (repository *CourseRepository) Update(ctx context.Context, course *courses.Course) error {
	return translate(repository.db.WithContext(ctx).Save(course).Error)
}

func (repository *CourseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&courses.Course{}, "id = ?", id).Error)
}

func (repository *CourseRepository) List(ctx context.Context, params courses.ListParams) ([]courses.Course, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&courses.Course{})
	if filter := listscope.Courses(params.Viewer); !filter.Unrestricted {
		query = query.Where(filter.Clause, filter.Args...)
	}
	if params.SchoolID != nil {
		query = query.Where("school_id = ?", *params.SchoolID)
	}
	if params.ActiveOnly {
		query = query.Where("active = ?", true)
	}
	if area := strings.TrimSpace(params.SubjectArea); area != "" {
		query = query.Where("subject_area = ?", area)
	}
	if keyword := strings.TrimSpace(params.Query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", like, like)
	}
	var result []courses.Course
	err := query.Order("name ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}
