package session

import "sync"

type ActiveSession struct {
	ClassID    string
	StartedAt  string
	Attendance map[string]string
}

var (
	mu sync.RWMutex
	s  *ActiveSession
)

func Set(v *ActiveSession) {
	mu.Lock()
	s = v
	mu.Unlock()
}

func Get() *ActiveSession {
	mu.RLock()
	defer mu.RUnlock()
	return s
}

func Clear() {
	mu.Lock()
	s = nil
	mu.Unlock()
}

func WithWrite(fn func(s *ActiveSession)) {
	mu.Lock()
	defer mu.Unlock()
	if s != nil {
		fn(s)
	}
}
