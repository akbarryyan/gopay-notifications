package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// planPresets memetakan nama plan ke max_devices default (§9 spec MVP --
// entitlement lain di docs/license-spec.md §17 ditunda, belum ada
// fiturnya). -1 berarti unlimited.
var planPresets = map[string]int{
	"Starter":    3,
	"Business":   10,
	"Enterprise": -1,
}

type accountVendorJSON struct {
	ID            string `json:"id"`
	BusinessName  string `json:"business_name"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	Plan          string `json:"plan"`
	MaxDevices    int    `json:"max_devices"`
	Status        string `json:"status"`
	ExpiresAt     string `json:"expires_at"`
	DaysRemaining int    `json:"days_remaining"`
	CreatedAt     string `json:"created_at"`
}

func toAccountVendorJSON(acc store.Account, now time.Time) accountVendorJSON {
	return accountVendorJSON{
		ID: acc.ID, BusinessName: acc.BusinessName, Email: acc.Email, Username: acc.Username,
		Plan: acc.Plan, MaxDevices: acc.MaxDevices, Status: acc.DerivedStatus(now),
		ExpiresAt:     acc.ExpiresAt.Format(dateOnlyLayout),
		DaysRemaining: int(acc.ExpiresAt.Sub(now).Hours() / 24),
		CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
	}
}

type createAccountRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Plan         string `json:"plan"`
	ExpiresAt    string `json:"expires_at"`
}

func randomPrefixedAccountID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random account id: %w", err)
	}
	return "acc_" + hex.EncodeToString(b), nil
}

// generateInitialPassword membuat password awal akun -- ditampilkan sekali
// ke vendor saat pembuatan (pola sama seperti API key/webhook secret),
// dikirim manual ke customer lewat kanal vendor sendiri (bukan email
// otomatis di MVP ini).
func generateInitialPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789abcdefghjkmnpqrstuvwxyz"
	pw := make([]byte, len(b))
	for i, v := range b {
		pw[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(pw), nil
}

// handleVendorCreateAccount membuat account baru (business_name/email/
// username/plan/expires_at diisi vendor lewat Vendor Dashboard) --
// menggantikan customer+license terpisah di License Server lama, sekarang
// satu entitas.
func (a *API) handleVendorCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.BusinessName == "" || req.Email == "" || req.Username == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "business_name, email, dan username wajib diisi")
		return
	}
	maxDevices, ok := planPresets[req.Plan]
	if !ok {
		a.writeError(w, http.StatusBadRequest, "invalid_plan", "plan harus salah satu: Starter, Business, Enterprise")
		return
	}
	expiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}
	id, err := randomPrefixedAccountID()
	if err != nil {
		slog.Error("generate account id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	password, err := generateInitialPassword()
	if err != nil {
		slog.Error("generate password gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: password, Plan: req.Plan, MaxDevices: maxDevices, ExpiresAt: expiresAt,
	})
	if errors.Is(err, store.ErrAccountEmailTaken) {
		a.writeError(w, http.StatusConflict, "email_taken", "email sudah dipakai akun lain")
		return
	}
	if errors.Is(err, store.ErrAccountUsernameTaken) {
		a.writeError(w, http.StatusConflict, "username_taken", "username sudah dipakai akun lain")
		return
	}
	if err != nil {
		slog.Error("create account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "ACCOUNT_CREATED", id,
		map[string]string{"business_name": req.BusinessName, "plan": req.Plan}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	acc, err := a.store.GetAccountByID(r.Context(), id)
	if err != nil {
		slog.Error("ambil account baru gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true, "account": toAccountVendorJSON(acc, a.now()), "initial_password": password,
	})
}

func (a *API) handleVendorListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.store.ListAccounts(r.Context())
	if err != nil {
		slog.Error("list accounts gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]accountVendorJSON, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, toAccountVendorJSON(acc, a.now()))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "accounts": out})
}

func (a *API) handleVendorGetAccount(w http.ResponseWriter, r *http.Request) {
	acc, err := a.store.GetAccountByID(r.Context(), r.PathValue("accountID"))
	if errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("get account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "account": toAccountVendorJSON(acc, a.now())})
}

type renewAccountRequest struct {
	ExpiresAt string `json:"expires_at"`
}

func (a *API) handleVendorRenewAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("accountID")
	var req renewAccountRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	newExpiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}
	if err := a.store.RenewAccount(r.Context(), id, newExpiresAt); errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("renew account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "ACCOUNT_RENEWED", id,
		map[string]string{"new_expires_at": req.ExpiresAt}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) setAccountAdminStatus(w http.ResponseWriter, r *http.Request, status, action string) {
	id := r.PathValue("accountID")
	if err := a.store.SetAccountAdminStatus(r.Context(), id, status); errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	} else if err != nil {
		slog.Error(action+" gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, action, id, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleVendorSuspendAccount(w http.ResponseWriter, r *http.Request) {
	a.setAccountAdminStatus(w, r, "suspended", "ACCOUNT_SUSPENDED")
}

func (a *API) handleVendorRevokeAccount(w http.ResponseWriter, r *http.Request) {
	a.setAccountAdminStatus(w, r, "revoked", "ACCOUNT_REVOKED")
}
