package database

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"lms/backend/pkg/core"
)

var errNotFound = core.ErrNotFound

func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return core.ErrAlreadyExists
	}
	return err
}

func exists(ctx context.Context, db *gorm.DB, model any, query string, args ...any) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(model).Where(query, args...).Count(&count).Error
	return count > 0, translate(err)
}

func existsQuery(ctx context.Context, query *gorm.DB) (bool, error) {
	var count int64
	err := query.WithContext(ctx).Limit(1).Count(&count).Error
	return count > 0, translate(err)
}

func window(page int, limit int, defaultLimit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > 200 {
		limit = 200
	}
	return (page - 1) * limit, limit
}
