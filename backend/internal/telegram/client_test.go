package telegram_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/telegram"
)

func TestGetMeDanGetUpdates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot123:abc/getMe":
			w.Write([]byte(`{"ok":true,"result":{"id":1,"username":"whuzpay_bot"}}`))
		case "/bot123:abc/getUpdates":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["offset"].(float64) != 7 {
				t.Errorf("offset = %v, mau 7", body["offset"])
			}
			w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"message":{"chat":{"id":55,"type":"private"},"text":"/start x"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := telegram.NewWithBaseURL(srv.URL, "123:abc")

	me, err := c.GetMe(context.Background())
	if err != nil || me.Username != "whuzpay_bot" {
		t.Fatalf("GetMe = %+v, %v", me, err)
	}
	ups, err := c.GetUpdates(context.Background(), 7, time.Second)
	if err != nil || len(ups) != 1 || ups[0].Message.Chat.ID != 55 || ups[0].Message.Text != "/start x" {
		t.Fatalf("GetUpdates = %+v, %v", ups, err)
	}
}

func TestKonflikDanGalatTidakMembocorkanToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"ok":false,"error_code":409,"description":"Conflict: terminated by other getUpdates request"}`))
	}))
	c := telegram.NewWithBaseURL(srv.URL, "123:rahasia")
	if _, err := c.GetUpdates(context.Background(), 0, time.Second); !errors.Is(err, telegram.ErrConflict) {
		t.Fatalf("err = %v, mau ErrConflict", err)
	}
	srv.Close()

	// Server mati: galat jaringan tidak boleh memuat URL berisi token.
	_, err := c.GetMe(context.Background())
	if err == nil || strings.Contains(err.Error(), "rahasia") {
		t.Fatalf("err = %v, mau galat tanpa token", err)
	}
}
