package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrAPIKeyNotFound dikembalikan bila key tidak dikenal atau sudah dicabut.
// Disatukan sengaja — pemanggil (verifikasi key) tidak boleh bisa membedakan
// "key salah" dari "key benar tapi dicabut" lewat error yang berbeda, sama
// seperti username-tak-dikenal disamakan dengan password salah di login admin.
var ErrAPIKeyNotFound = errors.New("store: api key tidak ditemukan atau sudah dicabut")

type APIKey struct {
	ID        string
	AccountID string
	Name      string
	CreatedAt time.Time
	RevokedAt *time.Time
}

// NewAPIKeyID menghasilkan ID key baru, pola sama seperti device_id di
// cmd/devicetool: crypto/rand + hex, bukan ULID atau library baru.
func NewAPIKeyID() (string, error) {
	return randomPrefixedID("key_")
}

func randomPrefixedID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: random id: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}

// GenerateAPIKeySecret membuat key mentah baru (ditampilkan sekali ke
// pemanggil) dan hash SHA-256-nya (yang disimpan). SHA-256, bukan bcrypt —
// key ini digenerate backend sendiri lewat crypto/rand, berentropi tinggi,
// bukan password manusia yang butuh hash lambat untuk menahan brute-force.
func GenerateAPIKeySecret() (raw string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("store: random secret: %w", err)
	}
	raw = "sk_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

// CreateAPIKey menyimpan API key baru. Key mentahnya sendiri tidak pernah
// disimpan, hanya hash-nya — dikembalikan lewat tipe hasil hanya oleh
// pemanggil di lapisan HTTP yang baru saja men-generate-nya.
func (s *Store) CreateAPIKey(ctx context.Context, accountID, id, name string, keyHash []byte) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO api_keys (id, account_id, name, key_hash) VALUES ($1, $2, $3, $4)`,
		id, accountID, name, keyHash)
	if err != nil {
		return fmt.Errorf("store: create api key: %w", err)
	}
	return nil
}

// ListAPIKeys mengembalikan seluruh key milik satu account, terbaru lebih
// dulu. Tidak pernah menyertakan key_hash — dashboard tidak boleh punya cara
// membocorkannya.
func (s *Store) ListAPIKeys(ctx context.Context, accountID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, account_id, name, created_at, revoked_at FROM api_keys
		 WHERE account_id = $1 ORDER BY created_at DESC`, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: list api keys: %w", err)
	}
	defer rows.Close()

	out := make([]APIKey, 0)
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.AccountID, &k.Name, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, fmt.Errorf("store: scan api key: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi api keys: %w", err)
	}
	return out, nil
}

// RevokeAPIKey mencabut key milik account ini. Idempotent: mencabut yang
// sudah dicabut tetap sukses tanpa mengubah revoked_at semula. Key milik
// account lain diperlakukan sama seperti tidak ada (ErrAPIKeyNotFound).
func (s *Store) RevokeAPIKey(ctx context.Context, accountID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND account_id = $2 AND revoked_at IS NULL`,
		id, accountID)
	if err != nil {
		return fmt.Errorf("store: revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Bisa jadi sudah dicabut (idempotent, bukan error) atau memang
		// tidak ada/bukan milik account ini — bedakan lewat existence check
		// supaya 404 tetap benar.
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = $1 AND account_id = $2)`, id, accountID).
			Scan(&exists); err != nil {
			return fmt.Errorf("store: cek api key: %w", err)
		}
		if !exists {
			return ErrAPIKeyNotFound
		}
	}
	return nil
}

// VerifyAPIKey mencari key aktif (belum dicabut) berdasarkan hash key
// mentah. Dipakai middleware auth endpoint invoice. key_hash tetap unik
// global (bukan di-scope per account) -- pemanggil (requireAPIKey) yang
// membaca AccountID dari hasilnya buat tahu pemiliknya.
func (s *Store) VerifyAPIKey(ctx context.Context, rawKey string) (APIKey, error) {
	sum := sha256.Sum256([]byte(rawKey))
	var k APIKey
	err := s.pool.QueryRow(ctx,
		`SELECT id, account_id, name, created_at, revoked_at FROM api_keys
		 WHERE key_hash = $1 AND revoked_at IS NULL`, sum[:]).
		Scan(&k.ID, &k.AccountID, &k.Name, &k.CreatedAt, &k.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, ErrAPIKeyNotFound
	}
	if err != nil {
		return APIKey{}, fmt.Errorf("store: verify api key: %w", err)
	}
	return k, nil
}
