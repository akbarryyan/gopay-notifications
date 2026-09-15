package httpapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// tinyPNGBase64 adalah PNG 1x1 valid (byte magic number PNG + IHDR minimal
// cukup untuk http.DetectContentType mengenalinya sebagai image/png --
// tidak perlu gambar utuh untuk keperluan test).
var tinyPNGBase64 = base64.StdEncoding.EncodeToString([]byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
})

func adminPutBody(t *testing.T, h http.Handler, cookie *http.Cookie, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadQRISImageBerhasil(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	get := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d (body=%s)", get.Code, get.Body.String())
	}
	if ct := get.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, mau image/png", ct)
	}
	wantBytes, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if get.Body.String() != string(wantBytes) {
		t.Fatal("body GET tidak sama dengan yang di-upload")
	}
}

func TestUploadQRISImageBase64TidakValid(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"bukan-base64-!!!","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "invalid_payload" {
		t.Fatalf("error = %q, mau invalid_payload", errBody.Error)
	}
}

func TestUploadQRISImageTerlaluBesar(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	big := make([]byte, 300*1024+1)
	bigB64 := base64.StdEncoding.EncodeToString(big)
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"`+bigB64+`","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "image_too_large" {
		t.Fatalf("error = %q, mau image_too_large", errBody.Error)
	}
}

func TestUploadQRISImageTipeTidakDidukung(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	// Klaim image/png tapi isinya teks biasa -- membuktikan validasi pakai
	// http.DetectContentType atas ISI byte, bukan field content_type yang
	// diklaim klien.
	textB64 := base64.StdEncoding.EncodeToString([]byte("ini bukan gambar sama sekali, cuma teks biasa"))
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"`+textB64+`","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "unsupported_image_type" {
		t.Fatalf("error = %q, mau unsupported_image_type", errBody.Error)
	}
}

func TestGetQRISImageBelumAdaMengembalikan404(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestDeleteQRISImageIdempotenLewatHTTP(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	first := adminDelete(t, h, cookie, "/api/v1/admin/account/qris-image")
	if first.Code != http.StatusOK {
		t.Fatalf("delete pertama status = %d", first.Code)
	}

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)

	second := adminDelete(t, h, cookie, "/api/v1/admin/account/qris-image")
	if second.Code != http.StatusOK {
		t.Fatalf("delete kedua status = %d", second.Code)
	}
	get := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET setelah delete status = %d, mau 404", get.Code)
	}
}

func TestQRISImageEndpointButuhSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPut, `{"image_base64":"x","content_type":"image/png"}`},
		{http.MethodDelete, ""},
	} {
		req := httptest.NewRequest(tc.method, "/api/v1/admin/account/qris-image", strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, mau 401 tanpa cookie", tc.method, rec.Code)
		}
	}
}

func TestGetAccountProfileMemuatQRISImageConfigured(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	before := adminGet(t, h, cookie, "/api/v1/admin/account")
	var beforeBody struct {
		Account struct {
			QRISImageConfigured bool `json:"qris_image_configured"`
		} `json:"account"`
	}
	json.Unmarshal(before.Body.Bytes(), &beforeBody)
	if beforeBody.Account.QRISImageConfigured {
		t.Fatal("qris_image_configured mau false sebelum upload")
	}

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)

	after := adminGet(t, h, cookie, "/api/v1/admin/account")
	var afterBody struct {
		Account struct {
			QRISImageConfigured bool `json:"qris_image_configured"`
		} `json:"account"`
	}
	json.Unmarshal(after.Body.Bytes(), &afterBody)
	if !afterBody.Account.QRISImageConfigured {
		t.Fatal("qris_image_configured mau true setelah upload")
	}
}
