package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/sessions"
)

type SessionRepository struct{ db *gorm.DB }

func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }

func (repository *SessionRepository) Create(ctx context.Context, session *sessions.Session) error {
	return translate(repository.db.WithContext(ctx).Create(session).Error)
}

func (repository *SessionRepository) GetActive(ctx context.Context, id uuid.UUID) (*sessions.Session, error) {
	var session sessions.Session
	err := repository.db.WithContext(ctx).
		Where("id = ? AND revoked_at IS NULL AND expires_at > ?", id.String(), time.Now()).
		Take(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (repository *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := repository.db.WithContext(ctx).
		Model(&sessions.Session{}).
		Where("id = ? AND revoked_at IS NULL", id.String()).
		Update("revoked_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
