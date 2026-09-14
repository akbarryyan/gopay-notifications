package telegrambot_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
	"github.com/akbarryyan/gopay-notifications/backend/internal/telegram"
	"github.com/akbarryyan/gopay-notifications/backend/internal/telegrambot"
)

type fakeStore struct {
	token      string
	offset     int64
	validCode  string
	consumeErr error
	linked     map[string]string // chat id -> code
}

func (f *fakeStore) GetNotificationSettings(context.Context, []byte) (store.NotificationSettings, error) {
	return store.NotificationSettings{TelegramBotToken: f.token}, nil
}
func (f *fakeStore) GetTelegramUpdateOffset(context.Context) (int64, error) { return f.offset, nil }
func (f *fakeStore) SetTelegramUpdateOffset(_ context.Context, o int64) error {
	f.offset = o
	return nil
}
func (f *fakeStore) ConsumeTelegramLinkCode(_ context.Context, hash []byte, chatID string, _ time.Time) (store.Account, error) {
	if f.consumeErr != nil {
		return store.Account{}, f.consumeErr
	}
	want := sha256.Sum256([]byte(f.validCode))
	if string(hash) != string(want[:]) {
		return store.Account{}, store.ErrTelegramLinkInvalid
	}
	if f.linked == nil {
		f.linked = map[string]string{}
	}
	f.linked[chatID] = f.validCode
	return store.Account{ID: "acc_1", BusinessName: "Toko Uji"}, nil
}

type sent struct {
	chatID int64
	text   string
}

type fakeClient struct {
	updates   []telegram.Update
	gotOffset int64
	sent      []sent
}

func (f *fakeClient) GetUpdates(_ context.Context, offset int64, _ time.Duration) ([]telegram.Update, error) {
	f.gotOffset = offset
	return f.updates, nil
}
func (f *fakeClient) SendMessage(_ context.Context, chatID int64, text string) error {
	f.sent = append(f.sent, sent{chatID, text})
	return nil
}

func msg(id int64, chatID int64, chatType, text string) telegram.Update {
	return telegram.Update{UpdateID: id, Message: &telegram.Message{Chat: telegram.Chat{ID: chatID, Type: chatType}, Text: text}}
}

func newBot(st *fakeStore, c *fakeClient) *telegrambot.Bot {
	return telegrambot.NewWithClient(st, nil, func(string) telegrambot.Client { return c }, time.Now)
}

func TestPollOnceTanpaTokenTidakMembaca(t *testing.T) {
	c := &fakeClient{}
	polled, err := newBot(&fakeStore{}, c).PollOnce(context.Background())
	if err != nil || polled || c.gotOffset != 0 {
		t.Fatalf("polled=%v err=%v, mau tidak membaca apa pun", polled, err)
	}
}

func TestPollOnceMenautkanDanMembalas(t *testing.T) {
	st := &fakeStore{token: "1:x", offset: 10, validCode: "KODE_OK"}
	c := &fakeClient{updates: []telegram.Update{
		msg(10, 555, "private", "/start KODE_OK"),
		msg(11, 666, "private", "/start KODE_SALAH"),
		msg(12, 777, "private", "/start"),
		msg(13, 888, "group", "/start KODE_OK"),
		msg(14, 999, "private", "halo bot"),
	}}

	polled, err := newBot(st, c).PollOnce(context.Background())
	if err != nil || !polled {
		t.Fatalf("PollOnce: polled=%v err=%v", polled, err)
	}
	if c.gotOffset != 10 {
		t.Fatalf("getUpdates offset = %d, mau 10 dari database", c.gotOffset)
	}
	if st.offset != 15 {
		t.Fatalf("offset tersimpan = %d, mau 15 (update terakhir + 1)", st.offset)
	}
	if len(st.linked) != 1 || st.linked["555"] != "KODE_OK" {
		t.Fatalf("tertaut = %v, mau cuma chat 555 (grup tidak boleh)", st.linked)
	}

	replies := map[int64]string{}
	for _, s := range c.sent {
		replies[s.chatID] = s.text
	}
	for chat, want := range map[int64]string{
		555: "Terhubung ke akun Toko Uji",
		666: "sudah tidak berlaku",
		777: "Hubungkan Telegram",
		888: "obrolan pribadi",
	} {
		if !strings.Contains(replies[chat], want) {
			t.Fatalf("balasan ke %d = %q, mau memuat %q", chat, replies[chat], want)
		}
	}
	if _, ok := replies[999]; ok {
		t.Fatal("pesan biasa dibalas -- bot ini bukan bot percakapan")
	}
}

func TestPollOnceGagalDatabaseTidakMemajukanOffset(t *testing.T) {
	st := &fakeStore{token: "1:x", offset: 20, validCode: "KODE_OK", consumeErr: errors.New("database mati")}
	c := &fakeClient{updates: []telegram.Update{msg(20, 555, "private", "/start KODE_OK")}}

	if _, err := newBot(st, c).PollOnce(context.Background()); err == nil {
		t.Fatal("err = nil, mau galat database diteruskan")
	}
	if st.offset != 20 {
		t.Fatalf("offset = %d, mau tetap 20 supaya pesan dicoba lagi", st.offset)
	}
}
