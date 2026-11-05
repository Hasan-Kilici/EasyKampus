package sessions

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, session *Session) error
	GetActive(ctx context.Context, id uuid.UUID) (*Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}
