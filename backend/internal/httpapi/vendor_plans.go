package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type planJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	MaxDevices  int      `json:"max_devices"`
	PriceLabel  string   `json:"price_label"`
	PricePeriod string   `json:"price_period"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Highlighted bool     `json:"highlighted"`
	Visible     bool     `json:"visible"`
	SortOrder   int      `json:"sort_order"`
	UpdatedAt   string   `json:"updated_at"`
}

func toPlanJSON(p store.Plan) planJSON {
	return planJSON{
		ID: p.ID, Name: p.Name, MaxDevices: p.MaxDevices, PriceLabel: p.PriceLabel,
		PricePeriod: p.PricePeriod, Description: p.Description, Features: p.Features,
		Highlighted: p.Highlighted, Visible: p.Visible, SortOrder: p.SortOrder,
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
}

// handleVendorListPlans melayani halaman kelola paket DAN dropdown plan di
// form buat/ubah account -- sengaja mengembalikan SEMUA plan (termasuk yang
// visible=false), beda dari handlePublicPricingPlans yang cuma untuk
// landing page.
func (a *API) handleVendorListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := a.store.ListPlans(r.Context())
	if err != nil {
		slog.Error("ambil plans gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]planJSON, 0, len(plans))
	for _, p := range plans {
		out = append(out, toPlanJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "plans": out})
}

type planRequest struct {
	Name        string   `json:"name"`
	MaxDevices  int      `json:"max_devices"`
	Unlimited   bool     `json:"unlimited"`
	PriceLabel  string   `json:"price_label"`
	PricePeriod string   `json:"price_period"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Highlighted bool     `json:"highlighted"`
	Visible     bool     `json:"visible"`
}

// toPlanInput memvalidasi dan menormalkan body request menjadi
// store.PlanInput -- dipakai bersama oleh create dan update.
func (a *API) toPlanInput(w http.ResponseWriter, req planRequest) (store.PlanInput, bool) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return store.PlanInput{}, false
	}
	maxDevices := req.MaxDevices
	if req.Unlimited {
		maxDevices = -1
	} else if maxDevices < 1 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"max_devices harus lebih dari 0 (atau set unlimited)")
		return store.PlanInput{}, false
	}

	features := make([]string, 0, len(req.Features))
	for _, f := range req.Features {
		f = strings.TrimSpace(f)
		if f != "" {
			features = append(features, f)
		}
	}
	priceLabel := strings.TrimSpace(req.PriceLabel)
	pricePeriod := strings.TrimSpace(req.PricePeriod)
	if priceLabel == "" && pricePeriod != "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"price_period cuma boleh diisi bila price_label juga diisi")
		return store.PlanInput{}, false
	}

	return store.PlanInput{
		Name: name, MaxDevices: maxDevices, PriceLabel: priceLabel, PricePeriod: pricePeriod,
		Description: strings.TrimSpace(req.Description), Features: features,
		Highlighted: req.Highlighted, Visible: req.Visible,
	}, true
}

// handleVendorCreatePlan menambah plan baru. Tidak ada batas jumlah plan --
// vendor bebas menambah selama nama tidak bentrok.
func (a *API) handleVendorCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req planRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	input, ok := a.toPlanInput(w, req)
	if !ok {
		return
	}
	id, err := store.NewPlanID()
	if err != nil {
		slog.Error("generate plan id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	plan, err := a.store.CreatePlan(r.Context(), id, input)
	if errors.Is(err, store.ErrPlanNameTaken) {
		a.writeError(w, http.StatusConflict, "name_taken", "nama plan sudah dipakai")
		return
	}
	if err != nil {
		slog.Error("create plan gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "PLAN_CREATED", plan.ID,
		map[string]string{"name": plan.Name}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "plan": toPlanJSON(plan)})
}

func (a *API) handleVendorUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("planID")
	var req planRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	input, ok := a.toPlanInput(w, req)
	if !ok {
		return
	}
	plan, err := a.store.UpdatePlan(r.Context(), id, input)
	if errors.Is(err, store.ErrPlanNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "plan tidak ditemukan")
		return
	}
	if errors.Is(err, store.ErrPlanNameTaken) {
		a.writeError(w, http.StatusConflict, "name_taken", "nama plan sudah dipakai")
		return
	}
	if err != nil {
		slog.Error("update plan gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "PLAN_UPDATED", plan.ID,
		map[string]string{"name": plan.Name}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "plan": toPlanJSON(plan)})
}

// handleVendorDeletePlan menghapus definisi plan. Aman dilakukan kapan pun
// -- lihat komentar store.DeletePlan soal kenapa account yang sudah
// memakai nama plan ini tidak terpengaruh.
func (a *API) handleVendorDeletePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("planID")
	if err := a.store.DeletePlan(r.Context(), id); errors.Is(err, store.ErrPlanNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "plan tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("delete plan gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "PLAN_DELETED", id, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

type movePlanRequest struct {
	// "up" | "down" -- bukan angka sort_order mentah, supaya frontend tidak
	// perlu tahu skema pengurutan internalnya sama sekali.
	Direction string `json:"direction"`
}

func (a *API) handleVendorMovePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("planID")
	var req movePlanRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	var direction int
	switch req.Direction {
	case "up":
		direction = -1
	case "down":
		direction = 1
	default:
		a.writeError(w, http.StatusBadRequest, "invalid_payload", `direction harus "up" atau "down"`)
		return
	}
	if err := a.store.MovePlan(r.Context(), id, direction); errors.Is(err, store.ErrPlanNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "plan tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("pindah urutan plan gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
