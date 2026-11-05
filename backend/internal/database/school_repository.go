package database

import (
	"context"
	"lms/backend/internal/database/listscope"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/schools"
)

type SchoolRepository struct {
	db *gorm.DB
}

func NewSchoolRepository(db *gorm.DB) *SchoolRepository {
	return &SchoolRepository{db: db}
}

func (repository *SchoolRepository) Create(ctx context.Context, school *schools.School) error {
	return translate(repository.db.WithContext(ctx).Create(school).Error)
}

func (repository *SchoolRepository) GetByID(ctx context.Context, id uuid.UUID) (*schools.School, error) {
	var school schools.School
	err := repository.db.WithContext(ctx).First(&school, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &school, nil
}

func (repository *SchoolRepository) Update(ctx context.Context, school *schools.School) error {
	result := repository.db.WithContext(ctx).Save(school)
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil
	}
	return nil
}

func (repository *SchoolRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&schools.School{}, "id = ?", id).Error)
}

func (repository *SchoolRepository) List(ctx context.Context, params schools.ListParams) ([]schools.School, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&schools.School{})
	if filter := listscope.Schools(params.Viewer); !filter.Unrestricted {
		query = query.Where(filter.Clause, filter.Args...)
	}
	if keyword := strings.TrimSpace(params.Query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", like, like)
	}
	var result []schools.School
	err := query.Order("name ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *SchoolRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &schools.School{}, "id = ?", id)
}

func (repository *SchoolRepository) ListStaff(ctx context.Context, schoolID uuid.UUID) ([]schools.Staff, error) {
	var result []schools.Staff
	err := repository.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("role ASC").
		Find(&result).Error
	return result, translate(err)
}

func (repository *SchoolRepository) AssignStaff(ctx context.Context, staff *schools.Staff) error {
	err := repository.db.WithContext(ctx).
		Where("school_id = ? AND user_id = ?", staff.SchoolID, staff.UserID).
		Delete(&schools.Staff{}).Error
	if err != nil {
		return translate(err)
	}
	return translate(repository.db.WithContext(ctx).Create(staff).Error)
}

func (repository *SchoolRepository) RemoveStaff(ctx context.Context, schoolID uuid.UUID, userID uuid.UUID) error {
	result := repository.db.WithContext(ctx).
		Delete(&schools.Staff{}, "school_id = ? AND user_id = ?", schoolID, userID)
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil
	}
	return nil
}
