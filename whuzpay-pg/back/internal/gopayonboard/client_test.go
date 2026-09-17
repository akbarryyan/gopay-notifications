package gopayonboard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignUp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/signup" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi-palsu"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123"); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
}

func TestSignUp_EmailTaken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"email_taken","message":"email sudah dipakai"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestSignUp_UsernameTaken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"username_taken","message":"username sudah dipakai"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("err = %v, want ErrUsernameTaken", err)
	}
}

func TestCreateAPIKey_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/api-keys" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"key_1","name":"whuzpay-pg","created_at":"2026-09-17T00:00:00Z","key":"sk_abc123"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	key, err := c.CreateAPIKey(t.Context(), "whuzpay-pg")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key != "sk_abc123" {
		t.Errorf("key = %q, want sk_abc123", key)
	}
}

func TestCreateWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/webhooks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://pg.whuzpay.com/api/v1/provider-webhooks/gopay" {
			t.Errorf("url = %v", body["url"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"wh_1","name":"whuzpay-pg","url":"https://pg.whuzpay.com/api/v1/provider-webhooks/gopay","events":["invoice.paid","invoice.expired"],"enabled":true,"created_at":"2026-09-17T00:00:00Z","secret":"whsec_abc123"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	secret, err := c.CreateWebhook(t.Context(), "whuzpay-pg", "https://pg.whuzpay.com/api/v1/provider-webhooks/gopay", []string{"invoice.paid", "invoice.expired"})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if secret != "whsec_abc123" {
		t.Errorf("secret = %q, want whsec_abc123", secret)
	}
}

func TestCreateDevice_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/devices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"device_id":"dev_abc123","device_secret":"c2VjcmV0"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "https://whuzpay.com")
	deviceID, deviceSecret, err := c.CreateDevice(t.Context(), "Toko Budi - Bridge")
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if deviceID != "dev_abc123" || deviceSecret != "c2VjcmV0" {
		t.Errorf("got %q/%q", deviceID, deviceSecret)
	}
	if got := c.PublicBackendURL(); got != "https://whuzpay.com/api/v1" {
		t.Errorf("PublicBackendURL() = %q", got)
	}
}

func TestUploadQRISImage_Success(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a} // PNG magic bytes cukup untuk http.DetectContentType
	b64 := base64.StdEncoding.EncodeToString(png)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/account/qris-image" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if body["content_type"] != "image/png" {
			t.Errorf("content_type = %q, want image/png", body["content_type"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	if err := c.UploadQRISImage(t.Context(), b64); err != nil {
		t.Fatalf("UploadQRISImage: %v", err)
	}
}

func TestSignUp_SesiDipakaiUlangUntukPanggilanBerikutnya(t *testing.T) {
	var sawCookieOnSecondCall bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi-palsu"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case "/api/v1/admin/api-keys":
			if c, err := r.Cookie("admin_session"); err == nil && c.Value == "sesi-palsu" {
				sawCookieOnSecondCall = true
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		}
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	if err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123"); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	if _, err := c.CreateAPIKey(t.Context(), "whuzpay-pg"); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if !sawCookieOnSecondCall {
		t.Error("cookie sesi dari SignUp tidak terbawa ke CreateAPIKey -- cookie jar tidak jalan")
	}
}
