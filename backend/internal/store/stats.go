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
func (s *Store) EventStats(ctx context.Context, now time.Time) (EventStats, error) {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := now.AddDate(0, 0, -7)

	var st EventStats
	err := s.pool.QueryRow(ctx,
		`SELECT
		   count(*) FILTER (WHERE ingested_at >= $1),
		   count(*) FILTER (WHERE ingested_at >= $2),
		   count(*),
		   max(ingested_at)
		 FROM notification_events`,
		startOfDay, sevenDaysAgo).
		Scan(&st.Today, &st.Last7Days, &st.Total, &st.LatestEventAt)
	if err != nil {
		return EventStats{}, fmt.Errorf("store: event stats: %w", err)
	}
	return st, nil
}
