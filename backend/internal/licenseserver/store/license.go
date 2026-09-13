package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrLicenseNotFound = errors.New("store: license tidak ditemukan atau key salah")
)

// AdminStatus adalah status administratif tersimpan — cuma tiga nilai.
// Status turunan waktu (expiring/expired) dihitung saat dibaca, bukan
// disimpan — lihat DerivedStatus.
type AdminStatus string

const (
	AdminStatusActive    AdminStatus = "active"
	AdminStatusSuspended AdminStatus = "suspended"
	AdminStatusRevoked   AdminStatus = "revoked"
)

type License struct {
	ID                      string
	CustomerID              string
	Plan                    string
	Status                  AdminStatus
	MaxDevices              int
	ProductionInstallations int
	UATInstallations        int
	IssuedAt                time.Time
	ExpiresAt               time.Time
	CreatedAt               time.Time
}

func NewLicenseID() (string, error) { return randomPrefixedID("lic_") }

const keyAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // tanpa 0/O/1/I, gampang dibaca manual

// GenerateLicenseKey membuat license key mentah baru dan hash SHA-256-nya.
// Formatnya "PB-<PLAN>-XXXX-XXXX-XXXX" (lihat §6 docs/license-spec.md).
// SHA-256, bukan bcrypt — key ini bukan password manusia, digenerate lewat
// crypto/rand dengan entropi tinggi (pola sama seperti API key di backend
// customer).
func GenerateLicenseKey(plan string) (raw string, hash []byte, err error) {
	planPart := strings.ToUpper(strings.ReplaceAll(plan, " ", ""))
	segments := make([]string, 3)
	for i := range segments {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return "", nil, fmt.Errorf("store: random key segment: %w", err)
		}
		seg := make([]byte, 4)
		for j, v := range b {
			seg[j] = keyAlphabet[int(v)%len(keyAlphabet)]
		}
		segments[i] = string(seg)
	}
	raw = fmt.Sprintf("PB-%s-%s-%s-%s", planPart, segments[0], segments[1], segments[2])
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

type CreateLicenseInput struct {
	ID                      string
	CustomerID              string
	KeyHash                 []byte
	Plan                    string
	MaxDevices              int
	ProductionInstallations int
	UATInstallations        int
	IssuedAt                time.Time
	ExpiresAt               time.Time
}

func (s *Store) CreateLicense(ctx context.Context, in CreateLicenseInput) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO licenses
		   (id, customer_id, key_hash, plan, status, max_devices,
		    production_installations, uat_installations, issued_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8, $9)`,
		in.ID, in.CustomerID, in.KeyHash, in.Plan, in.MaxDevices,
		in.ProductionInstallations, in.UATInstallations, in.IssuedAt, in.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store: create license: %w", err)
	}
	return nil
}

func scanLicense(row pgx.Row) (License, error) {
	var l License
	err := row.Scan(&l.ID, &l.CustomerID, &l.Plan, &l.Status, &l.MaxDevices,
		&l.ProductionInstallations, &l.UATInstallations, &l.IssuedAt, &l.ExpiresAt, &l.CreatedAt)
	return l, err
}

const licenseColumns = `id, customer_id, plan, status, max_devices,
	production_installations, uat_installations, issued_at, expires_at, created_at`

func (s *Store) GetLicenseByID(ctx context.Context, id string) (License, error) {
	l, err := scanLicense(s.pool.QueryRow(ctx,
		`SELECT `+licenseColumns+` FROM licenses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return License{}, ErrLicenseNotFound
	}
	if err != nil {
		return License{}, fmt.Errorf("store: get license: %w", err)
	}
	return l, nil
}

// GetLicenseByKey mencari license dari hash key mentah — dipakai /activate
// dan /validate. Error yang sama untuk "key tak dikenal" maupun error DB
// lain yang tersamar sebagai not-found tidak dibedakan di lapisan HTTP:
// pemanggil yang salah key tidak boleh bisa membedakan key salah dari
// license yang memang tidak ada.
func (s *Store) GetLicenseByKeyHash(ctx context.Context, keyHash []byte) (License, error) {
	l, err := scanLicense(s.pool.QueryRow(ctx,
		`SELECT `+licenseColumns+` FROM licenses WHERE key_hash = $1`, keyHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return License{}, ErrLicenseNotFound
	}
	if err != nil {
		return License{}, fmt.Errorf("store: get license by key: %w", err)
	}
	return l, nil
}

func (s *Store) ListLicensesByCustomer(ctx context.Context, customerID string) ([]License, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+licenseColumns+` FROM licenses WHERE customer_id = $1 ORDER BY created_at DESC`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("store: list licenses: %w", err)
	}
	defer rows.Close()

	var out []License
	for rows.Next() {
		l, err := scanLicense(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan license: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// RenewLicense mengubah expires_at. Tidak menyentuh status administratif —
// me-renew license yang suspended/revoked tidak diam-diam mengaktifkannya
// lagi, itu perlu aksi eksplisit (SetLicenseStatus) terpisah.
func (s *Store) RenewLicense(ctx context.Context, id string, newExpiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE licenses SET expires_at = $2, updated_at = now() WHERE id = $1`,
		id, newExpiresAt)
	if err != nil {
		return fmt.Errorf("store: renew license: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLicenseNotFound
	}
	return nil
}

func (s *Store) SetLicenseStatus(ctx context.Context, id string, status AdminStatus) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE licenses SET status = $2, updated_at = now() WHERE id = $1`, id, string(status))
	if err != nil {
		return fmt.Errorf("store: set license status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLicenseNotFound
	}
	return nil
}

// DerivedStatus menghitung status yang dikirim ke client (§3 spec):
// suspended/revoked langsung dari kolom, else active/expiring/expired
// ditentukan dari expires_at relatif terhadap now.
func (l License) DerivedStatus(now time.Time, warningThresholdDays int) string {
	switch l.Status {
	case AdminStatusSuspended:
		return "suspended"
	case AdminStatusRevoked:
		return "revoked"
	}
	expiresEndOfDay := l.ExpiresAt.Add(24*time.Hour - time.Second)
	if now.After(expiresEndOfDay) {
		return "expired"
	}
	daysRemaining := int(expiresEndOfDay.Sub(now).Hours() / 24)
	if daysRemaining <= warningThresholdDays {
		return "expiring"
	}
	return "active"
}
