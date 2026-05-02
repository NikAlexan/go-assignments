package idempotency

import "sync"

type Store struct {
	mu   sync.RWMutex
	seen map[string]bool
}

func NewStore() *Store {
	return &Store{seen: make(map[string]bool)}
}

func (s *Store) Seen(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seen[id]
}

func (s *Store) Mark(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[id] = true
}
