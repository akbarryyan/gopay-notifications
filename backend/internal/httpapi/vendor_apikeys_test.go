package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// seedRawAPIKeyForAccount menyisipkan API key langsung lewat store (hash
// dummy, key mentahnya tidak pernah dibutuhkan test list/revoke) --
// bukan lewat POST /api/v1/admin/api-keys supaya test ini tidak butuh sesi
// customer sama sekali, cuma sesi vendor.
func seedRawAPIKeyForAccount(t *testing.T, accountID, id, name string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.CreateAPIKey(context.Background(), accountID, id, name, []byte("hash-dummy")); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
}

func TestVendorAPIKeysButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/vendor/accounts/acc_x/api-keys"},
		{http.MethodDelete, "/api/v1/vendor/accounts/acc_x/api-keys/key_x"},
	} {
		rec := vendorRequest(t, h, nil, tc.method, tc.path, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, mau 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestVendorListAPIKeys(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawAPIKeyForAccount(t, accountID, "key_v1", "Website Toko")

	rec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/api-keys", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		APIKeys []struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			RevokedAt *string `json:"revoked_at"`
		} `json:"api_keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.APIKeys) != 1 || body.APIKeys[0].ID != "key_v1" || body.APIKeys[0].Name != "Website Toko" {
		t.Fatalf("api_keys = %+v, mau 1 baris key_v1", body.APIKeys)
	}
	if body.APIKeys[0].RevokedAt != nil {
		t.Fatalf("revoked_at = %v, mau nil (belum dicabut)", body.APIKeys[0].RevokedAt)
	}
	// key_hash tidak boleh pernah tercermin di response dalam bentuk apa pun.
	if strings.Contains(rec.Body.String(), "hash-dummy") {
		t.Fatal("response tidak boleh memuat key_hash sama sekali")
	}
}

func TestVendorRevokeAPIKey(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawAPIKeyForAccount(t, accountID, "key_cabut", "Website Bocor")

	rec := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/api-keys/key_cabut", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/api-keys", "")
	var body struct {
		APIKeys []struct {
			ID        string  `json:"id"`
			RevokedAt *string `json:"revoked_at"`
		} `json:"api_keys"`
	}
	json.Unmarshal(list.Body.Bytes(), &body)
	if len(body.APIKeys) != 1 || body.APIKeys[0].RevokedAt == nil {
		t.Fatalf("api_keys = %+v, mau key_cabut dengan revoked_at terisi", body.APIKeys)
	}
}

func TestVendorRevokeAPIKeyIdempotent(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawAPIKeyForAccount(t, accountID, "key_idem", "Website")

	first := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/api-keys/key_idem", "")
	second := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/api-keys/key_idem", "")
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("status = %d, %d, mau 200 keduanya (idempotent)", first.Code, second.Code)
	}
}

func TestVendorRevokeAPIKeyTakDikenal(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")

	rec := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/api-keys/key_tidak_ada", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}
