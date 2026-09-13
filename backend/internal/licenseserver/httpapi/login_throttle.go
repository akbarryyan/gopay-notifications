package httpapi

import (
	"sync"
	"time"
)

// loginThrottle membatasi percobaan per alamat IP — dipakai baik untuk
// login admin vendor maupun /activate dan /validate (endpoint publik yang
// menghadap internet langsung). Salinan dari backend/internal/httpapi
// (paket berbeda, tidak bisa diimpor lintas paket internal tanpa
// mengekspornya — duplikasi kecil ini lebih sederhana daripada itu).
type loginThrottle struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newLoginThrottle() *loginThrottle {
	return &loginThrottle{attempts: make(map[string][]time.Time)}
}

const (
	throttleMaxAttempts = 5
	throttleWindow      = 15 * time.Minute
)

func (t *loginThrottle) Allowed(ip string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	kept := t.pruneLocked(ip, now)
	return len(kept) < throttleMaxAttempts
}

func (t *loginThrottle) RecordFailure(ip string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	kept := t.pruneLocked(ip, now)
	t.attempts[ip] = append(kept, now)
}

func (t *loginThrottle) RecordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, ip)
}

func (t *loginThrottle) pruneLocked(ip string, now time.Time) []time.Time {
	var kept []time.Time
	for _, at := range t.attempts[ip] {
		if now.Sub(at) < throttleWindow {
			kept = append(kept, at)
		}
	}
	t.attempts[ip] = kept
	return kept
}
