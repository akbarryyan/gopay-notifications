package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func webhookKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i*11 + 3)
	}
	return k
}

func createTestWebhook(t *testing.T, s *store.Store, name string, events []string) (id string, secret []byte) {
	t.Helper()
	id, err := store.NewWebhookID()
	if err != nil {
		t.Fatalf("NewWebhookID: %v", err)
	}
	_, secretBytes, err := store.GenerateWebhookSecret()
	if err != nil {
		t.Fatalf("GenerateWebhookSecret: %v", err)
	}
	if err := s.CreateWebhookEndpoint(context.Background(), webhookKey(), id, name, "https://merchant.test/hook", events, secretBytes); err != nil {
		t.Fatalf("CreateWebhookEndpoint: %v", err)
	}
	return id, secretBytes
}

func TestCreateAndGetWebhookEndpointSecretBulatKembali(t *testing.T) {
	s := testStore(t)
	id, secret := createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})

	got, gotSecret, err := s.GetWebhookEndpoint(context.Background(), webhookKey(), id)
	if err != nil {
		t.Fatalf("GetWebhookEndpoint: %v", err)
	}
	if got.Name != "Production" || got.URL != "https://merchant.test/hook" {
		t.Fatalf("endpoint = %+v, tidak sesuai yang dibuat", got)
	}
	if string(gotSecret) != string(secret) {
		t.Fatal("secret yang didekripsi tidak sama dengan yang dienkripsi — enkripsi/dekripsi tidak bulat")
	}
}

func TestGetWebhookEndpointTidakDitemukan(t *testing.T) {
	s := testStore(t)
	_, _, err := s.GetWebhookEndpoint(context.Background(), webhookKey(), "wh_tidak_ada")
	if err != store.ErrWebhookNotFound {
		t.Fatalf("err = %v, mau ErrWebhookNotFound", err)
	}
}

func TestListWebhookEndpointsRingkasanPercobaanTerakhir(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	id, _ := createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})

	list, err := s.ListWebhookEndpoints(ctx)
	if err != nil {
		t.Fatalf("ListWebhookEndpoints: %v", err)
	}
	if len(list) != 1 || list[0].LastDeliveryAt != nil {
		t.Fatalf("list = %+v, mau 1 endpoint tanpa riwayat pengiriman", list)
	}

	if _, err := s.EnqueueTestDelivery(ctx, now, id, []byte(`{}`)); err != nil {
		t.Fatalf("EnqueueTestDelivery: %v", err)
	}

	list, err = s.ListWebhookEndpoints(ctx)
	if err != nil {
		t.Fatalf("ListWebhookEndpoints kedua: %v", err)
	}
	if len(list) != 1 || list[0].LastDeliveryAt == nil || list[0].LastDeliveryStatus == nil {
		t.Fatalf("list = %+v, mau ringkasan percobaan terakhir terisi", list)
	}
	if *list[0].LastDeliveryStatus != store.WebhookDeliveryPending {
		t.Fatalf("LastDeliveryStatus = %s, mau PENDING", *list[0].LastDeliveryStatus)
	}
}

func TestSetWebhookEndpointEnabled(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})

	if err := s.SetWebhookEndpointEnabled(ctx, id, false); err != nil {
		t.Fatalf("SetWebhookEndpointEnabled: %v", err)
	}
	got, _, err := s.GetWebhookEndpoint(ctx, webhookKey(), id)
	if err != nil {
		t.Fatalf("GetWebhookEndpoint: %v", err)
	}
	if got.Enabled {
		t.Fatal("endpoint seharusnya nonaktif")
	}
}

func TestSetWebhookEndpointEnabledTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.SetWebhookEndpointEnabled(context.Background(), "wh_tidak_ada", true)
	if err != store.ErrWebhookNotFound {
		t.Fatalf("err = %v, mau ErrWebhookNotFound", err)
	}
}

func TestDeleteWebhookEndpointCascadeDeliveries(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	id, _ := createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})
	if _, err := s.EnqueueTestDelivery(ctx, now, id, []byte(`{}`)); err != nil {
		t.Fatalf("EnqueueTestDelivery: %v", err)
	}

	if err := s.DeleteWebhookEndpoint(ctx, id); err != nil {
		t.Fatalf("DeleteWebhookEndpoint: %v", err)
	}

	deliveries, err := s.ListWebhookDeliveries(ctx, id, 50, 0)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries: %v", err)
	}
	if len(deliveries) != 0 {
		t.Fatalf("deliveries setelah endpoint dihapus = %d, mau 0 (CASCADE)", len(deliveries))
	}
}

func TestDeleteWebhookEndpointTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.DeleteWebhookEndpoint(context.Background(), "wh_tidak_ada")
	if err != store.ErrWebhookNotFound {
		t.Fatalf("err = %v, mau ErrWebhookNotFound", err)
	}
}

func TestEnqueueWebhookDeliveriesHanyaEndpointYangBerlangganan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()

	subscribed, _ := createTestWebhook(t, s, "Langganan invoice.paid", []string{store.WebhookEventInvoicePaid})
	createTestWebhook(t, s, "Langganan invoice.expired saja", []string{store.WebhookEventInvoiceExpired})

	ids, err := s.EnqueueWebhookDeliveries(ctx, now, store.WebhookEventInvoicePaid, nil, []byte(`{}`))
	if err != nil {
		t.Fatalf("EnqueueWebhookDeliveries: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("len(ids) = %d, mau 1 (hanya endpoint yang berlangganan invoice.paid)", len(ids))
	}

	due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now)
	if err != nil {
		t.Fatalf("DueWebhookDeliveries: %v", err)
	}
	if len(due) != 1 || due[0].EndpointID != subscribed {
		t.Fatalf("due = %+v, mau 1 baris milik endpoint %s", due, subscribed)
	}
}

func TestEnqueueWebhookDeliveriesMelewatiEndpointNonaktif(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()

	id, _ := createTestWebhook(t, s, "Nonaktif", []string{store.WebhookEventInvoicePaid})
	if err := s.SetWebhookEndpointEnabled(ctx, id, false); err != nil {
		t.Fatalf("SetWebhookEndpointEnabled: %v", err)
	}

	ids, err := s.EnqueueWebhookDeliveries(ctx, now, store.WebhookEventInvoicePaid, nil, []byte(`{}`))
	if err != nil {
		t.Fatalf("EnqueueWebhookDeliveries: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("len(ids) = %d, mau 0 — endpoint nonaktif tidak boleh dapat delivery baru", len(ids))
	}
}

func TestDueWebhookDeliveriesTidakMengambilYangBelumJatuhTempo(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})

	future := now.Add(1 * time.Hour)
	if _, err := s.EnqueueWebhookDeliveries(ctx, future, store.WebhookEventInvoicePaid, nil, []byte(`{}`)); err != nil {
		t.Fatalf("EnqueueWebhookDeliveries: %v", err)
	}

	due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now)
	if err != nil {
		t.Fatalf("DueWebhookDeliveries: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %d, mau 0 — next_attempt_at masih di masa depan", len(due))
	}
}

func TestRecordDeliverySuccess(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})
	ids, err := s.EnqueueWebhookDeliveries(ctx, now, store.WebhookEventInvoicePaid, nil, []byte(`{}`))
	if err != nil || len(ids) != 1 {
		t.Fatalf("EnqueueWebhookDeliveries: ids=%v err=%v", ids, err)
	}

	if err := s.RecordDeliverySuccess(ctx, ids[0], now, 200, 42); err != nil {
		t.Fatalf("RecordDeliverySuccess: %v", err)
	}

	due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("DueWebhookDeliveries: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due setelah DELIVERED = %d, mau 0 (tidak boleh diambil lagi)", len(due))
	}
}

func TestRecordDeliveryFailureBackoffDanMenyerah(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})
	ids, err := s.EnqueueWebhookDeliveries(ctx, now, store.WebhookEventInvoicePaid, nil, []byte(`{}`))
	if err != nil || len(ids) != 1 {
		t.Fatalf("EnqueueWebhookDeliveries: ids=%v err=%v", ids, err)
	}
	id := ids[0]
	httpStatus := 500

	wantBackoff := []time.Duration{1, 2, 4, 8, 16}
	for attempt := 1; attempt <= 4; attempt++ {
		if err := s.RecordDeliveryFailure(ctx, id, now, attempt, &httpStatus, 10); err != nil {
			t.Fatalf("RecordDeliveryFailure attempt=%d: %v", attempt, err)
		}
		// Belum jatuh tempo tepat di now — backoff seharusnya menunda ke masa depan.
		due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now)
		if err != nil {
			t.Fatalf("DueWebhookDeliveries: %v", err)
		}
		if len(due) != 0 {
			t.Fatalf("attempt=%d: due di waktu now = %d, mau 0 (masih menunggu backoff)", attempt, len(due))
		}
		due, err = s.DueWebhookDeliveries(ctx, webhookKey(), now.Add(wantBackoff[attempt-1]*time.Minute+time.Second))
		if err != nil {
			t.Fatalf("DueWebhookDeliveries setelah backoff: %v", err)
		}
		if len(due) != 1 {
			t.Fatalf("attempt=%d: due setelah %s = %d, mau 1", attempt, wantBackoff[attempt-1]*time.Minute, len(due))
		}
	}

	// Percobaan ke-5: menyerah permanen, FAILED, tidak lagi diambil DueWebhookDeliveries.
	if err := s.RecordDeliveryFailure(ctx, id, now, 5, &httpStatus, 10); err != nil {
		t.Fatalf("RecordDeliveryFailure attempt=5: %v", err)
	}
	due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("DueWebhookDeliveries final: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due setelah attempt ke-5 = %d, mau 0 (FAILED permanen, tidak di-retry lagi)", len(due))
	}
}

func TestExpireInvoicesAndListNewlyExpiredHanyaSekaliPerInvoice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	inv, _, err := s.CreateInvoice(ctx, past, "ORDER-expire-webhook", 30000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	now := time.Now()
	first, err := s.ExpireInvoicesAndListNewlyExpired(ctx, now)
	if err != nil {
		t.Fatalf("ExpireInvoicesAndListNewlyExpired pertama: %v", err)
	}
	if len(first) != 1 || first[0] != inv.ID {
		t.Fatalf("panggilan pertama = %v, mau [%s]", first, inv.ID)
	}

	second, err := s.ExpireInvoicesAndListNewlyExpired(ctx, now)
	if err != nil {
		t.Fatalf("ExpireInvoicesAndListNewlyExpired kedua: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("panggilan kedua = %v, mau kosong (sudah EXPIRED, bukan baru lagi)", second)
	}
}

func TestEnqueueAndRecordTestDeliveryTidakPernahRetrying(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	id, _ := createTestWebhook(t, s, "Production", []string{store.WebhookEventInvoicePaid})

	deliveryID, err := s.EnqueueTestDelivery(ctx, now, id, []byte(`{"event":"test"}`))
	if err != nil {
		t.Fatalf("EnqueueTestDelivery: %v", err)
	}

	// Test gagal HARUS langsung FAILED, tidak pernah RETRYING — admin sudah
	// melihat hasilnya seketika, mengulang otomatis nanti hanya mengejutkan.
	if err := s.RecordTestDeliveryResult(ctx, deliveryID, now, false, 500, 12); err != nil {
		t.Fatalf("RecordTestDeliveryResult: %v", err)
	}
	due, err := s.DueWebhookDeliveries(ctx, webhookKey(), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("DueWebhookDeliveries: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due setelah test delivery gagal = %d, mau 0 (tidak pernah di-retry)", len(due))
	}

	deliveries, err := s.ListWebhookDeliveries(ctx, id, 50, 0)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries: %v", err)
	}
	if len(deliveries) != 1 || deliveries[0].Status != store.WebhookDeliveryFailed {
		t.Fatalf("deliveries = %+v, mau 1 baris FAILED", deliveries)
	}
	if deliveries[0].InvoiceID != nil {
		t.Fatal("InvoiceID delivery test seharusnya nil — tidak pernah menyentuh invoice sungguhan")
	}
}
