package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrPlanNotFound dikembalikan bila id/nama plan tidak terdaftar.
var ErrPlanNotFound = errors.New("store: plan tidak ditemukan")

// ErrPlanNameTaken dikembalikan bila nama plan sudah dipakai plan lain.
var ErrPlanNameTaken = errors.New("store: nama plan sudah dipakai")

// Plan adalah satu paket -- sumber kebenaran ganda: kuota device saat
// account dibuat/diganti plan (MaxDevices), DAN teks yang tampil di section
// Harga landing page publik (semua field selain MaxDevices/Visible).
//
// SENGAJA bukan foreign key dari accounts.plan -- lihat komentar migrasi
// 00017_plans.sql soal kenapa account yang sudah ada tidak boleh ikut
// berubah kalau baris ini nanti diedit/dihapus.
type Plan struct {
	ID          string
	Name        string
	MaxDevices  int // -1 = unlimited
	PriceLabel  string
	PricePeriod string
	Description string
	// Features disimpan sebagai JSONB array string di database, dibaca
	// balik apa adanya -- urutannya dipertahankan Postgres untuk array.
	Features    []string
	Highlighted bool
	// Visible=false berarti disembunyikan dari GET /pricing-plans publik,
	// tapi tetap bisa dipakai vendor membuat/mengubah account.
	Visible   bool
	SortOrder int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func isPlanUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "plans_name_key"
}

const planSelectCols = `SELECT id, name, max_devices, price_label, price_period, description,
	features, highlighted, visible, sort_order, created_at, updated_at FROM plans`

type planScanner interface {
	Scan(dest ...any) error
}

func scanPlan(row planScanner) (Plan, error) {
	var p Plan
	var featuresRaw []byte
	err := row.Scan(&p.ID, &p.Name, &p.MaxDevices, &p.PriceLabel, &p.PricePeriod, &p.Description,
		&featuresRaw, &p.Highlighted, &p.Visible, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Plan{}, err
	}
	p.Features = []string{}
	if len(featuresRaw) > 0 {
		if err := json.Unmarshal(featuresRaw, &p.Features); err != nil {
			return Plan{}, fmt.Errorf("store: unmarshal features plan %s: %w", p.ID, err)
		}
	}
	return p, nil
}

// NewPlanID mengikuti pola ID lain di paket ini: prefix + hex acak.
func NewPlanID() (string, error) {
	return randomPrefixedID("plan_")
}

// ListPlans mengembalikan seluruh plan terurut sort_order lalu nama --
// dipakai halaman kelola paket di Vendor Dashboard (onlyVisible=false) dan
// dropdown plan saat membuat/mengubah account (juga false, vendor boleh
// memilih plan yang disembunyikan dari publik). ListVisiblePlans di bawah
// yang dipakai endpoint publik.
func (s *Store) ListPlans(ctx context.Context) ([]Plan, error) {
	return s.queryPlans(ctx, planSelectCols+` ORDER BY sort_order, name`)
}

// ListVisiblePlans dipakai GET /api/v1/pricing-plans (publik, tanpa auth) --
// hanya plan dengan visible=true, dalam urutan tampil di landing page.
func (s *Store) ListVisiblePlans(ctx context.Context) ([]Plan, error) {
	return s.queryPlans(ctx, planSelectCols+` WHERE visible ORDER BY sort_order, name`)
}

func (s *Store) queryPlans(ctx context.Context, query string) ([]Plan, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("store: list plans: %w", err)
	}
	defer rows.Close()

	out := make([]Plan, 0)
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan plan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi plans: %w", err)
	}
	return out, nil
}

// GetPlanByName dipakai lookup internal saat account dibuat/diganti plan
// (handleVendorCreateAccount/handleVendorChangePlan) dan saat signup
// swalayan mengambil kuota device paket trial ("Starter") -- SENGAJA tidak
// difilter visible, supaya menyembunyikan sebuah plan dari landing page
// tidak diam-diam mematahkan signup atau pembuatan account yang masih
// memakainya.
func (s *Store) GetPlanByName(ctx context.Context, name string) (Plan, error) {
	p, err := scanPlan(s.pool.QueryRow(ctx, planSelectCols+` WHERE name = $1`, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("store: get plan by name: %w", err)
	}
	return p, nil
}

// PlanInput adalah field yang bisa diisi vendor lewat form -- selalu
// menggantikan (bukan tri-state seperti NotificationSettingsUpdate): tidak
// ada nilai di sini yang rahasia/sensitif, jadi form boleh selalu mengirim
// seluruh isinya.
type PlanInput struct {
	Name        string
	MaxDevices  int
	PriceLabel  string
	PricePeriod string
	Description string
	Features    []string
	Highlighted bool
	Visible     bool
}

// CreatePlan menambah plan baru di urutan paling akhir (sort_order
// tertinggi + 1) -- vendor mengurutkan ulang lewat MovePlan setelahnya
// kalau perlu.
func (s *Store) CreatePlan(ctx context.Context, id string, in PlanInput) (Plan, error) {
	features, err := json.Marshal(in.Features)
	if err != nil {
		return Plan{}, fmt.Errorf("store: marshal features: %w", err)
	}
	row := s.pool.QueryRow(ctx,
		`INSERT INTO plans (id, name, max_devices, price_label, price_period, description,
		                     features, highlighted, visible, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
		         COALESCE((SELECT max(sort_order) + 1 FROM plans), 1))
		 RETURNING `+planSelectColsInner,
		id, in.Name, in.MaxDevices, in.PriceLabel, in.PricePeriod, in.Description,
		features, in.Highlighted, in.Visible)
	p, err := scanPlan(row)
	if isPlanUniqueViolation(err) {
		return Plan{}, ErrPlanNameTaken
	}
	if err != nil {
		return Plan{}, fmt.Errorf("store: create plan: %w", err)
	}
	return p, nil
}

// planSelectColsInner adalah daftar kolom polos (tanpa "SELECT ... FROM
// plans") -- dipakai ulang di klausa RETURNING supaya urutannya selalu
// sinkron dengan scanPlan.
const planSelectColsInner = `id, name, max_devices, price_label, price_period, description,
	features, highlighted, visible, sort_order, created_at, updated_at`

// UpdatePlan mengganti seluruh field yang bisa diisi vendor. sort_order
// TIDAK disentuh di sini -- diubah lewat MovePlan.
func (s *Store) UpdatePlan(ctx context.Context, id string, in PlanInput) (Plan, error) {
	features, err := json.Marshal(in.Features)
	if err != nil {
		return Plan{}, fmt.Errorf("store: marshal features: %w", err)
	}
	row := s.pool.QueryRow(ctx,
		`UPDATE plans SET name = $2, max_devices = $3, price_label = $4, price_period = $5,
		                  description = $6, features = $7, highlighted = $8, visible = $9,
		                  updated_at = now()
		 WHERE id = $1
		 RETURNING `+planSelectColsInner,
		id, in.Name, in.MaxDevices, in.PriceLabel, in.PricePeriod, in.Description,
		features, in.Highlighted, in.Visible)
	p, err := scanPlan(row)
	if isPlanUniqueViolation(err) {
		return Plan{}, ErrPlanNameTaken
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("store: update plan: %w", err)
	}
	return p, nil
}

// DeletePlan menghapus definisi plan. Aman terhadap account yang sudah
// memakai namanya -- accounts.plan cuma teks bebas, tidak direferensikan
// balik ke sini (lihat komentar migrasi 00017_plans.sql), dan
// accounts.max_devices sudah tersalin sendiri sejak account itu dibuat.
func (s *Store) DeletePlan(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPlanNotFound
	}
	return nil
}

// MovePlan menukar sort_order dengan tetangga langsung (naik atau turun
// satu posisi) -- pengurutan sederhana ala "panah atas/bawah", bukan
// drag-and-drop bebas yang butuh state lebih rumit di frontend.
func (s *Store) MovePlan(ctx context.Context, id string, direction int) error {
	if direction != -1 && direction != 1 {
		return fmt.Errorf("store: arah pindah plan harus -1 atau 1")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: mulai transaksi pindah plan: %w", err)
	}
	defer tx.Rollback(ctx)

	var thisOrder int
	var thisName string
	if err := tx.QueryRow(ctx, `SELECT sort_order, name FROM plans WHERE id = $1 FOR UPDATE`, id).
		Scan(&thisOrder, &thisName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPlanNotFound
		}
		return fmt.Errorf("store: baca plan: %w", err)
	}

	// Tetangga: sort_order tersebut biasanya beda, tapi bisa juga sama
	// (dua plan pernah dibuat di urutan yang sama) -- name dipakai sebagai
	// pemutus seri supaya arah "naik/turun" tetap konsisten dengan urutan
	// tampil ORDER BY sort_order, name.
	cmp := "<"
	order := "DESC"
	if direction == 1 {
		cmp = ">"
		order = "ASC"
	}
	var neighborID string
	var neighborOrder int
	err = tx.QueryRow(ctx,
		`SELECT id, sort_order FROM plans
		 WHERE (sort_order, name) `+cmp+` ($1, $2)
		 ORDER BY sort_order `+order+`, name `+order+`
		 LIMIT 1 FOR UPDATE`,
		thisOrder, thisName).Scan(&neighborID, &neighborOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // sudah di ujung -- diam saja, bukan error
	}
	if err != nil {
		return fmt.Errorf("store: cari tetangga plan: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE plans SET sort_order = $2, updated_at = now() WHERE id = $1`, id, neighborOrder); err != nil {
		return fmt.Errorf("store: tukar sort_order (plan): %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE plans SET sort_order = $2, updated_at = now() WHERE id = $1`, neighborID, thisOrder); err != nil {
		return fmt.Errorf("store: tukar sort_order (tetangga): %w", err)
	}
	return tx.Commit(ctx)
}
