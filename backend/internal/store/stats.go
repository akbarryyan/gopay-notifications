package store

import (
	"context"
	"fmt"
	"time"
)

// EventStats adalah ringkasan event untuk kartu Overview di dashboard.
type EventStats struct {
	Today         int
	Last7Days     int
	Total         int
	LatestEventAt *time.Time
}

// EventStats menghitung ringkasan event relatif terhadap now.
//
// "Today" dihitung memakai batas waktu UTC hari ini, bukan window bergulir
// 24 jam — supaya definisinya konsisten dengan yang terlihat orang di
// dashboard ("hari ini" berarti sejak tengah malam, bukan "24 jam terakhir").
func (s *Store) EventStats(ctx context.Context, accountID string, now time.Time) (EventStats, error) {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := now.AddDate(0, 0, -7)

	var st EventStats
	err := s.pool.QueryRow(ctx,
		`SELECT
		   count(*) FILTER (WHERE ingested_at >= $1),
		   count(*) FILTER (WHERE ingested_at >= $2),
		   count(*),
		   max(ingested_at)
		 FROM notification_events
		 WHERE account_id = $3`,
		startOfDay, sevenDaysAgo, accountID).
		Scan(&st.Today, &st.Last7Days, &st.Total, &st.LatestEventAt)
	if err != nil {
		return EventStats{}, fmt.Errorf("store: event stats: %w", err)
	}
	return st, nil
}

// DailyEventCount adalah satu titik pada grafik tren Overview -- dua
// series sekaligus (jumlah event dan nominal diterima) supaya dashboard
// bisa menampilkan grafik biaxial, dua skala yang jauh berbeda pada satu
// sumbu waktu yang sama.
type DailyEventCount struct {
	Date         time.Time // tengah malam UTC hari itu
	Count        int
	PaidAmountRp int64 // jumlah invoice yang lunas (paid_at) pada hari itu, dalam rupiah
}

// DailyEventCounts mengembalikan jumlah event DAN total nominal lunas per
// hari untuk `days` hari terakhir (termasuk hari ini), hari tertua lebih
// dulu. Hari tanpa event/pembayaran ikut disertakan dengan nilai 0 —
// dihitung di Go, bukan mengandalkan SQL generate seri tanggal — supaya
// grafik di dashboard tidak bolong pada hari sepi.
//
// Jumlah event dikelompokkan lewat ingested_at (bukan received_at yang
// berasal dari perangkat), selaras dengan EventStats — kejadian dihitung
// menurut kapan ia benar-benar sampai di server, bukan jam HP pengirim yang
// bisa meleset. Nominal lunas dikelompokkan lewat paid_at (invoice bisa
// dibuat satu hari dan baru lunas hari lain).
func (s *Store) DailyEventCounts(ctx context.Context, accountID string, now time.Time, days int) ([]DailyEventCount, error) {
	now = now.UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	since := startOfToday.AddDate(0, 0, -(days - 1))

	rows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', ingested_at AT TIME ZONE 'UTC') AS day, count(*)
		 FROM notification_events
		 WHERE ingested_at >= $1 AND account_id = $2
		 GROUP BY day`, since, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: daily event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int, days)
	for rows.Next() {
		var day time.Time
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, fmt.Errorf("store: scan daily event count: %w", err)
		}
		counts[day.Format("2006-01-02")] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi daily event counts: %w", err)
	}

	amountRows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', paid_at AT TIME ZONE 'UTC') AS day, sum(requested_amount)
		 FROM invoices
		 WHERE paid_at >= $1 AND status = $2 AND account_id = $3
		 GROUP BY day`, since, InvoiceStatusPaid, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: daily paid amount: %w", err)
	}
	defer amountRows.Close()

	amounts := make(map[string]int64, days)
	for amountRows.Next() {
		var day time.Time
		var sum int64
		if err := amountRows.Scan(&day, &sum); err != nil {
			return nil, fmt.Errorf("store: scan daily paid amount: %w", err)
		}
		amounts[day.Format("2006-01-02")] = sum
	}
	if err := amountRows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi daily paid amount: %w", err)
	}

	out := make([]DailyEventCount, days)
	for i := range days {
		d := since.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		out[i] = DailyEventCount{Date: d, Count: counts[key], PaidAmountRp: amounts[key]}
	}
	return out, nil
}
