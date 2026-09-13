package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrAdminNotFound dikembalikan bila username tidak terdaftar. Sama persis
// pola backend/internal/store — satu akun vendor sudah cukup untuk MVP.
var ErrAdminNotFound = errors.New("store: admin tidak ditemukan")

type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
}

func (s *Store) UpsertAdmin(ctx context.Context, username, plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO admin_users (username, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (username) DO UPDATE
		   SET password_hash = EXCLUDED.password_hash, updated_at = now()`,
		username, string(hash))
	if err != nil {
		return fmt.Errorf("store: upsert admin: %w", err)
	}
	return nil
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (AdminUser, error) {
	var a AdminUser
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM admin_users WHERE username = $1`, username).
		Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUser{}, ErrAdminNotFound
	}
	if err != nil {
		return AdminUser{}, fmt.Errorf("store: select admin: %w", err)
	}
	return a, nil
}

func (a AdminUser) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plaintext)) == nil
}
