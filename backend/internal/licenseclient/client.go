// Package licenseclient memanggil License Server dari backend customer:
// aktivasi sekali di awal (dipicu LICENSE_KEY di .env, bukan panggilan
// jaringan tiap request) dan validasi periodik (goroutine di
// cmd/server/main.go). Hasil yang berhasil ditulis ke local license state
// (internal/licensecheck) — paket ini tidak pernah membaca file itu untuk
// memutuskan boleh/tidaknya request lain, itu tugas requireLicense.
//
// Lihat docs/superpowers/specs/2026-09-13-online-license-platform-design.md §5-6.
package licenseclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/version"
)

type Client struct {
	httpClient    *http.Client
	serverURL     string
	licenseKey    string
	environment   string
	stateFilePath string
	now           func() time.Time
}

func New(serverURL, licenseKey, environment, stateFilePath string, now func() time.Time) *Client {
	if now == nil {
		now = time.Now
	}
	return &Client{
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		serverURL:     serverURL,
		licenseKey:    licenseKey,
		environment:   environment,
		stateFilePath: stateFilePath,
		now:           now,
	}
}

type stateResponse struct {
	Success        bool   `json:"success"`
	InstallationID string `json:"installation_id"`
	LicenseState   string `json:"license_state"`
	Error          string `json:"error"`
	Message        string `json:"message"`
}

// Refresh adalah satu putaran kerja, dipanggil sekali saat startup dan tiap
// tick ticker 24 jam di cmd/server/main.go:
//
//   - LICENSE_KEY kosong -> tidak melakukan apa pun (instalasi memang belum
//     diberi lisensi).
//   - Belum ada local state (StatusMissing) -> coba /activate.
//   - Sudah ada local state (installation_id diketahui) -> coba /validate.
//
// Kegagalan jaringan/5xx TIDAK menghapus state lama (grace period ditangani
// requireLicense lewat validated_at, lihat internal/licensecheck). Kegagalan
// tegas (401/404 — key dicabut atau installation direset vendor) MENGHAPUS
// state lama, supaya tidak dipercaya lagi sampai grace period habis sendiri.
func (c *Client) Refresh(ctx context.Context) error {
	if c.licenseKey == "" {
		return nil
	}

	if installationID, ok := licensecheck.PeekInstallationID(c.stateFilePath); ok {
		return c.validate(ctx, installationID)
	}
	return c.activate(ctx)
}

func (c *Client) activate(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{
		"license_key":     c.licenseKey,
		"environment":     c.environment,
		"product_version": version.Current,
	})
	if err != nil {
		return fmt.Errorf("licenseclient: marshal activate request: %w", err)
	}

	resp, err := c.post(ctx, "/api/v1/license/activate", body, "")
	if err != nil {
		return err // jaringan bermasalah -- tidak ada state lama untuk dihapus
	}
	defer resp.Body.Close()

	var out stateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("licenseclient: decode activate response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// Key salah, license tidak aktif, atau kuota installation penuh --
		// dicatat sebagai kegagalan biasa (caller yang log), bukan
		// menghapus apa pun karena memang belum ada state untuk dihapus.
		return fmt.Errorf("licenseclient: activate ditolak (%d): %s", resp.StatusCode, out.Message)
	}

	return writeStateFile(c.stateFilePath, out.LicenseState)
}

func (c *Client) validate(ctx context.Context, installationID string) error {
	body, err := json.Marshal(map[string]string{
		"installation_id": installationID,
		"product_version": version.Current,
	})
	if err != nil {
		return fmt.Errorf("licenseclient: marshal validate request: %w", err)
	}

	resp, err := c.post(ctx, "/api/v1/license/validate", body, c.licenseKey)
	if err != nil {
		return err // jaringan bermasalah -- biarkan state lama, grace period yang menentukan
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		// Key dicabut, atau installation ini sudah di-reset vendor (migrasi
		// VPS dari sisi lain) -- state lama tidak lagi sah, hapus sekarang,
		// jangan tunggu grace period.
		if err := os.Remove(c.stateFilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("licenseclient: hapus state lama: %w", err)
		}
		return fmt.Errorf("licenseclient: validate ditolak (%d)", resp.StatusCode)
	}

	var out stateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("licenseclient: decode validate response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("licenseclient: validate gagal (%d): %s", resp.StatusCode, out.Message)
	}

	return writeStateFile(c.stateFilePath, out.LicenseState)
}

func (c *Client) post(ctx context.Context, path string, body []byte, bearer string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("licenseclient: buat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("licenseclient: %s: %w", path, err)
	}
	return resp, nil
}

// writeStateFile menulis atomik (temp file + rename) supaya request lain
// yang membaca license-state.lic lewat internal/licensecheck.Load tidak
// pernah melihat file setengah tertulis.
func writeStateFile(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".license-state-*.tmp")
	if err != nil {
		return fmt.Errorf("licenseclient: buat temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck -- no-op setelah rename berhasil

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("licenseclient: tulis temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("licenseclient: tutup temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("licenseclient: chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("licenseclient: rename ke %s: %w", path, err)
	}
	return nil
}
