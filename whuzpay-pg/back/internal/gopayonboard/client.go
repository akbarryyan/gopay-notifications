package gopayonboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Client memanggil endpoint session-based gopay-notifications untuk
// menyediakan (provisioning) akun, API key, webhook, device, dan QRIS
// sekali di awal saat merchant baru daftar di whuzpay-pg (lihat spec
// 2026-09-17-whuzpay-pg-unified-onboarding-design.md). SELALU dibuat baru
// per registrasi lewat NewClient -- cookie jar-nya TIDAK PERNAH dibagi
// antar request/merchant, dan tidak pernah ditulis ke database. Kalau
// dibuat sekali lalu dipakai ulang untuk banyak merchant, sesi satu
// merchant bisa bocor ke merchant lain (satu jar = satu domain = satu set
// cookie, dan seluruh panggilan mengarah ke domain gopay-notifications
// yang sama).
//
// Terpisah dari internal/provider/gopay.Adapter: itu memanggil
// POST /invoices per transaksi pakai API key merchant; ini memanggil
// /admin/* pakai sesi, sekali saja, di titik yang berbeda dalam siklus
// hidup akun.
type Client struct {
	baseURL    string // loopback di produksi (GOPAY_BASE_URL)
	publicURL  string // publik (GOPAY_PUBLIC_BASE_URL) -- dipakai QR pairing, HP tidak bisa mengakses loopback
	httpClient *http.Client
}

func NewClient(baseURL, publicURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("gopayonboard: buat cookie jar: %w", err)
	}
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		publicURL: strings.TrimRight(publicURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
	}, nil
}

// PublicBackendURL adalah nilai backend_url yang WAJIB ditaruh di payload
// QR pairing -- harus URL publik (https://whuzpay.com/api/v1), bukan
// baseURL (bisa loopback http://127.0.0.1:8080 di produksi) karena HP
// tidak bisa mengakses loopback VPS.
func (c *Client) PublicBackendURL() string {
	return c.publicURL + "/api/v1"
}

type apiErrorBody struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (c *Client) SignUp(ctx context.Context, businessName, email, username, password string) error {
	body, err := json.Marshal(map[string]string{
		"business_name": businessName,
		"email":         email,
		"username":      username,
		"password":      password,
	})
	if err != nil {
		return fmt.Errorf("gopayonboard: encode signup request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/signup", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gopayonboard: build signup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gopayonboard: signup: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var apiErr apiErrorBody
	_ = json.Unmarshal(respBody, &apiErr)
	switch apiErr.Error {
	case "email_taken":
		return ErrEmailTaken
	case "username_taken":
		return ErrUsernameTaken
	}
	return fmt.Errorf("gopayonboard: signup status %d: %s", resp.StatusCode, string(respBody))
}

func (c *Client) CreateAPIKey(ctx context.Context, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/api-keys", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gopayonboard: build create api key request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gopayonboard: create api key: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("gopayonboard: create api key status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gopayonboard: decode create api key response: %w", err)
	}
	return out.Key, nil
}

func (c *Client) CreateWebhook(ctx context.Context, name, url string, events []string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"name":   name,
		"url":    url,
		"events": events,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/webhooks", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gopayonboard: build create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gopayonboard: create webhook: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("gopayonboard: create webhook status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gopayonboard: decode create webhook response: %w", err)
	}
	return out.Secret, nil
}

func (c *Client) CreateDevice(ctx context.Context, name string) (deviceID, deviceSecret string, err error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/devices", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("gopayonboard: build create device request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("gopayonboard: create device: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("gopayonboard: create device status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		DeviceID     string `json:"device_id"`
		DeviceSecret string `json:"device_secret"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", "", fmt.Errorf("gopayonboard: decode create device response: %w", err)
	}
	return out.DeviceID, out.DeviceSecret, nil
}

func (c *Client) UploadQRISImage(ctx context.Context, imageBase64 string) error {
	decoded, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return fmt.Errorf("gopayonboard: decode qris image: %w", err)
	}
	contentType := http.DetectContentType(decoded)

	body, _ := json.Marshal(map[string]string{
		"image_base64": imageBase64,
		"content_type": contentType,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/v1/admin/account/qris-image", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gopayonboard: build upload qris request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gopayonboard: upload qris: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gopayonboard: upload qris status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
