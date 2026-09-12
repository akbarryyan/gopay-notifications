package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type apiKeyBody struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	RevokedAt *string `json:"revoked_at"`
	Key       string  `json:"key"`
}

func createAPIKeyReq(t *testing.T, h http.Handler, cookie *http.Cookie, name string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"name":"` + name + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminCreateAPIKeyMengembalikanKeyMentahSekali(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := createAPIKeyReq(t, h, cookie, "Website utama")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var body apiKeyBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Key == "" {
		t.Fatal("key mentah kosong di response create — harus ada, hanya sekali ini")
	}
	if body.Name != "Website utama" {
		t.Fatalf("Name = %s, mau 'Website utama'", body.Name)
	}
}

func TestAdminListAPIKeysTidakMenyertakanKeyMentah(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	createAPIKeyReq(t, h, cookie, "Website utama")

	rec := adminGet(t, h, cookie, "/api/v1/admin/api-keys")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	// Struct tanpa field Key sama sekali — kalau field "key" tetap ada di
	// JSON respons (harusnya tidak), unmarshal ke sini tetap tidak akan
	// menangkapnya; pengujian sesungguhnya ada di grep string mentah di bawah.
	if strings.Contains(rec.Body.String(), `"key":`) {
		t.Fatalf("respons list membocorkan key mentah: %s", rec.Body.String())
	}
}

func TestAdminRevokeAPIKeyMenolakKeyBerikutnya(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	created := createAPIKeyReq(t, h, cookie, "Website utama")
	var body apiKeyBody
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	revokeReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+body.ID, nil)
	revokeReq.AddCookie(cookie)
	revokeRec := httptest.NewRecorder()
	h.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("status revoke = %d (body=%s)", revokeRec.Code, revokeRec.Body.String())
	}

	// Idempotent: mencabut lagi tetap 200.
	revokeReq2 := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+body.ID, nil)
	revokeReq2.AddCookie(cookie)
	revokeRec2 := httptest.NewRecorder()
	h.ServeHTTP(revokeRec2, revokeReq2)
	if revokeRec2.Code != http.StatusOK {
		t.Fatalf("status revoke kedua = %d, mau tetap 200 (idempotent)", revokeRec2.Code)
	}
}

func TestAdminRevokeAPIKeyTidakDitemukan(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/key_tidak_ada", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}

func TestAdminAPIKeysMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", strings.NewReader(`{"name":"x"}`)),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, mau 401", req.Method, req.URL.Path, rec.Code)
		}
	}
}
