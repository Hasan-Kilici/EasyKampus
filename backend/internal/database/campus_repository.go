package database

import (
	"context"
	"lms/backend/internal/database/listscope"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/schools"
)

type CampusRepository struct {
	db *gorm.DB
}

func NewCampusRepository(db *gorm.DB) *CampusRepository {
	return &CampusRepository{db: db}
}

func (repository *CampusRepository) Create(ctx context.Context, campus *campuses.Campus) error {
	return translate(repository.db.WithContext(ctx).Create(campus).Error)
}

func (repository *CampusRepository) GetByID(ctx context.Context, id uuid.UUID) (*campuses.Campus, error) {
	var campus campuses.Campus
	err := repository.db.WithContext(ctx).First(&campus, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &campus, nil
}

func (repository *CampusRepository) GetSchoolID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	campus, err := repository.GetByID(ctx, id)
	if err != nil {
		return uuid.Nil, err
	}
	return campus.SchoolID, nil
}

func (repository *CampusRepository) Update(ctx context.Context, campus *campuses.Campus) error {
	return translate(repository.db.WithContext(ctx).Save(campus).Error)
}

func (repository *CampusRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&campuses.Campus{}, "id = ?", id).Error)
}

func (repository *CampusRepository) List(ctx context.Context, params campuses.ListParams) ([]campuses.Campus, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&campuses.Campus{})
	if filter := listscope.Campuses(params.Viewer); !filter.Unrestricted {
		query = query.Where(filter.Clause, filter.Args...)
	}
	if params.SchoolID != nil {
		query = query.Where("school_id = ?", *params.SchoolID)
	}
	if keyword := strings.TrimSpace(params.Query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", like, like)
	}
	var result []campuses.Campus
	err := query.Order("name ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *CampusRepository) SchoolExists(ctx context.Context, schoolID uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &schools.School{}, "id = ?", schoolID)
}
