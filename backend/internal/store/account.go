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
	// TelegramChatID opsional -- diisi customer sendiri di Settings
	// dashboard mereka. Nil berarti pengingat cuma lewat email.
	TelegramChatID *string
	// PasswordChangedAt nil berarti password belum pernah diganti sejak
	// account dibuat. Sesi yang diterbitkan sebelum waktu ini ditolak.
	PasswordChangedAt *time.Time
	// EmailVerifiedAt nil berarti belum diverifikasi -- pengingat, bukan
	// gerbang: account tetap bisa dipakai penuh selama menunggu.
	EmailVerifiedAt *time.Time
}

// VerifyPassword membandingkan password mentah dengan hash tersimpan.
func (a Account) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plaintext)) == nil
}

// DerivedStatus menghitung status yang dikirim ke Customer Dashboard/API --
// pola sama persis licenseserver/store.License.DerivedStatus (dulu), cuma
// sekarang satu query di database yang sama, tidak ada lagi jaringan atau
// grace period.
//
// SENGAJA cuma 4 nilai (active/expired/suspended/revoked) -- status
// "expiring" (peringatan dini 30 hari sebelum expires_at) pernah ada, tapi
// dicabut atas permintaan eksplisit Akbar: sisa waktu berminggu-minggu
// terasa membingungkan ditandai "akan berakhir", dan itu memang bukan
// keadaan yang butuh tindakan customer/vendor. Pengingat aktif ke customer
// (email/Telegram) tetap ada lewat internal/reminder, 7 hari sebelum
// benar-benar habis -- itu jalur yang tepat untuk "hampir habis", bukan
// status pasif di dashboard.
func (a Account) DerivedStatus(now time.Time) string {
	if a.AdminStatus == "suspended" || a.AdminStatus == "revoked" {
		return a.AdminStatus
	}
	if now.After(a.ExpiresAt) {
		return "expired"
	}
	return "active"
}

// Operational melaporkan apakah account ini boleh memakai endpoint
// device/admin/API key -- hanya "active".
func (a Account) Operational(now time.Time) bool {
	return a.DerivedStatus(now) == "active"
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
	plan, max_devices, admin_status, expires_at, created_at, updated_at, telegram_chat_id, password_changed_at, email_verified_at FROM accounts`

type accountScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row accountScanner) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.BusinessName, &a.Email, &a.Username, &a.PasswordHash,
		&a.Plan, &a.MaxDevices, &a.AdminStatus, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt,
		&a.TelegramChatID, &a.PasswordChangedAt, &a.EmailVerifiedAt)
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

// SetAccountTelegramChatID menyimpan (atau menghapus, bila nil) chat id
// Telegram yang diisi customer sendiri di Settings dashboard mereka.
func (s *Store) SetAccountTelegramChatID(ctx context.Context, id string, chatID *string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET telegram_chat_id = $2, updated_at = now() WHERE id = $1`, id, chatID)
	if err != nil {
		return fmt.Errorf("store: set telegram chat id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// AccountsNeedingExpiryReminder mengembalikan akun yang perlu diingatkan
// bahwa masa aktifnya segera berakhir: masih operasional (bukan
// suspended/revoked), berakhir dalam `withinDays` hari ke depan, dan
// pengingat untuk NILAI expires_at itu belum pernah dikirim.
//
// Perbandingan `expiry_reminder_sent_for IS DISTINCT FROM expires_at`
// (bukan `<`, bukan IS NULL saja) yang membuat perpanjangan otomatis
// membuka pengingat periode berikutnya -- lihat alasannya di migrasi
// 00010_expiry_reminder.sql.
//
// Akun yang SUDAH kedaluwarsa tidak ikut: pengingat yang datang setelah
// layanan berhenti bukan pengingat, cuma pemberitahuan yang terlambat.
func (s *Store) AccountsNeedingExpiryReminder(ctx context.Context, now time.Time, withinDays int) ([]Account, error) {
	deadline := now.AddDate(0, 0, withinDays)
	rows, err := s.pool.Query(ctx,
		accountSelectCols+`
		 WHERE admin_status = 'active'
		   AND expires_at > $1
		   AND expires_at <= $2
		   AND expiry_reminder_sent_for IS DISTINCT FROM expires_at
		 ORDER BY expires_at ASC`, now, deadline)
	if err != nil {
		return nil, fmt.Errorf("store: akun yang perlu pengingat kedaluwarsa: %w", err)
	}
	defer rows.Close()

	out := make([]Account, 0)
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan akun pengingat: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi akun pengingat: %w", err)
	}
	return out, nil
}

// MarkExpiryReminderSent menandai pengingat untuk periode `expiresAt`
// sudah terkirim. `AND expires_at = $2` menjaga dari kasus akun diperpanjang
// tepat di antara pembacaan dan penandaan: kalau begitu, tidak ada baris
// yang tersentuh dan pengingat periode baru tetap akan dikirim nanti.
func (s *Store) MarkExpiryReminderSent(ctx context.Context, id string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE accounts SET expiry_reminder_sent_for = $2 WHERE id = $1 AND expires_at = $2`,
		id, expiresAt)
	if err != nil {
		return fmt.Errorf("store: tandai pengingat kedaluwarsa terkirim: %w", err)
	}
	return nil
}

// SetAccountPlan mengubah plan (dan kuota max_devices yang mengikutinya)
// akun yang sudah ada -- dipakai vendor untuk upgrade/downgrade tanpa harus
// membuat ulang akun. plan dan maxDevices divalidasi/dipetakan dari
// planPresets di lapisan HTTP (sama seperti handleVendorCreateAccount),
// bukan di sini. Tidak menyentuh expires_at atau admin_status sama sekali.
func (s *Store) SetAccountPlan(ctx context.Context, id, plan string, maxDevices int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET plan = $2, max_devices = $3, updated_at = now() WHERE id = $1`,
		id, plan, maxDevices)
	if err != nil {
		return fmt.Errorf("store: set account plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}
