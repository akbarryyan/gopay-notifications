package httpapi

import (
	"sync"
	"time"
)

// loginThrottle membatasi percobaan login per alamat IP.
//
// bcrypt sendiri sudah lambat, tetapi endpoint login menghadap internet
// langsung di banyak pemasangan self-hosted — tanpa batas percobaan sama
// sekali adalah celah yang terlalu murah untuk dibiarkan, bahkan untuk MVP.
type loginThrottle struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newLoginThrottle() *loginThrottle {
	return &loginThrottle{attempts: make(map[string][]time.Time)}
}

const (
	loginMaxAttempts = 5
	loginWindow      = 15 * time.Minute
)

// Allowed melaporkan apakah IP masih boleh mencoba login.
func (t *loginThrottle) Allowed(ip string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	kept := t.pruneLocked(ip, now)
	return len(kept) < loginMaxAttempts
}

// RecordFailure mencatat satu percobaan gagal.
func (t *loginThrottle) RecordFailure(ip string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	kept := t.pruneLocked(ip, now)
	t.attempts[ip] = append(kept, now)
}

// RecordSuccess membersihkan riwayat kegagalan setelah login berhasil.
func (t *loginThrottle) RecordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, ip)
}

func (t *loginThrottle) pruneLocked(ip string, now time.Time) []time.Time {
	var kept []time.Time
	for _, at := range t.attempts[ip] {
		if now.Sub(at) < loginWindow {
			kept = append(kept, at)
		}
	}
	t.attempts[ip] = kept
	return kept
}
