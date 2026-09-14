package devicealert_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/devicealert"
	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type fakeStore struct {
	offline, recovered []store.DeviceAlert
	marked, cleared    []string
	logged             []store.NotificationLogEntry
}

func (f *fakeStore) DevicesNeedingOfflineAlert(context.Context, time.Time) ([]store.DeviceAlert, error) {
	return f.offline, nil
}

func (f *fakeStore) DevicesRecoveredFromOffline(context.Context, time.Time) ([]store.DeviceAlert, error) {
	return f.recovered, nil
}

func (f *fakeStore) MarkDeviceOfflineAlerted(_ context.Context, id string, _ time.Time) error {
	f.marked = append(f.marked, id)
	return nil
}

func (f *fakeStore) ClearDeviceOfflineAlert(_ context.Context, id string) error {
	f.cleared = append(f.cleared, id)
	return nil
}

func (f *fakeStore) LogNotification(_ context.Context, e store.NotificationLogEntry) error {
	f.logged = append(f.logged, e)
	return nil
}

type fakeSender struct {
	errFor  map[string]error
	subject []string
}

func (f *fakeSender) SendEmail(_ context.Context, to, subject, _ string) error {
	if err, ok := f.errFor[to]; ok {
		return err
	}
	f.subject = append(f.subject, subject)
	return nil
}

func (f *fakeSender) SendTelegram(context.Context, string, string) error { return nil }
func (f *fakeSender) TelegramEnabled() bool                              { return false }

func enabled(s notify.Sender) notify.SenderFactory {
	return func(context.Context) (notify.Sender, bool, error) { return s, true, nil }
}

func alert(deviceID, email string) store.DeviceAlert {
	return store.DeviceAlert{
		DeviceID: deviceID, DeviceName: "HP " + deviceID, AccountID: "acc_" + deviceID,
		BusinessName: "Toko", Email: email, HeartbeatAt: time.Now().Add(-time.Hour),
	}
}

func TestRunMengabarkanOfflineDanOnline(t *testing.T) {
	st := &fakeStore{
		offline:   []store.DeviceAlert{alert("dev_1", "a@t.test")},
		recovered: []store.DeviceAlert{alert("dev_2", "b@t.test")},
	}
	sender := &fakeSender{}

	res, err := devicealert.New(st, enabled(sender)).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Offline != 1 || res.Online != 1 {
		t.Fatalf("res = %+v, mau 1 offline + 1 online", res)
	}
	if len(st.marked) != 1 || st.marked[0] != "dev_1" || len(st.cleared) != 1 || st.cleared[0] != "dev_2" {
		t.Fatalf("marked=%v cleared=%v", st.marked, st.cleared)
	}
	if len(st.logged) != 2 || st.logged[0].Kind != store.NotificationKindDeviceOffline ||
		st.logged[1].Kind != store.NotificationKindDeviceOnline ||
		st.logged[0].DeviceName == nil || *st.logged[0].DeviceName != "HP dev_1" {
		t.Fatalf("riwayat = %+v", st.logged)
	}
}

func TestRunGagalKirimTidakDitandai(t *testing.T) {
	st := &fakeStore{
		offline:   []store.DeviceAlert{alert("dev_gagal", "gagal@t.test"), alert("dev_ok", "ok@t.test")},
		recovered: []store.DeviceAlert{alert("dev_pulih", "gagal@t.test")},
	}
	sender := &fakeSender{errFor: map[string]error{"gagal@t.test": errors.New("smtp ditolak")}}

	res, err := devicealert.New(st, enabled(sender)).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Offline != 1 || res.Online != 0 {
		t.Fatalf("res = %+v, mau cuma dev_ok terhitung", res)
	}
	// Yang gagal tidak ditandai/dibersihkan -- putaran berikutnya mencoba lagi.
	if len(st.marked) != 1 || st.marked[0] != "dev_ok" || len(st.cleared) != 0 {
		t.Fatalf("marked=%v cleared=%v", st.marked, st.cleared)
	}
}

func TestRunDilewatiBilaSMTPBelumDikonfigurasi(t *testing.T) {
	st := &fakeStore{offline: []store.DeviceAlert{alert("dev_1", "a@t.test")}}
	sender := &fakeSender{}
	disabled := func(context.Context) (notify.Sender, bool, error) { return sender, false, nil }

	res, err := devicealert.New(st, disabled).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Offline != 0 || len(sender.subject) != 0 || len(st.marked) != 0 {
		t.Fatalf("res=%+v terkirim=%d ditandai=%d, mau tidak ada apa pun", res, len(sender.subject), len(st.marked))
	}
}
