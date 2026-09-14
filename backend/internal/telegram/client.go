// Package telegram adalah klien kecil Bot API Telegram: getMe, getUpdates,
// dan sendMessage -- cuma yang dipakai proyek ini.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// DefaultBaseURL adalah alamat Bot API. Bisa diganti di test.
const DefaultBaseURL = "https://api.telegram.org"

// ErrConflict: bot yang sama sedang dibaca proses lain (getUpdates lain,
// mis. backend di laptop memakai token produksi) atau punya webhook aktif.
// Telegram cuma mengizinkan SATU pembaca per bot.
var ErrConflict = errors.New("telegram: bot sedang dipakai proses lain atau punya webhook aktif")

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(token string) *Client { return NewWithBaseURL(DefaultBaseURL, token) }

func NewWithBaseURL(baseURL, token string) *Client {
	// Timeout harus lebih panjang dari long polling getUpdates (50 detik).
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: 70 * time.Second}}
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"` // "private" | "group" | "supergroup" | "channel"
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

func (c *Client) call(ctx context.Context, method string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method), bytes.NewReader(body))
	if err != nil {
		// Galat pembuatan request bisa memuat URL lengkap berisi token.
		return fmt.Errorf("telegram: %s: request tidak valid", method)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Galat dari http.Client memuat URL lengkap -- termasuk token bot.
		// Jangan pernah diteruskan apa adanya ke log atau respons HTTP.
		return fmt.Errorf("telegram: %s: gagal menghubungi api telegram", method)
	}
	defer resp.Body.Close()

	var r apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("telegram: %s: jawaban tidak dapat dibaca (status %d)", method, resp.StatusCode)
	}
	if !r.OK {
		if r.ErrorCode == http.StatusConflict {
			return ErrConflict
		}
		return fmt.Errorf("telegram: %s: %d %s", method, r.ErrorCode, r.Description)
	}
	if out != nil {
		if err := json.Unmarshal(r.Result, out); err != nil {
			return fmt.Errorf("telegram: %s: hasil tidak dapat dibaca: %w", method, err)
		}
	}
	return nil
}

// GetMe dipakai mengambil username bot untuk menyusun deep link.
func (c *Client) GetMe(ctx context.Context) (User, error) {
	var u User
	err := c.call(ctx, "getMe", map[string]any{}, &u)
	return u, err
}

// GetUpdates melakukan long polling: menunggu sampai `timeout` bila belum ada
// pesan baru. Cuma pesan (bukan edit, callback, dsb.) yang diminta.
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	var updates []Update
	err := c.call(ctx, "getUpdates", map[string]any{
		"offset":          offset,
		"timeout":         int(timeout.Seconds()),
		"allowed_updates": []string{"message"},
	}, &updates)
	return updates, err
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return c.call(ctx, "sendMessage", map[string]any{"chat_id": chatID, "text": text}, nil)
}
