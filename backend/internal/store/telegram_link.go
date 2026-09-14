package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// TelegramLinkTTL: kode "Hubungkan Telegram" berlaku 15 menit -- cukup
// untuk membuka Telegram (termasuk memasangnya dulu di HP), cukup pendek
// supaya link yang tertinggal di riwayat browser tidak bisa dipakai lama.
const TelegramLinkTTL = 15 * time.Minute

// ErrTelegramLinkInvalid mencakup kode tidak dikenal, kedaluwarsa, dan
// sudah dipakai.
var ErrTelegramLinkInvalid = errors.New("store: kode tautan telegram tidak valid")

// CreateTelegramLinkCode menyimpan hash kode baru dan menghapus kode lama
// account itu yang belum dipakai -- cuma link terakhir yang berlaku.
func (s *Store) CreateTelegramLinkCode(ctx context.Context, accountID string, codeHash []byte, now time.Time) (time.Time, error) {
	expires := now.Add(TelegramLinkTTL)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("store: mulai transaksi tautan telegram: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM telegram_link_codes WHERE account_id = $1 AND used_at IS NULL`, accountID); err != nil {
		return time.Time{}, fmt.Errorf("store: hapus kode tautan lama: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO telegram_link_codes (account_id, code_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		accountID, codeHash, expires, now); err != nil {
		return time.Time{}, fmt.Errorf("store: simpan kode tautan: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("store: commit kode tautan: %w", err)
	}
	return expires, nil
}

// ConsumeTelegramLinkCode memakai kode (sekali pakai) untuk menyimpan chat
// id ke account pemiliknya, lalu mengembalikan account itu.
func (s *Store) ConsumeTelegramLinkCode(ctx context.Context, codeHash []byte, chatID string, now time.Time) (Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, fmt.Errorf("store: mulai transaksi tautan telegram: %w", err)
	}
	defer tx.Rollback(ctx)

	var accountID string
	err = tx.QueryRow(ctx,
		`SELECT account_id FROM telegram_link_codes
		 WHERE code_hash = $1 AND used_at IS NULL AND expires_at > $2
		 FOR UPDATE`, codeHash, now).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrTelegramLinkInvalid
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: cari kode tautan: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE telegram_link_codes SET used_at = $2 WHERE code_hash = $1`, codeHash, now); err != nil {
		return Account{}, fmt.Errorf("store: tandai kode tautan: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET telegram_chat_id = $2, updated_at = now() WHERE id = $1`, accountID, chatID); err != nil {
		return Account{}, fmt.Errorf("store: simpan chat id: %w", err)
	}
	acc, err := scanAccount(tx.QueryRow(ctx, accountSelectCols+` WHERE id = $1`, accountID))
	if err != nil {
		return Account{}, fmt.Errorf("store: baca account setelah tautan: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Account{}, fmt.Errorf("store: commit tautan telegram: %w", err)
	}
	return acc, nil
}

// GetTelegramUpdateOffset: 0 bila pengaturan belum pernah disimpan.
func (s *Store) GetTelegramUpdateOffset(ctx context.Context) (int64, error) {
	var offset int64
	err := s.pool.QueryRow(ctx, `SELECT telegram_update_offset FROM notification_settings WHERE id = TRUE`).Scan(&offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: baca offset telegram: %w", err)
	}
	return offset, nil
}

// SetTelegramUpdateOffset menyimpan posisi getUpdates berikutnya.
func (s *Store) SetTelegramUpdateOffset(ctx context.Context, offset int64) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE notification_settings SET telegram_update_offset = $1 WHERE id = TRUE`, offset); err != nil {
		return fmt.Errorf("store: simpan offset telegram: %w", err)
	}
	return nil
}
