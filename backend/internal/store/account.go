package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// ErrAccountNotFound dikembalikan bila id/username tidak terdaftar.
var ErrAccountNotFound = errors.New("store: account tidak ditemukan")

// ErrAccountEmailTaken/ErrAccountUsernameTaken dikembalikan CreateAccount
// kalau email/username sudah dipakai account lain (constraint UNIQUE di
// migration 00008_accounts.sql, nama constraint otomatis Postgres untuk
// kolom polos: "accounts_email_key"/"accounts_username_key").
var (
	ErrAccountEmailTaken    = errors.New("store: email sudah dipakai")
	ErrAccountUsernameTaken = errors.New("store: username sudah dipakai")
)

func isAccountUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
	}
	return false
}

// WarningThresholdDays: ambang status berubah dari "active" ke "expiring".
const WarningThresholdDays = 30

type Account struct {
	ID           string
	BusinessName string
	Email        string
	Username     string
	PasswordHash string
	Plan         string
	MaxDevices   int
	AdminStatus  string // "active" | "suspended" | "revoked"
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// VerifyPassword membandingkan password mentah dengan hash tersimpan.
func (a Account) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plaintext)) == nil
}

// DerivedStatus menghitung status yang dikirim ke Customer Dashboard/API --
// pola sama persis licenseserver/store.License.DerivedStatus (dulu), cuma
// sekarang satu query di database yang sama, tidak ada lagi jaringan atau
// grace period.
func (a Account) DerivedStatus(now time.Time) string {
	if a.AdminStatus == "suspended" || a.AdminStatus == "revoked" {
		return a.AdminStatus
	}
	if now.After(a.ExpiresAt) {
		return "expired"
	}
	daysRemaining := int(a.ExpiresAt.Sub(now).Hours() / 24)
	if daysRemaining <= WarningThresholdDays {
		return "expiring"
	}
	return "active"
}

// Operational melaporkan apakah account ini boleh memakai endpoint
// device/admin/API key -- hanya "active" dan "expiring".
func (a Account) Operational(now time.Time) bool {
	status := a.DerivedStatus(now)
	return status == "active" || status == "expiring"
}

type CreateAccountInput struct {
	ID                string
	BusinessName      string
	Email             string
	Username          string
	PlaintextPassword string
	Plan              string
	MaxDevices        int
	ExpiresAt         time.Time
}

// CreateAccount membuat akun baru. Dipanggil dari endpoint vendor --
// password awal digenerate di lapisan HTTP, cuma hash-nya yang sampai ke
// sini, sama pola seperti CreateAPIKey/CreateWebhookEndpoint.
func (s *Store) CreateAccount(ctx context.Context, in CreateAccountInput) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.PlaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password account: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		in.ID, in.BusinessName, in.Email, in.Username, string(hash), in.Plan, in.MaxDevices, in.ExpiresAt)
	if isAccountUniqueViolation(err, "accounts_email_key") {
		return ErrAccountEmailTaken
	}
	if isAccountUniqueViolation(err, "accounts_username_key") {
		return ErrAccountUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("store: create account: %w", err)
	}
	return nil
}

const accountSelectCols = `SELECT id, business_name, email, username, password_hash,
	plan, max_devices, admin_status, expires_at, created_at, updated_at FROM accounts`

type accountScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row accountScanner) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.BusinessName, &a.Email, &a.Username, &a.PasswordHash,
		&a.Plan, &a.MaxDevices, &a.AdminStatus, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// GetAccountByUsername dipakai handleAdminLogin (Customer Dashboard).
func (s *Store) GetAccountByUsername(ctx context.Context, username string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelectCols+` WHERE username = $1`, username)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: select account by username: %w", err)
	}
	return a, nil
}

// GetAccountByID dipakai requireActiveAccount dan endpoint vendor.
func (s *Store) GetAccountByID(ctx context.Context, id string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelectCols+` WHERE id = $1`, id)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: select account by id: %w", err)
	}
	return a, nil
}

// ListAccounts dipakai Vendor Dashboard.
func (s *Store) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelectCols+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list accounts: %w", err)
	}
	defer rows.Close()

	out := make([]Account, 0)
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi accounts: %w", err)
	}
	return out, nil
}

// RenewAccount memperpanjang expires_at. Tidak menyentuh admin_status --
// perpanjang akun yang disuspend TIDAK otomatis mengaktifkannya lagi,
// vendor harus eksplisit SetAccountAdminStatus(active) juga kalau memang mau.
func (s *Store) RenewAccount(ctx context.Context, id string, newExpiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET expires_at = $2, updated_at = now() WHERE id = $1`, id, newExpiresAt)
	if err != nil {
		return fmt.Errorf("store: renew account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// SetAccountAdminStatus dipakai handleVendorSuspendAccount/handleVendorRevokeAccount.
// status harus salah satu dari "active"/"suspended"/"revoked" -- validasi
// nilai yang boleh dilakukan pemanggil di lapisan HTTP, bukan di sini.
func (s *Store) SetAccountAdminStatus(ctx context.Context, id, status string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET admin_status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("store: set account admin_status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}
