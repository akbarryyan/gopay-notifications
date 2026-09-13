package licenseclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func statePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "license-state.lic")
}

func TestRefreshTanpaLicenseKeyTidakMelakukanApaPun(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	c := New(srv.URL, "", "production", statePath(t), nil)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if called {
		t.Fatal("tidak boleh memanggil server sama sekali tanpa LICENSE_KEY")
	}
}

func TestRefreshBelumAktivasiMemanggilActivateDanMenulisFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/license/activate" {
			t.Errorf("path = %s, mau /api/v1/license/activate", r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["license_key"] != "PB-TEST-KEY" || body["environment"] != "production" {
			t.Errorf("body tidak sesuai: %+v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success": true, "installation_id": "inst_1", "license_state": "payload\nsignature\n",
		})
	}))
	defer srv.Close()

	path := statePath(t)
	c := New(srv.URL, "PB-TEST-KEY", "production", path, nil)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("baca file state: %v", err)
	}
	if string(content) != "payload\nsignature\n" {
		t.Fatalf("isi file = %q, mau %q", content, "payload\nsignature\n")
	}
}

func TestRefreshActivateGagalJaringanTidakMenulisApaPun(t *testing.T) {
	c := New("http://127.0.0.1:1", "PB-TEST-KEY", "production", statePath(t), nil)
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("mau error karena server tidak terjangkau, dapat nil")
	}
}

func TestRefreshSudahAktivasiMemanggilValidate(t *testing.T) {
	path := statePath(t)
	// Refresh menentukan activate vs validate lewat
	// licensecheck.PeekInstallationID, yang TIDAK memverifikasi signature
	// (lihat komentarnya) -- jadi payload base64 valid berisi
	// installation_id sudah cukup untuk test ini, signature-nya sendiri
	// (baris kedua) boleh sembarang.
	var validateCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/license/activate":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true, "installation_id": "inst_1",
				"license_state": "eyJpbnN0YWxsYXRpb25faWQiOiAiaW5zdF8xIn0=\nc2ln\n",
			})
		case "/api/v1/license/validate":
			validateCalled = true
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["installation_id"] != "inst_1" {
				t.Errorf("installation_id = %q, mau inst_1", body["installation_id"])
			}
			if r.Header.Get("Authorization") != "Bearer PB-TEST-KEY" {
				t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
			}
			json.NewEncoder(w).Encode(map[string]any{
				"success": true, "license_state": "payload2\nsignature2\n",
			})
		default:
			t.Errorf("path tak terduga: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "PB-TEST-KEY", "production", path, nil)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh pertama (activate): %v", err)
	}
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh kedua (validate): %v", err)
	}
	if !validateCalled {
		t.Fatal("validate tidak pernah dipanggil pada Refresh kedua")
	}

	content, _ := os.ReadFile(path)
	if string(content) != "payload2\nsignature2\n" {
		t.Fatalf("isi file setelah validate = %q", content)
	}
}

func TestRefreshValidateDitolakMenghapusFile(t *testing.T) {
	path := statePath(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/license/activate":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true, "installation_id": "inst_1",
				"license_state": "eyJpbnN0YWxsYXRpb25faWQiOiAiaW5zdF8xIn0=\nc2ln\n",
			})
		case "/api/v1/license/validate":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "installation_not_found"})
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "PB-TEST-KEY", "production", path, nil)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh pertama (activate): %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file state harus ada sebelum validate ditolak: %v", err)
	}

	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("mau error saat validate ditolak, dapat nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file state harus terhapus setelah validate ditolak 404")
	}
}

func TestRefreshValidateGagalJaringanTidakMenghapusFile(t *testing.T) {
	path := statePath(t)
	activateSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": true, "installation_id": "inst_1",
			"license_state": "eyJpbnN0YWxsYXRpb25faWQiOiAiaW5zdF8xIn0=\nc2ln\n",
		})
	}))
	defer activateSrv.Close()

	c := New(activateSrv.URL, "PB-TEST-KEY", "production", path, nil)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh pertama (activate): %v", err)
	}

	// Ganti ke server yang tidak ada -- mensimulasikan License Server down
	// saat validate berikutnya.
	c2 := New("http://127.0.0.1:1", "PB-TEST-KEY", "production", path, nil)
	if err := c2.Refresh(context.Background()); err == nil {
		t.Fatal("mau error jaringan, dapat nil")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("file state TIDAK boleh terhapus saat kegagalan jaringan (grace period)")
	}
}

func TestNewNowDefaultKeTimeNow(t *testing.T) {
	c := New("http://example.com", "", "production", statePath(t), nil)
	if c.now == nil {
		t.Fatal("now tidak boleh nil setelah New")
	}
	if diff := c.now().Sub(time.Now()); diff > time.Second || diff < -time.Second {
		t.Fatalf("now() tidak dekat dengan waktu sungguhan: diff=%v", diff)
	}
}
