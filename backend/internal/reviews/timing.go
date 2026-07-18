package reviews

import (
	"sync"
	"time"
)

type TimingStore struct { mu sync.Mutex; opened map[string]time.Time; visited map[string]bool }
func NewTimingStore() *TimingStore { return &TimingStore{opened: make(map[string]time.Time), visited: make(map[string]bool)} }
func (s *TimingStore) Open(reviewerCode, caseID string, now time.Time) bool { key := key(reviewerCode, caseID); s.mu.Lock(); defer s.mu.Unlock(); revisited := s.visited[key]; s.visited[key] = true; s.opened[key] = now; return revisited }
func (s *TimingStore) Close(reviewerCode, caseID string, now time.Time) (int, bool) { key := key(reviewerCode, caseID); s.mu.Lock(); defer s.mu.Unlock(); openedAt, ok := s.opened[key]; if !ok { return 0, false }; delete(s.opened, key); seconds := int(now.Sub(openedAt).Seconds()); if seconds < 0 { seconds = 0 }; return seconds, true }
