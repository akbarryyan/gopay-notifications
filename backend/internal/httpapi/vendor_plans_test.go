package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
)

type planBody struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	MaxDevices  int      `json:"max_devices"`
	PriceLabel  string   `json:"price_label"`
	PricePeriod string   `json:"price_period"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Highlighted bool     `json:"highlighted"`
	Visible     bool     `json:"visible"`
}

func TestVendorPlansButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/vendor/plans"},
		{http.MethodPost, "/api/v1/vendor/plans"},
		{http.MethodPatch, "/api/v1/vendor/plans/plan_x"},
		{http.MethodDelete, "/api/v1/vendor/plans/plan_x"},
		{http.MethodPost, "/api/v1/vendor/plans/plan_x/move"},
	} {
		rec := vendorRequest(t, h, nil, tc.method, tc.path, "{}")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, mau 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestVendorListPlansMengembalikanSeedBawaan(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	rec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/plans", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Plans []planBody `json:"plans"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Plans) != 3 {
		t.Fatalf("plans = %+v, mau 3 (seed bawaan Starter/Business/Enterprise)", body.Plans)
	}
}

func TestVendorCreateUpdateDeletePlan(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	createRec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans", `{
		"name": "Ultra", "max_devices": 20, "price_label": "Rp 499.000", "price_period": "/bulan",
		"description": "Untuk volume sangat tinggi", "features": ["Fitur A", "  ", "Fitur B"],
		"highlighted": true, "visible": true
	}`)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		Plan planBody `json:"plan"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Plan.MaxDevices != 20 || len(created.Plan.Features) != 2 {
		t.Fatalf("plan dibuat = %+v, mau max_devices 20 dan 2 fitur (baris kosong dibuang)", created.Plan)
	}

	// Nama bentrok dengan plan seed bawaan.
	dup := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans", `{"name":"Starter","max_devices":1,"visible":true}`)
	if dup.Code != http.StatusConflict || errorCode(t, dup) != "name_taken" {
		t.Fatalf("dup status = %d body=%s, mau 409 name_taken", dup.Code, dup.Body.String())
	}

	// max_devices <= 0 tanpa unlimited ditolak.
	invalid := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans", `{"name":"Salah","max_devices":0,"visible":true}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, mau 400", invalid.Code)
	}

	// unlimited: true mengabaikan max_devices dan tersimpan sebagai -1.
	unlimitedRec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans", `{"name":"Tanpa Batas","unlimited":true,"visible":true}`)
	var unlimited struct {
		Plan planBody `json:"plan"`
	}
	json.Unmarshal(unlimitedRec.Body.Bytes(), &unlimited)
	if unlimited.Plan.MaxDevices != -1 {
		t.Fatalf("max_devices = %d, mau -1 (unlimited)", unlimited.Plan.MaxDevices)
	}

	// Update.
	updateRec := vendorRequest(t, h, cookie, http.MethodPatch, "/api/v1/vendor/plans/"+created.Plan.ID, `{
		"name": "Ultra Plus", "max_devices": 30, "price_label": "Rp 599.000", "price_period": "/bulan",
		"description": "diperbarui", "features": ["Fitur C"], "highlighted": false, "visible": false
	}`)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated struct {
		Plan planBody `json:"plan"`
	}
	json.Unmarshal(updateRec.Body.Bytes(), &updated)
	if updated.Plan.Name != "Ultra Plus" || updated.Plan.Visible {
		t.Fatalf("plan setelah update = %+v", updated.Plan)
	}

	// Yang disembunyikan tidak muncul di endpoint publik.
	pub := vendorRequest(t, h, nil, http.MethodGet, "/api/v1/pricing-plans", "")
	var pubBody struct {
		Plans []struct {
			Name string `json:"name"`
		} `json:"plans"`
	}
	json.Unmarshal(pub.Body.Bytes(), &pubBody)
	for _, p := range pubBody.Plans {
		if p.Name == "Ultra Plus" {
			t.Fatal("plan visible=false ikut muncul di endpoint publik")
		}
	}

	// Update ke id yang tidak ada.
	if rec := vendorRequest(t, h, cookie, http.MethodPatch, "/api/v1/vendor/plans/plan_tidak_ada", `{"name":"X","max_devices":1,"visible":true}`); rec.Code != http.StatusNotFound {
		t.Fatalf("update tidak ada: status = %d, mau 404", rec.Code)
	}

	// Delete.
	if rec := vendorRequest(t, h, cookie, http.MethodDelete, "/api/v1/vendor/plans/"+created.Plan.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := vendorRequest(t, h, cookie, http.MethodDelete, "/api/v1/vendor/plans/"+created.Plan.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete ulang: status = %d, mau 404", rec.Code)
	}
}

func TestVendorMovePlan(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	listRec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/plans", "")
	var list struct {
		Plans []planBody `json:"plans"`
	}
	json.Unmarshal(listRec.Body.Bytes(), &list)
	if len(list.Plans) != 3 {
		t.Fatalf("plans awal = %d, mau 3", len(list.Plans))
	}
	firstID := list.Plans[0].ID

	if rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans/"+firstID+"/move", `{"direction":"down"}`); rec.Code != http.StatusOK {
		t.Fatalf("move status = %d body=%s", rec.Code, rec.Body.String())
	}
	after := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/plans", "")
	var listAfter struct {
		Plans []planBody `json:"plans"`
	}
	json.Unmarshal(after.Body.Bytes(), &listAfter)
	if listAfter.Plans[0].ID == firstID {
		t.Fatalf("urutan tidak berubah setelah move down: %+v", listAfter.Plans)
	}

	if rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans/"+firstID+"/move", `{"direction":"sideways"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("direction salah: status = %d, mau 400", rec.Code)
	}
	if rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/plans/plan_tidak_ada/move", `{"direction":"up"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("plan tidak ada: status = %d, mau 404", rec.Code)
	}
}

func TestPublicPricingPlans(t *testing.T) {
	h := newAPIWithVendor(t)

	rec := vendorRequest(t, h, nil, http.MethodGet, "/api/v1/pricing-plans", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Plans []struct {
			Name        string   `json:"name"`
			DeviceLabel string   `json:"device_label"`
			Features    []string `json:"features"`
			Highlighted bool     `json:"highlighted"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Plans) != 3 {
		t.Fatalf("plans = %+v, mau 3", body.Plans)
	}
	labels := map[string]string{}
	for _, p := range body.Plans {
		labels[p.Name] = p.DeviceLabel
	}
	if labels["Starter"] != "3 device" || labels["Enterprise"] != "Device tanpa batas" {
		t.Fatalf("device labels = %+v", labels)
	}
}

func TestSignupGagalBilaPlanTrialTidakAda(t *testing.T) {
	s := newTestStore(t)
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(),
		func() time.Time { return fixedNow }).Handler()

	// Signup normal dulu -- plan "Starter" masih ada dari seedDefaultPlans.
	if rec := signupReq(t, h, "Toko Baru", "baru@uji.test", "toko_baru_2", "password123"); rec.Code != http.StatusOK {
		t.Fatalf("signup dengan plan Starter tersedia harus 200, status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Hapus plan "Starter" langsung lewat store -- signup TIDAK BOLEH
	// diam-diam memakai kuota lama atau sembarang angka, harus menjawab
	// jelas "belum tersedia" (503), bukan 500 atau sukses dengan kuota
	// yang tidak jelas dari mana asalnya.
	starter, err := s.GetPlanByName(context.Background(), "Starter")
	if err != nil {
		t.Fatalf("GetPlanByName Starter: %v", err)
	}
	if err := s.DeletePlan(context.Background(), starter.ID); err != nil {
		t.Fatalf("DeletePlan Starter: %v", err)
	}

	rec := signupReq(t, h, "Toko Lain", "lain2@uji.test", "toko_lain_2", "password123")
	if rec.Code != http.StatusServiceUnavailable || errorCode(t, rec) != "not_available" {
		t.Fatalf("status = %d body=%s, mau 503 not_available setelah plan Starter dihapus", rec.Code, rec.Body.String())
	}
}
