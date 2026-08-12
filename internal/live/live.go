// Package live holds the most recent raw value of every polled signal in memory,
// so the dashboard's real-time view never has to hit the database.
package live

import (
	"strconv"
	"sync"
	"time"
)

type Sample struct {
	Raw  int64
	Time time.Time
}

type Snapshot struct {
	mu sync.RWMutex
	m  map[string]Sample
}

func New() *Snapshot { return &Snapshot{m: make(map[string]Sample)} }

func (s *Snapshot) Set(key string, raw int64, t time.Time) {
	s.mu.Lock()
	s.m[key] = Sample{Raw: raw, Time: t}
	s.mu.Unlock()
}

func (s *Snapshot) Get(key string) (Sample, bool) {
	s.mu.RLock()
	v, ok := s.m[key]
	s.mu.RUnlock()
	return v, ok
}

// Furnace returns a register value from the ABC board (Modbus addr 1).
func (s *Snapshot) Furnace(reg int) (int64, bool) {
	v, ok := s.Get("furnace:1:" + strconv.Itoa(reg))
	return v.Raw, ok
}

// FreshestAge reports how long ago the newest sample landed (data-liveness).
func (s *Snapshot) FreshestAge(now time.Time) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var newest time.Time
	for _, v := range s.m {
		if v.Time.After(newest) {
			newest = v.Time
		}
	}
	if newest.IsZero() {
		return -1
	}
	return now.Sub(newest)
}
