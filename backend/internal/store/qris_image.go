package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrQRISImageNotFound dikembalikan bila account belum pernah upload QRIS.
var ErrQRISImageNotFound = errors.New("store: qris image tidak ditemukan")

// QRISImage adalah gambar QRIS statis milik satu account -- relasi 1:1,
// upload baru menimpa yang lama (lihat komentar migrasi 00018).
type QRISImage struct {
	AccountID   string
	ImageData   []byte
	ContentType string
	UpdatedAt   time.Time
}

// UpsertQRISImage menyimpan/mengganti gambar QRIS account ini.
func (s *Store) UpsertQRISImage(ctx context.Context, accountID string, data []byte, contentType string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO account_qris_images (account_id, image_data, content_type, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (account_id) DO UPDATE
		 SET image_data = EXCLUDED.image_data, content_type = EXCLUDED.content_type, updated_at = now()`,
		accountID, data, contentType)
	if err != nil {
		return fmt.Errorf("store: upsert qris image: %w", err)
	}
	return nil
}

// GetQRISImage mengambil gambar QRIS account ini.
func (s *Store) GetQRISImage(ctx context.Context, accountID string) (QRISImage, error) {
	var img QRISImage
	img.AccountID = accountID
	err := s.pool.QueryRow(ctx,
		`SELECT image_data, content_type, updated_at FROM account_qris_images WHERE account_id = $1`,
		accountID).Scan(&img.ImageData, &img.ContentType, &img.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return QRISImage{}, ErrQRISImageNotFound
	}
	if err != nil {
		return QRISImage{}, fmt.Errorf("store: get qris image: %w", err)
	}
	return img, nil
}

// DeleteQRISImage menghapus gambar QRIS account ini. Idempotent -- tidak
// error kalau memang belum ada.
func (s *Store) DeleteQRISImage(ctx context.Context, accountID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM account_qris_images WHERE account_id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("store: delete qris image: %w", err)
	}
	return nil
}

// HasQRISImage dipakai handleAdminGetAccount (status ringan, tanpa ambil
// gambar penuh) dan handleCreateInvoice (gerbang sebelum invoice dibuat).
func (s *Store) HasQRISImage(ctx context.Context, accountID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM account_qris_images WHERE account_id = $1)`, accountID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: cek qris image: %w", err)
	}
	return exists, nil
}
