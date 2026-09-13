package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInstallationLimitReached = errors.New("store: kuota installation environment ini sudah penuh")
	ErrInstallationNotFound     = errors.New("store: installation tidak ditemukan")
	ErrLicenseNotActive         = errors.New("store: license tidak aktif")
)

type Installation struct {
	ID             string
	LicenseID      string
	Environment    string
	ProductVersion string
	ActivatedAt    time.Time
	ReleasedAt     *time.Time
}

func NewInstallationID() (string, error) { return randomPrefixedID("inst_") }

// Activate membuat installation baru untuk license, mengecek kuota
// environment secara atomik lewat row lock — mencegah dua aktivasi
// bersamaan sama-sama lolos kuota (pola sama seperti UPDATE...RETURNING
// yang dipakai backend/internal/store untuk mencegah race sejenis).
func (s *Store) Activate(ctx context.Context, license License, environment, productVersion string) (Installation, error) {
	if license.Status != AdminStatusActive {
		return Installation{}, ErrLicenseNotActive
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Installation{}, fmt.Errorf("store: begin activate: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Row lock supaya dua /activate untuk license yang sama tidak
	// sama-sama membaca hitungan kuota "belum penuh" lalu berdua insert.
	var quota int
	col := "production_installations"
	if environment == "uat" {
		col = "uat_installations"
	}
	if err := tx.QueryRow(ctx,
		`SELECT `+col+` FROM licenses WHERE id = $1 FOR UPDATE`, license.ID).Scan(&quota); err != nil {
		return Installation{}, fmt.Errorf("store: lock license: %w", err)
	}

	var used int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM installations
		 WHERE license_id = $1 AND environment = $2 AND released_at IS NULL`,
		license.ID, environment).Scan(&used); err != nil {
		return Installation{}, fmt.Errorf("store: hitung installation: %w", err)
	}
	if used >= quota {
		return Installation{}, ErrInstallationLimitReached
	}

	id, err := NewInstallationID()
	if err != nil {
		return Installation{}, err
	}
	var inst Installation
	err = tx.QueryRow(ctx,
		`INSERT INTO installations (id, license_id, environment, product_version)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, license_id, environment, product_version, activated_at, released_at`,
		id, license.ID, environment, productVersion).
		Scan(&inst.ID, &inst.LicenseID, &inst.Environment, &inst.ProductVersion,
			&inst.ActivatedAt, &inst.ReleasedAt)
	if err != nil {
		return Installation{}, fmt.Errorf("store: insert installation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Installation{}, fmt.Errorf("store: commit activate: %w", err)
	}
	return inst, nil
}

// GetActiveInstallation mengambil installation yang MASIH terikat (belum
// di-release) ke license tertentu — dipakai /validate untuk memastikan
// installation_id yang dikirim backend customer benar-benar masih sah,
// bukan sekadar installation_id yang PERNAH ada (yang sudah di-reset vendor
// harus ditolak, memicu backend customer menghapus local state-nya).
func (s *Store) GetActiveInstallation(ctx context.Context, licenseID, installationID string) (Installation, error) {
	var inst Installation
	err := s.pool.QueryRow(ctx,
		`SELECT id, license_id, environment, product_version, activated_at, released_at
		 FROM installations
		 WHERE id = $1 AND license_id = $2 AND released_at IS NULL`,
		installationID, licenseID).
		Scan(&inst.ID, &inst.LicenseID, &inst.Environment, &inst.ProductVersion,
			&inst.ActivatedAt, &inst.ReleasedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Installation{}, ErrInstallationNotFound
	}
	if err != nil {
		return Installation{}, fmt.Errorf("store: get installation: %w", err)
	}
	return inst, nil
}

func (s *Store) ListInstallationsByLicense(ctx context.Context, licenseID string) ([]Installation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, license_id, environment, product_version, activated_at, released_at
		 FROM installations WHERE license_id = $1 ORDER BY activated_at DESC`, licenseID)
	if err != nil {
		return nil, fmt.Errorf("store: list installations: %w", err)
	}
	defer rows.Close()

	var out []Installation
	for rows.Next() {
		var inst Installation
		if err := rows.Scan(&inst.ID, &inst.LicenseID, &inst.Environment, &inst.ProductVersion,
			&inst.ActivatedAt, &inst.ReleasedAt); err != nil {
			return nil, fmt.Errorf("store: scan installation: %w", err)
		}
		out = append(out, inst)
	}
	return out, rows.Err()
}

// ResetInstallation membuka kuota lagi ("Reset Installation", §22
// license-spec.md) — dipakai saat customer migrasi VPS. Mengembalikan
// ErrInstallationNotFound baik untuk ID yang tidak ada maupun yang sudah
// di-release sebelumnya — pemanggil (vendor dashboard) menampilkan pesan
// yang sama untuk keduanya, tidak perlu membedakan.
func (s *Store) ResetInstallation(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE installations SET released_at = now() WHERE id = $1 AND released_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("store: reset installation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInstallationNotFound
	}
	return nil
}
