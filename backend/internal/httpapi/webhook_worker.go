package httpapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// enqueueWebhooks membangun payload dari invoice saat ini lalu menyisipkan
// satu baris webhook_deliveries per endpoint yang berlangganan event itu.
func (a *API) enqueueWebhooks(ctx context.Context, accountID, event, invoiceID string, now time.Time) error {
	inv, err := a.store.GetInvoiceByID(ctx, accountID, invoiceID)
	if err != nil {
		return err
	}
	payload, err := buildInvoicePayload(event, inv, now)
	if err != nil {
		return err
	}
	_, err = a.store.EnqueueWebhookDeliveries(ctx, now, event, &invoiceID, payload)
	return err
}

// triggerInvoiceWebhook dipanggil dari handleCallback (invoice.paid) dan
// dari handleAdminMatchException (invoice.paid manual) lewat goroutine
// terpisah — context sendiri, bukan context request HTTP yang memicunya,
// supaya selesainya request itu tidak ikut membatalkan pengiriman webhook.
func (a *API) triggerInvoiceWebhook(accountID, event, invoiceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := a.now()

	if err := a.enqueueWebhooks(ctx, accountID, event, invoiceID, now); err != nil {
		slog.Error("enqueue webhook gagal", "event", event, "invoice_id", invoiceID, "err", err)
		return
	}
	// Percobaan pertama langsung, bukan menunggu putaran worker berkala
	// berikutnya (spec §3.2/§3.3) — memanggil ProcessDueWebhooks juga
	// otomatis memproses delivery lain yang kebetulan sudah jatuh tempo di
	// saat yang sama, sedikit lebih banyak kerja tapi menghindari
	// duplikasi logic "kirim lalu tulis hasilnya".
	if err := a.ProcessDueWebhooks(ctx, now); err != nil {
		slog.Error("process due webhooks gagal", "err", err)
	}
}

// attemptDelivery mengirim satu delivery lalu menuliskan hasilnya —
// dipakai baik oleh percobaan pertama (triggerInvoiceWebhook) maupun retry
// (ProcessDueWebhooks), supaya logic-nya satu tempat.
func (a *API) attemptDelivery(ctx context.Context, d store.WebhookDeliveryDue, now time.Time) error {
	outcome := a.sendWebhook(ctx, d.EndpointURL, d.EndpointSecret, d.Event, d.Payload)

	if outcome.Success {
		return a.store.RecordDeliverySuccess(ctx, d.ID, now, outcome.HTTPStatus, outcome.DurationMs)
	}

	attempt := d.Attempt + 1
	var httpStatus *int
	if outcome.HTTPStatus != 0 {
		httpStatus = &outcome.HTTPStatus
	}
	return a.store.RecordDeliveryFailure(ctx, d.ID, now, attempt, httpStatus, outcome.DurationMs)
}

// ProcessDueWebhooks adalah satu putaran kerja: expire invoice yang lewat
// waktu (memicu invoice.expired tepat sekali per invoice), lalu proses
// seluruh delivery yang sudah jatuh tempo (percobaan baru maupun retry).
//
// Dipanggil manual dengan now palsu di test, dipanggil oleh time.Ticker di
// cmd/server untuk yang sungguhan (lihat cmd/server/main.go).
func (a *API) ProcessDueWebhooks(ctx context.Context, now time.Time) error {
	// Worker ini lintas akun -- tidak ada satu status account untuk dicek
	// di sini (beda dari requireActiveAccount yang menggerbangi ingestion
	// per akun). Suspend/revoke sebuah akun mencegah invoice BARU dibuat
	// (requireActiveAccount di POST /api/v1/invoices), tapi invoice yang
	// sudah PENDING sebelum disuspend tetap wajar diproses sampai selesai
	// di sini -- itu bukan aktivitas baru, cuma menyelesaikan yang sudah
	// terlanjur berjalan.
	expired, err := a.store.ExpireInvoicesAndListNewlyExpired(ctx, now)
	if err != nil {
		return err
	}
	for _, ref := range expired {
		if err := a.enqueueWebhooks(ctx, ref.AccountID, store.WebhookEventInvoiceExpired, ref.ID, now); err != nil {
			slog.Error("enqueue webhook invoice.expired gagal", "invoice_id", ref.ID, "err", err)
		}
	}

	due, err := a.store.DueWebhookDeliveries(ctx, a.webhookSecretKey, now)
	if err != nil {
		return err
	}
	for _, d := range due {
		if err := a.attemptDelivery(ctx, d, now); err != nil {
			slog.Error("attempt webhook delivery gagal", "delivery_id", d.ID, "err", err)
		}
	}
	return nil
}
