package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrVendorAdminNotFound dikembalikan bila username tidak terdaftar.
var ErrVendorAdminNotFound = errors.New("store: vendor admin tidak ditemukan")

type VendorAdmin struct {
	ID           int64
	Username     string
	PasswordHash string
}

// UpsertVendorAdmin membuat atau memperbarui password vendor admin (Akbar).
// Pola sama seperti UpsertAdmin lama -- dipakai cmd/admintool.
func (s *Store) UpsertVendorAdmin(ctx context.Context, username, plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password vendor admin: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO vendor_admins (username, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (username) DO UPDATE
		   SET password_hash = EXCLUDED.password_hash, updated_at = now()`,
		username, string(hash))
	if err != nil {
		return fmt.Errorf("store: upsert vendor admin: %w", err)
	}
	return nil
}

// GetVendorAdminByUsername dipakai handleVendorLogin.
func (s *Store) GetVendorAdminByUsername(ctx context.Context, username string) (VendorAdmin, error) {
	var v VendorAdmin
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM vendor_admins WHERE username = $1`, username).
		Scan(&v.ID, &v.Username, &v.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorAdmin{}, ErrVendorAdminNotFound
	}
	if err != nil {
		return VendorAdmin{}, fmt.Errorf("store: select vendor admin: %w", err)
	}
	return v, nil
}

// VerifyPassword membandingkan password mentah dengan hash tersimpan.
func (v VendorAdmin) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(v.PasswordHash), []byte(plaintext)) == nil
}
