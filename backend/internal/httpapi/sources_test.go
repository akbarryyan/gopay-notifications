package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSourcesMengembalikanConnectorYangDikenal(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestAPI(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var body struct {
		Sources []struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Packages []string `json:"packages"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Sources) == 0 {
		t.Fatal("sources kosong")
	}
	if body.Sources[0].ID != "gopay" || body.Sources[0].Name != "GoPay" {
		t.Fatalf("sources[0] = %+v", body.Sources[0])
	}
	if len(body.Sources[0].Packages) == 0 {
		t.Fatal("packages kosong — dashboard butuh ini untuk panduan setup")
	}
}

func TestSourcesTidakMemerlukanAutentikasi(t *testing.T) {
	// Daftar connector bukan rahasia, dan dashboard membutuhkannya
	// sebelum device mana pun terdaftar.
	rec := httptest.NewRecorder()
	newTestAPI(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
}
