package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/users"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (repository *UserRepository) Create(ctx context.Context, user *users.User) error {
	return translate(repository.db.WithContext(ctx).Create(user).Error)
}

func (repository *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*users.User, error) {
	var user users.User
	err := repository.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repository *UserRepository) GetByEmail(ctx context.Context, emailHash string) (*users.User, error) {
	var user users.User
	err := repository.db.WithContext(ctx).Where("email_hash = ?", emailHash).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repository *UserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]users.User, error) {
	if len(ids) == 0 {
		return []users.User{}, nil
	}
	var result []users.User
	err := repository.db.WithContext(ctx).Where("id IN ?", ids).Find(&result).Error
	return result, err
}

func (repository *UserRepository) Update(ctx context.Context, user *users.User) error {
	result := repository.db.WithContext(ctx).Save(user)
	if err := translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (repository *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := repository.db.WithContext(ctx).Delete(&users.User{}, "id = ?", id)
	if err := translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (repository *UserRepository) List(ctx context.Context, params users.ListParams) ([]users.User, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var result []users.User
	err := repository.db.WithContext(ctx).
		Offset((page - 1) * limit).
		Limit(limit).
		Order("created_at DESC").
		Find(&result).Error
	return result, err
}
