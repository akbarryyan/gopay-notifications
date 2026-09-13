// Package store membungkus akses PostgreSQL untuk License Server —
// database gopay_license, terpisah total dari database gopay milik tiap
// instalasi customer (termasuk instalasi Akbar sendiri sebagai customer
// pertamanya di whuzpay.com). Sengaja bukan paket yang sama dengan
// backend/internal/store walau strukturnya serupa: schema-nya berbeda,
// dan mencampur keduanya lewat satu Store generik cuma bikin bingung mana
// yang menyimpan data licensing dan mana yang menyimpan data pembayaran.
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: buka pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Close() { s.pool.Close() }

// randomPrefixedID menghasilkan ID acak, pola sama seperti backend/internal/store
// (crypto/rand + hex, bukan ULID atau library baru).
func randomPrefixedID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: random id: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}
