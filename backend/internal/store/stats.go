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

// DailyEventCount adalah satu titik pada grafik tren Overview.
type DailyEventCount struct {
	Date  time.Time // tengah malam UTC hari itu
	Count int
}

// DailyEventCounts mengembalikan jumlah event per hari untuk `days` hari
// terakhir (termasuk hari ini), hari tertua lebih dulu. Hari tanpa event ikut
// disertakan dengan Count 0 — dihitung di Go, bukan mengandalkan SQL generate
// seri tanggal — supaya grafik di dashboard tidak bolong pada hari sepi.
//
// Dikelompokkan lewat ingested_at (bukan received_at yang berasal dari
// perangkat), selaras dengan EventStats — kejadian dihitung menurut kapan ia
// benar-benar sampai di server, bukan jam HP pengirim yang bisa meleset.
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

	out := make([]DailyEventCount, days)
	for i := range days {
		d := since.AddDate(0, 0, i)
		out[i] = DailyEventCount{Date: d, Count: counts[d.Format("2006-01-02")]}
	}
	return out, nil
}
