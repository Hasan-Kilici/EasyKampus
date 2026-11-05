package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"lms/backend/pkg/domain/sessions"
)

// SessionService caches successful session lookups for a short TTL. Auth runs on
// every protected request, so without this the same session row is read from
// SQLite on every call. Revocation removes the cache entry immediately; the TTL
// only bounds entries revoked by another process.
type SessionService struct {
	inner sessions.Service
	cache *expirable.LRU[string, *sessions.Session]
}

func NewSessionService(inner sessions.Service, size int, ttl time.Duration) *SessionService {
	return &SessionService{
		inner: inner,
		cache: expirable.NewLRU[string, *sessions.Session](size, nil, ttl),
	}
}

func (service *SessionService) Create(ctx context.Context, userID uuid.UUID, userAgent string, lifetime time.Duration) (*sessions.Session, error) {
	session, err := service.inner.Create(ctx, userID, userAgent, lifetime)
	if err != nil {
		return nil, err
	}
	copy := *session
	service.cache.Add(session.ID.String(), &copy)
	return session, nil
}

func (service *SessionService) GetActive(ctx context.Context, id string) (*sessions.Session, error) {
	key, ok := sessionCacheKey(id)
	if ok {
		if cached, found := service.cache.Get(key); found {
			if cached.RevokedAt == nil && time.Now().Before(cached.ExpiresAt) {
				copy := *cached
				return &copy, nil
			}
			service.cache.Remove(key)
		}
	}
	session, err := service.inner.GetActive(ctx, id)
	if err != nil {
		return nil, err
	}
	copy := *session
	service.cache.Add(session.ID.String(), &copy)
	return session, nil
}

func (service *SessionService) Revoke(ctx context.Context, id string) error {
	if key, ok := sessionCacheKey(id); ok {
		service.cache.Remove(key)
	}
	return service.inner.Revoke(ctx, id)
}

func sessionCacheKey(id string) (string, bool) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}
