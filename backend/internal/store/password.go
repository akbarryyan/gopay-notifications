package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetTTL: link reset berlaku 30 menit. Cukup untuk membuka email
// yang masuk agak telat, cukup pendek supaya link yang tertinggal di kotak
// masuk tidak bisa dipakai berhari-hari kemudian.
const PasswordResetTTL = 30 * time.Minute

// ErrResetTokenInvalid mencakup token tidak dikenal, kedaluwarsa, dan
// sudah dipakai -- sengaja tidak dibedakan ke pemanggil HTTP.
var ErrResetTokenInvalid = errors.New("store: token reset tidak valid")

// GetAccountByEmail dipakai permintaan reset password. Tidak peka huruf
// besar/kecil: orang mengetik alamatnya sendiri tidak selalu persis sama
// dengan saat mendaftar.
func (s *Store) GetAccountByEmail(ctx context.Context, email string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelectCols+` WHERE lower(email) = lower($1)`, email)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: select account by email: %w", err)
	}
	return a, nil
}

// PasswordResetRequestedSince melaporkan apakah account ini sudah meminta
// reset sejak waktu tertentu -- dipakai menahan spam email reset ke satu
// alamat walau permintaannya datang dari banyak IP.
func (s *Store) PasswordResetRequestedSince(ctx context.Context, accountID string, since time.Time) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM password_reset_tokens WHERE account_id = $1 AND created_at >= $2)`,
		accountID, since).Scan(&exists); err != nil {
		return false, fmt.Errorf("store: cek permintaan reset: %w", err)
	}
	return exists, nil
}

// CreatePasswordResetToken menyimpan hash token baru dan menghapus token
// lama yang belum dipakai -- cuma link dari email TERAKHIR yang berlaku.
func (s *Store) CreatePasswordResetToken(ctx context.Context, accountID string, tokenHash []byte, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: mulai transaksi reset: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE account_id = $1 AND used_at IS NULL`, accountID); err != nil {
		return fmt.Errorf("store: hapus token reset lama: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO password_reset_tokens (account_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4)`,
		accountID, tokenHash, now.Add(PasswordResetTTL), now); err != nil {
		return fmt.Errorf("store: simpan token reset: %w", err)
	}
	return tx.Commit(ctx)
}

// ResetPasswordWithToken memakai token (sekali pakai) untuk mengganti
// password, lalu mengembalikan account pemiliknya.
func (s *Store) ResetPasswordWithToken(ctx context.Context, tokenHash []byte, newPassword string, now time.Time) (Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, fmt.Errorf("store: hash password baru: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, fmt.Errorf("store: mulai transaksi reset: %w", err)
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE: dua request dengan token yang sama bersamaan tidak boleh
	// sama-sama lolos.
	var accountID string
	err = tx.QueryRow(ctx,
		`SELECT account_id FROM password_reset_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		 FOR UPDATE`, tokenHash, now).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrResetTokenInvalid
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: cari token reset: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $2 WHERE token_hash = $1`, tokenHash, now); err != nil {
		return Account{}, fmt.Errorf("store: tandai token reset: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE account_id = $1 AND used_at IS NULL`, accountID); err != nil {
		return Account{}, fmt.Errorf("store: hapus token reset lain: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET password_hash = $2, password_changed_at = $3, updated_at = now() WHERE id = $1`,
		accountID, string(hash), now); err != nil {
		return Account{}, fmt.Errorf("store: ganti password: %w", err)
	}

	acc, err := scanAccount(tx.QueryRow(ctx, accountSelectCols+` WHERE id = $1`, accountID))
	if err != nil {
		return Account{}, fmt.Errorf("store: baca account setelah reset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Account{}, fmt.Errorf("store: commit reset: %w", err)
	}
	return acc, nil
}

// ChangeAccountPassword dipakai ganti password dari Settings (password lama
// sudah diverifikasi pemanggil). Token reset yang belum dipakai ikut
// dihapus: link lama di kotak masuk tidak boleh bisa membatalkan password
// yang baru saja dipilih.
func (s *Store) ChangeAccountPassword(ctx context.Context, accountID, newPassword string, now time.Time) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password baru: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: mulai transaksi ganti password: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE accounts SET password_hash = $2, password_changed_at = $3, updated_at = now() WHERE id = $1`,
		accountID, string(hash), now)
	if err != nil {
		return fmt.Errorf("store: ganti password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE account_id = $1 AND used_at IS NULL`, accountID); err != nil {
		return fmt.Errorf("store: hapus token reset: %w", err)
	}
	return tx.Commit(ctx)
}

// UpdateAccountProfile mengubah nama bisnis dan email yang diisi customer
// sendiri di Settings.
// UpdateAccountProfile mengubah nama bisnis dan email. Mengganti ke email
// yang BEDA dari yang tersimpan otomatis mengosongkan email_verified_at --
// alamat baru itu belum pernah dibuktikan bisa menerima email dari kami.
func (s *Store) UpdateAccountProfile(ctx context.Context, accountID, businessName, email string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts
		 SET business_name = $2, email = $3, updated_at = now(),
		     email_verified_at = CASE WHEN email = $3 THEN email_verified_at ELSE NULL END
		 WHERE id = $1`,
		accountID, businessName, email)
	if isAccountUniqueViolation(err, "accounts_email_key") {
		return ErrAccountEmailTaken
	}
	if err != nil {
		return fmt.Errorf("store: ubah profil account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}
