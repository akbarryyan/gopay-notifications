package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// EmailVerificationTTL: link berlaku 24 jam -- lebih longgar dari reset
// password (30 menit) karena memverifikasi email bukan tindakan sensitif
// waktu, cuma pembuktian alamatnya benar-benar bisa menerima email.
const EmailVerificationTTL = 24 * time.Hour

var ErrEmailVerificationInvalid = errors.New("store: token verifikasi email tidak valid")

// CreateEmailVerificationToken menyimpan hash token baru dan menghapus
// token lama account itu yang belum dipakai -- cuma link terakhir yang
// berlaku, pola sama dengan CreatePasswordResetToken.
func (s *Store) CreateEmailVerificationToken(ctx context.Context, accountID string, tokenHash []byte, now time.Time) (time.Time, error) {
	expires := now.Add(EmailVerificationTTL)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("store: mulai transaksi verifikasi email: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM email_verification_tokens WHERE account_id = $1 AND used_at IS NULL`, accountID); err != nil {
		return time.Time{}, fmt.Errorf("store: hapus token verifikasi lama: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO email_verification_tokens (account_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		accountID, tokenHash, expires, now); err != nil {
		return time.Time{}, fmt.Errorf("store: simpan token verifikasi: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("store: commit token verifikasi: %w", err)
	}
	return expires, nil
}

// ConsumeEmailVerificationToken memakai token (sekali pakai) untuk
// menandai email account itu terverifikasi, lalu mengembalikan account-nya.
//
// Aman dipakai walau email sempat berganti setelah link dikirim: token
// menunjuk ke account_id, dan yang ditandai adalah email yang tersimpan
// SEKARANG -- kalau memang berbeda dari yang dikirimi link, verifikasi itu
// otomatis tidak berarti apa-apa (UpdateAccountProfile sudah mengosongkan
// email_verified_at saat email berganti, jadi tetap butuh verifikasi baru).
func (s *Store) ConsumeEmailVerificationToken(ctx context.Context, tokenHash []byte, now time.Time) (Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, fmt.Errorf("store: mulai transaksi verifikasi email: %w", err)
	}
	defer tx.Rollback(ctx)

	var accountID string
	err = tx.QueryRow(ctx,
		`SELECT account_id FROM email_verification_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		 FOR UPDATE`, tokenHash, now).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrEmailVerificationInvalid
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: cari token verifikasi: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE email_verification_tokens SET used_at = $2 WHERE token_hash = $1`, tokenHash, now); err != nil {
		return Account{}, fmt.Errorf("store: tandai token verifikasi: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET email_verified_at = $2, updated_at = now()
		 WHERE id = $1 AND email_verified_at IS NULL`, accountID, now); err != nil {
		return Account{}, fmt.Errorf("store: tandai email terverifikasi: %w", err)
	}
	acc, err := scanAccount(tx.QueryRow(ctx, accountSelectCols+` WHERE id = $1`, accountID))
	if err != nil {
		return Account{}, fmt.Errorf("store: baca account setelah verifikasi: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Account{}, fmt.Errorf("store: commit verifikasi email: %w", err)
	}
	return acc, nil
}

// EmailVerificationRequestedSince: dipakai membatasi kirim ulang -- pola
// sama dengan PasswordResetRequestedSince.
func (s *Store) EmailVerificationRequestedSince(ctx context.Context, accountID string, since time.Time) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM email_verification_tokens WHERE account_id = $1 AND created_at >= $2)`,
		accountID, since).Scan(&exists); err != nil {
		return false, fmt.Errorf("store: cek permintaan verifikasi: %w", err)
	}
	return exists, nil
}
