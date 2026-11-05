package sessions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, userAgent string, lifetime time.Duration) (*Session, error)
	GetActive(ctx context.Context, id string) (*Session, error)
	Revoke(ctx context.Context, id string) error
}

type service struct{ repository Repository }

func NewService(repository Repository) Service { return &service{repository: repository} }

func (service *service) Create(ctx context.Context, userID uuid.UUID, userAgent string, lifetime time.Duration) (*Session, error) {
	if userID == uuid.Nil || lifetime <= 0 {
		return nil, core.ErrValidation
	}
	session := &Session{UserID: userID, UserAgent: userAgent, JWTKey: uuid.NewString(), ExpiresAt: time.Now().Add(lifetime)}
	if err := service.repository.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (service *service) GetActive(ctx context.Context, id string) (*Session, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetActive(ctx, parsedID)
}

func (service *service) Revoke(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Revoke(ctx, parsedID)
}
