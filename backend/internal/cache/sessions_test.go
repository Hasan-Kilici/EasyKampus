package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/sessions"
)

type fakeSessionService struct {
	getCalls int
	revoked  bool
	sessions map[string]*sessions.Session
}

func (fake *fakeSessionService) Create(_ context.Context, userID uuid.UUID, _ string, lifetime time.Duration) (*sessions.Session, error) {
	session := &sessions.Session{Model: base.Model{ID: uuid.New()}, UserID: userID, ExpiresAt: time.Now().Add(lifetime)}
	return session, nil
}

func (fake *fakeSessionService) GetActive(_ context.Context, id string) (*sessions.Session, error) {
	fake.getCalls++
	if fake.revoked {
		return nil, core.ErrUnauthorized
	}
	session, ok := fake.sessions[id]
	if !ok {
		return nil, core.ErrUnauthorized
	}
	copy := *session
	return &copy, nil
}

func (fake *fakeSessionService) Revoke(_ context.Context, id string) error {
	fake.revoked = true
	return nil
}

func TestSessionServiceCachesLookups(t *testing.T) {
	id := uuid.NewString()
	fake := &fakeSessionService{sessions: map[string]*sessions.Session{
		id: {Model: base.Model{ID: uuid.MustParse(id)}, UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)},
	}}
	service := NewSessionService(fake, 10, time.Minute)

	first, err := service.GetActive(context.Background(), id)
	if err != nil {
		t.Fatalf("first GetActive: %v", err)
	}
	second, err := service.GetActive(context.Background(), strings.ToUpper(id))
	if err != nil {
		t.Fatalf("second GetActive: %v", err)
	}
	if fake.getCalls != 1 {
		t.Fatalf("expected 1 DB lookup, got %d", fake.getCalls)
	}
	if first.ID != second.ID {
		t.Fatalf("cached session mismatch: %v vs %v", first.ID, second.ID)
	}
	if first == second {
		t.Fatal("expected a copy, got the same pointer")
	}
}

func TestSessionServiceRevokeInvalidatesCache(t *testing.T) {
	id := uuid.NewString()
	fake := &fakeSessionService{sessions: map[string]*sessions.Session{
		id: {Model: base.Model{ID: uuid.MustParse(id)}, UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)},
	}}
	service := NewSessionService(fake, 10, time.Minute)

	if _, err := service.GetActive(context.Background(), id); err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if err := service.Revoke(context.Background(), id); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := service.GetActive(context.Background(), id); err == nil {
		t.Fatal("expected revoked session to be rejected")
	}
	if fake.getCalls != 2 {
		t.Fatalf("expected cache miss after revoke, got %d DB lookups", fake.getCalls)
	}
}
