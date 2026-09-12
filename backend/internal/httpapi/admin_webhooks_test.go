package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type webhookBody struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

func createWebhookReq(t *testing.T, h http.Handler, cookie *http.Cookie, name, url string, events []string) *httptest.ResponseRecorder {
	t.Helper()
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}
	body := `{"name":"` + name + `","url":"` + url + `","events":` + string(eventsJSON) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminCreateWebhookMengembalikanSecretSekali(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := createWebhookReq(t, h, cookie, "Production", "https://merchant.test/hook", []string{"invoice.paid", "invoice.expired"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var body webhookBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Secret == "" {
		t.Fatal("secret mentah kosong di response create — harus ada, hanya sekali ini")
	}
	if len(body.Events) != 2 {
		t.Fatalf("Events = %v, mau 2", body.Events)
	}
}

func TestAdminCreateWebhookValidasi(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	cases := []struct {
		name, url string
		events    []string
	}{
		{"", "https://x.test", []string{"invoice.paid"}},
		{"Production", "", []string{"invoice.paid"}},
		{"Production", "https://x.test", nil},
		{"Production", "https://x.test", []string{"payment.received"}},
	}
	for _, c := range cases {
		rec := createWebhookReq(t, h, cookie, c.name, c.url, c.events)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("case %+v: status = %d, mau 400 (body=%s)", c, rec.Code, rec.Body.String())
		}
	}
}

func TestAdminListWebhooksTidakMenyertakanSecret(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	createWebhookReq(t, h, cookie, "Production", "https://merchant.test/hook", []string{"invoice.paid"})

	rec := adminGet(t, h, cookie, "/api/v1/admin/webhooks")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"secret":`) {
		t.Fatalf("respons list membocorkan secret: %s", rec.Body.String())
	}
}

func TestAdminSetWebhookEnabled(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	created := createWebhookReq(t, h, cookie, "Production", "https://merchant.test/hook", []string{"invoice.paid"})
	var body webhookBody
	json.Unmarshal(created.Body.Bytes(), &body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/webhooks/"+body.ID, strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestAdminSetWebhookEnabledTidakDitemukan(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/webhooks/wh_tidak_ada", strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}

func TestAdminDeleteWebhook(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	created := createWebhookReq(t, h, cookie, "Production", "https://merchant.test/hook", []string{"invoice.paid"})
	var body webhookBody
	json.Unmarshal(created.Body.Bytes(), &body)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/webhooks/"+body.ID, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	listRec := adminGet(t, h, cookie, "/api/v1/admin/webhooks")
	var list struct {
		Webhooks []webhookBody `json:"webhooks"`
	}
	json.Unmarshal(listRec.Body.Bytes(), &list)
	if len(list.Webhooks) != 0 {
		t.Fatalf("webhooks setelah dihapus = %d, mau 0", len(list.Webhooks))
	}
}

func TestAdminTestWebhookMengirimKeEndpointSungguhan(t *testing.T) {
	var gotEvent, gotSignature string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEvent = r.Header.Get("X-Webhook-Event")
		gotSignature = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()

	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	created := createWebhookReq(t, h, cookie, "Production", fake.URL, []string{"invoice.paid"})
	var body webhookBody
	json.Unmarshal(created.Body.Bytes(), &body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks/"+body.ID+"/test", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var result struct {
		Delivered  bool `json:"delivered"`
		HTTPStatus int  `json:"http_status"`
	}
	json.Unmarshal(rec.Body.Bytes(), &result)
	if !result.Delivered || result.HTTPStatus != 200 {
		t.Fatalf("result = %+v, mau delivered=true http_status=200", result)
	}
	if gotEvent != "test" {
		t.Fatalf("X-Webhook-Event yang diterima merchant = %q, mau test", gotEvent)
	}
	if gotSignature == "" {
		t.Fatal("X-Webhook-Signature kosong — payload test tetap harus ditandatangani")
	}

	deliveries := adminGet(t, h, cookie, "/api/v1/admin/webhooks/"+body.ID+"/deliveries")
	var dbody struct {
		Deliveries []struct {
			Event     string  `json:"event"`
			Status    string  `json:"status"`
			InvoiceID *string `json:"invoice_id"`
		} `json:"deliveries"`
	}
	json.Unmarshal(deliveries.Body.Bytes(), &dbody)
	if len(dbody.Deliveries) != 1 || dbody.Deliveries[0].Status != "DELIVERED" {
		t.Fatalf("deliveries = %+v, mau 1 baris DELIVERED", dbody.Deliveries)
	}
	if dbody.Deliveries[0].InvoiceID != nil {
		t.Fatal("delivery test tidak boleh terkait invoice_id manapun")
	}
}

func TestAdminTestWebhookEndpointTidakDitemukan(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks/wh_tidak_ada/test", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}

func TestAdminWebhooksMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/webhooks", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks", strings.NewReader(`{}`)),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, mau 401", req.Method, req.URL.Path, rec.Code)
		}
	}
}
