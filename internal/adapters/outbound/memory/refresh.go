package memory

import (
	"context"
	"sync"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RefreshStore is an in-memory app.RefreshTokenStore for local/dev runs. Refresh
// tokens are single-use: Consume revokes the record so a replay is rejected.
type RefreshStore struct {
	mu      sync.Mutex
	records map[string]app.RefreshRecord
}

// NewRefreshStore builds an empty in-memory refresh-token store.
func NewRefreshStore() *RefreshStore {
	return &RefreshStore{records: map[string]app.RefreshRecord{}}
}

// Reset clears every refresh grant (test/dev reseed helper).
func (s *RefreshStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = map[string]app.RefreshRecord{}
}

// Save records a freshly issued refresh grant.
func (s *RefreshStore) Save(_ context.Context, rec app.RefreshRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[rec.ID] = rec
	return nil
}

// Consume validates a grant by jti and revokes it (single-use rotation).
func (s *RefreshStore) Consume(_ context.Context, jti string, now time.Time) (app.RefreshRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[jti]
	if !ok || rec.Revoked {
		return app.RefreshRecord{}, kernel.Unauthorized("auth: refresh token is not valid")
	}
	if !now.Before(rec.ExpiresAt) {
		return app.RefreshRecord{}, kernel.Unauthorized("auth: refresh token has expired")
	}
	rec.Revoked = true
	s.records[jti] = rec
	return rec, nil
}

// Revoke marks a grant revoked; an unknown jti is a no-op.
func (s *RefreshStore) Revoke(_ context.Context, jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.records[jti]; ok {
		rec.Revoked = true
		s.records[jti] = rec
	}
	return nil
}

var _ app.RefreshTokenStore = (*RefreshStore)(nil)
