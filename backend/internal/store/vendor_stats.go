package store

import (
	"context"
	"fmt"
	"time"
)

// VendorOverviewStats meringkas seluruh account lintas platform, untuk
// halaman Dashboard di Vendor Dashboard -- beda dari EventStats/
// DailyEventCounts di stats.go yang selalu di-scope satu accountID
// (dipakai Customer Dashboard), method-method di file ini SENGAJA lintas
// account karena memang itu tugas vendor: melihat seluruh platform.
type VendorOverviewStats struct {
	TotalAccounts int
	Active        int
	Expiring      int
	Expired       int
	Suspended     int
	Revoked       int
	NewThisWeek   int
	TotalDevices  int
	// TotalPaidAmountRp: total nominal invoice yang pernah lunas, sepanjang
	// waktu, lintas seluruh account -- bukan per hari (lihat VendorDailyStats
	// untuk itu).
	TotalPaidAmountRp int64
}

// VendorOverviewStats menghitung ringkasan lewat ListAccounts yang sudah
// ada (bukan query agregat SQL terpisah untuk status) -- DerivedStatus
// adalah method Go di Account, satu-satunya sumber kebenaran soal kapan
// account dianggap "expiring" dkk (WarningThresholdDays). Menduplikasi
// logika itu ke SQL berisiko dua tempat itu diam-diam tidak sinkron.
func (s *Store) VendorOverviewStats(ctx context.Context, now time.Time) (VendorOverviewStats, error) {
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return VendorOverviewStats{}, fmt.Errorf("store: vendor overview stats: %w", err)
	}

	var st VendorOverviewStats
	st.TotalAccounts = len(accounts)
	sevenDaysAgo := now.AddDate(0, 0, -7)
	for _, acc := range accounts {
		switch acc.DerivedStatus(now) {
		case "active":
			st.Active++
		case "expiring":
			st.Expiring++
		case "expired":
			st.Expired++
		case "suspended":
			st.Suspended++
		case "revoked":
			st.Revoked++
		}
		if acc.CreatedAt.After(sevenDaysAgo) {
			st.NewThisWeek++
		}
	}

	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM devices`).Scan(&st.TotalDevices); err != nil {
		return VendorOverviewStats{}, fmt.Errorf("store: total devices: %w", err)
	}

	if err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(requested_amount), 0) FROM invoices WHERE status = $1`, InvoiceStatusPaid).
		Scan(&st.TotalPaidAmountRp); err != nil {
		return VendorOverviewStats{}, fmt.Errorf("store: total paid amount: %w", err)
	}

	return st, nil
}

// VendorDailyStat adalah satu titik pada grafik tren Dashboard Vendor --
// dua series (account baru dan nominal lunas), lintas SEMUA account,
// sama seperti DailyEventCount di stats.go tapi tanpa filter account_id.
type VendorDailyStat struct {
	Date         time.Time // tengah malam UTC hari itu
	NewAccounts  int
	PaidAmountRp int64
}

// VendorDailyStats mengembalikan jumlah account baru DAN total nominal
// lunas per hari untuk `days` hari terakhir (termasuk hari ini), hari
// tertua lebih dulu. Hari sepi ikut disertakan dengan nilai 0 -- pola
// sama persis DailyEventCounts, cuma dikelompokkan lintas account.
func (s *Store) VendorDailyStats(ctx context.Context, now time.Time, days int) ([]VendorDailyStat, error) {
	now = now.UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	since := startOfToday.AddDate(0, 0, -(days - 1))

	accountRows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AS day, count(*)
		 FROM accounts
		 WHERE created_at >= $1
		 GROUP BY day`, since)
	if err != nil {
		return nil, fmt.Errorf("store: vendor daily new accounts: %w", err)
	}
	defer accountRows.Close()

	newAccounts := make(map[string]int, days)
	for accountRows.Next() {
		var day time.Time
		var n int
		if err := accountRows.Scan(&day, &n); err != nil {
			return nil, fmt.Errorf("store: scan vendor daily new accounts: %w", err)
		}
		newAccounts[day.Format("2006-01-02")] = n
	}
	if err := accountRows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi vendor daily new accounts: %w", err)
	}

	amountRows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', paid_at AT TIME ZONE 'UTC') AS day, sum(requested_amount)
		 FROM invoices
		 WHERE paid_at >= $1 AND status = $2
		 GROUP BY day`, since, InvoiceStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("store: vendor daily paid amount: %w", err)
	}
	defer amountRows.Close()

	amounts := make(map[string]int64, days)
	for amountRows.Next() {
		var day time.Time
		var sum int64
		if err := amountRows.Scan(&day, &sum); err != nil {
			return nil, fmt.Errorf("store: scan vendor daily paid amount: %w", err)
		}
		amounts[day.Format("2006-01-02")] = sum
	}
	if err := amountRows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi vendor daily paid amount: %w", err)
	}

	out := make([]VendorDailyStat, days)
	for i := range days {
		d := since.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		out[i] = VendorDailyStat{Date: d, NewAccounts: newAccounts[key], PaidAmountRp: amounts[key]}
	}
	return out, nil
}
