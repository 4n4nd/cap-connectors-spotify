package store

import "sync"

// TokenStore exposes a minimal surface for retrieving user access tokens.
type TokenStore interface {
	AccessToken(userID string) string
	SetAccessToken(userID, token string)
}

type memoryTokenStore struct {
	mu sync.RWMutex
	m  map[string]string
}

// NewMemoryTokenStore returns an in-memory TokenStore.
func NewMemoryTokenStore() TokenStore {
	return &memoryTokenStore{
		m: make(map[string]string),
	}
}

func (s *memoryTokenStore) AccessToken(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[userID]
}

func (s *memoryTokenStore) SetAccessToken(userID, token string) {
	s.mu.Lock()
	s.m[userID] = token
	s.mu.Unlock()
}
