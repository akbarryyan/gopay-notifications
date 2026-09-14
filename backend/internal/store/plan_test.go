package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func createTestPlan(t *testing.T, s *store.Store, name string, maxDevices int) store.Plan {
	t.Helper()
	id, err := store.NewPlanID()
	if err != nil {
		t.Fatalf("NewPlanID: %v", err)
	}
	p, err := s.CreatePlan(context.Background(), id, store.PlanInput{
		Name: name, MaxDevices: maxDevices, PriceLabel: "Rp 99.000", PricePeriod: "/bulan",
		Description: "deskripsi uji", Features: []string{"Fitur A", "Fitur B"},
		Highlighted: false, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreatePlan %s: %v", name, err)
	}
	return p
}

func TestCreatePlanDanGetByName(t *testing.T) {
	s := testStore(t)
	created := createTestPlan(t, s, "Uji Coba", 5)

	if created.ID == "" || created.SortOrder == 0 {
		t.Fatalf("plan baru = %+v, mau id dan sort_order terisi", created)
	}
	if len(created.Features) != 2 || created.Features[0] != "Fitur A" {
		t.Fatalf("features = %v, urutan/isinya tidak sesuai", created.Features)
	}

	got, err := s.GetPlanByName(context.Background(), "Uji Coba")
	if err != nil {
		t.Fatalf("GetPlanByName: %v", err)
	}
	if got.MaxDevices != 5 || got.PriceLabel != "Rp 99.000" {
		t.Fatalf("got = %+v", got)
	}

	if _, err := s.GetPlanByName(context.Background(), "Tidak Ada"); !errors.Is(err, store.ErrPlanNotFound) {
		t.Fatalf("err = %v, mau ErrPlanNotFound", err)
	}
}

func TestCreatePlanNamaBentrokDitolak(t *testing.T) {
	s := testStore(t)
	createTestPlan(t, s, "Duplikat", 5)

	id, _ := store.NewPlanID()
	_, err := s.CreatePlan(context.Background(), id, store.PlanInput{Name: "Duplikat", MaxDevices: 1, Visible: true})
	if !errors.Is(err, store.ErrPlanNameTaken) {
		t.Fatalf("err = %v, mau ErrPlanNameTaken", err)
	}
}

func TestUpdatePlan(t *testing.T) {
	s := testStore(t)
	p := createTestPlan(t, s, "Sebelum", 5)

	updated, err := s.UpdatePlan(context.Background(), p.ID, store.PlanInput{
		Name: "Sesudah", MaxDevices: -1, PriceLabel: "Rp 199.000", PricePeriod: "/bulan",
		Description: "baru", Features: []string{"X"}, Highlighted: true, Visible: false,
	})
	if err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}
	if updated.Name != "Sesudah" || updated.MaxDevices != -1 || !updated.Highlighted || updated.Visible {
		t.Fatalf("updated = %+v", updated)
	}

	if _, err := s.UpdatePlan(context.Background(), "plan_tidak_ada", store.PlanInput{Name: "X", MaxDevices: 1, Visible: true}); !errors.Is(err, store.ErrPlanNotFound) {
		t.Fatalf("err = %v, mau ErrPlanNotFound", err)
	}
}

func TestUpdatePlanNamaBentrokDenganPlanLain(t *testing.T) {
	s := testStore(t)
	createTestPlan(t, s, "Plan A", 1)
	planB := createTestPlan(t, s, "Plan B", 1)

	if _, err := s.UpdatePlan(context.Background(), planB.ID, store.PlanInput{Name: "Plan A", MaxDevices: 1, Visible: true}); !errors.Is(err, store.ErrPlanNameTaken) {
		t.Fatalf("err = %v, mau ErrPlanNameTaken", err)
	}
}

func TestDeletePlanAmanWalauSudahDipakaiAccount(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	p := createTestPlan(t, s, "Akan Dihapus", 5)

	// account sudah memakai NAMA plan ini -- accounts.plan cuma teks bebas,
	// hapus plan tidak boleh menyentuh account yang sudah ada sama sekali.
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_pakai_plan", BusinessName: "Toko", Email: "toko@uji.test", Username: "toko",
		PlaintextPassword: "rahasia123", Plan: p.Name, MaxDevices: p.MaxDevices,
	}); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if err := s.DeletePlan(ctx, p.ID); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}
	if _, err := s.GetPlanByName(ctx, p.Name); !errors.Is(err, store.ErrPlanNotFound) {
		t.Fatalf("plan masih ada setelah dihapus: err = %v", err)
	}
	acc, err := s.GetAccountByID(ctx, "acc_pakai_plan")
	if err != nil || acc.Plan != p.Name || acc.MaxDevices != p.MaxDevices {
		t.Fatalf("account berubah setelah plan-nya dihapus: %+v, %v", acc, err)
	}

	if err := s.DeletePlan(ctx, "plan_tidak_ada"); !errors.Is(err, store.ErrPlanNotFound) {
		t.Fatalf("err = %v, mau ErrPlanNotFound", err)
	}
}

func TestListPlansDanListVisiblePlans(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	createTestPlan(t, s, "Terlihat 1", 1)
	hidden := createTestPlan(t, s, "Tersembunyi", 1)
	if _, err := s.UpdatePlan(ctx, hidden.ID, store.PlanInput{
		Name: hidden.Name, MaxDevices: hidden.MaxDevices, Visible: false,
	}); err != nil {
		t.Fatalf("sembunyikan plan: %v", err)
	}
	createTestPlan(t, s, "Terlihat 2", 1)

	all, err := s.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListPlans = %d, mau 3 (termasuk yang disembunyikan)", len(all))
	}

	visible, err := s.ListVisiblePlans(ctx)
	if err != nil {
		t.Fatalf("ListVisiblePlans: %v", err)
	}
	if len(visible) != 2 {
		t.Fatalf("ListVisiblePlans = %d, mau 2 (yang tersembunyi tidak ikut)", len(visible))
	}
	for _, p := range visible {
		if p.Name == "Tersembunyi" {
			t.Fatal("plan tersembunyi ikut muncul di ListVisiblePlans")
		}
	}
}

func TestMovePlan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a := createTestPlan(t, s, "A", 1)
	b := createTestPlan(t, s, "B", 1)
	c := createTestPlan(t, s, "C", 1)

	orderedNames := func() []string {
		t.Helper()
		plans, err := s.ListPlans(ctx)
		if err != nil {
			t.Fatalf("ListPlans: %v", err)
		}
		names := make([]string, len(plans))
		for i, p := range plans {
			names[i] = p.Name
		}
		return names
	}

	if got := orderedNames(); got[0] != "A" || got[1] != "B" || got[2] != "C" {
		t.Fatalf("urutan awal = %v, mau [A B C]", got)
	}

	// Tukar B naik -> [B A C].
	if err := s.MovePlan(ctx, b.ID, -1); err != nil {
		t.Fatalf("MovePlan naik: %v", err)
	}
	if got := orderedNames(); got[0] != "B" || got[1] != "A" || got[2] != "C" {
		t.Fatalf("setelah B naik = %v, mau [B A C]", got)
	}

	// A sekarang di tengah, turun -> [B C A].
	if err := s.MovePlan(ctx, a.ID, 1); err != nil {
		t.Fatalf("MovePlan turun: %v", err)
	}
	if got := orderedNames(); got[0] != "B" || got[1] != "C" || got[2] != "A" {
		t.Fatalf("setelah A turun = %v, mau [B C A]", got)
	}

	// B sudah di ujung paling atas -- naik lagi tidak boleh error, dan
	// urutan tidak berubah.
	if err := s.MovePlan(ctx, b.ID, -1); err != nil {
		t.Fatalf("MovePlan di ujung: %v", err)
	}
	if got := orderedNames(); got[0] != "B" || got[1] != "C" || got[2] != "A" {
		t.Fatalf("urutan berubah walau sudah di ujung: %v", got)
	}

	if err := s.MovePlan(ctx, "plan_tidak_ada", -1); !errors.Is(err, store.ErrPlanNotFound) {
		t.Fatalf("err = %v, mau ErrPlanNotFound", err)
	}
	_ = c
}
