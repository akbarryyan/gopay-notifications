package httpapi

import (
	"testing"
	"time"
)

func TestLoginThrottleMengizinkanSampaiBatas(t *testing.T) {
	th := newLoginThrottle()
	now := time.Unix(1789200000, 0)

	for i := 0; i < loginMaxAttempts; i++ {
		if !th.Allowed("1.2.3.4", now) {
			t.Fatalf("percobaan ke-%d seharusnya masih diizinkan", i+1)
		}
		th.RecordFailure("1.2.3.4", now)
	}

	if th.Allowed("1.2.3.4", now) {
		t.Fatal("percobaan setelah batas seharusnya ditolak")
	}
}

func TestLoginThrottlePulihSetelahJendelaLewat(t *testing.T) {
	th := newLoginThrottle()
	now := time.Unix(1789200000, 0)

	for i := 0; i < loginMaxAttempts; i++ {
		th.RecordFailure("1.2.3.4", now)
	}
	if th.Allowed("1.2.3.4", now) {
		t.Fatal("seharusnya terkunci di dalam jendela")
	}

	setelahJendela := now.Add(loginWindow + time.Minute)
	if !th.Allowed("1.2.3.4", setelahJendela) {
		t.Fatal("seharusnya pulih setelah jendela lewat")
	}
}

func TestLoginThrottleTidakMencampurIP(t *testing.T) {
	th := newLoginThrottle()
	now := time.Unix(1789200000, 0)

	for i := 0; i < loginMaxAttempts; i++ {
		th.RecordFailure("1.2.3.4", now)
	}

	if !th.Allowed("5.6.7.8", now) {
		t.Fatal("IP lain tidak boleh ikut terkunci")
	}
}

func TestLoginThrottleSuksesMembersihkanRiwayat(t *testing.T) {
	th := newLoginThrottle()
	now := time.Unix(1789200000, 0)

	for i := 0; i < loginMaxAttempts-1; i++ {
		th.RecordFailure("1.2.3.4", now)
	}
	th.RecordSuccess("1.2.3.4")

	for i := 0; i < loginMaxAttempts-1; i++ {
		if !th.Allowed("1.2.3.4", now) {
			t.Fatal("riwayat seharusnya bersih setelah login berhasil")
		}
		th.RecordFailure("1.2.3.4", now)
	}
}
