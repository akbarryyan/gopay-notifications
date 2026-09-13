package store_test

import (
	"context"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func createTestAPIKey(t *testing.T, s *store.Store, accountID, name string) (id, rawKey string) {
	t.Helper()
	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(context.Background(), accountID, id, name, hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	return id, rawKey
}

func TestVerifyAPIKeyBenar(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")
	_, rawKey := createTestAPIKey(t, s, "acc_1", "Website utama")

	k, err := s.VerifyAPIKey(context.Background(), rawKey)
	if err != nil {
		t.Fatalf("VerifyAPIKey: %v", err)
	}
	if k.Name != "Website utama" {
		t.Fatalf("Name = %s, mau 'Website utama'", k.Name)
	}
	if k.AccountID != "acc_1" {
		t.Fatalf("AccountID = %s, mau acc_1", k.AccountID)
	}
}

func TestVerifyAPIKeySalahDitolak(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")
	createTestAPIKey(t, s, "acc_1", "Website utama")

	_, err := s.VerifyAPIKey(context.Background(), "sk_bukan-key-yang-benar")
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound", err)
	}
}

func TestVerifyAPIKeyYangSudahDicabutDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	id, rawKey := createTestAPIKey(t, s, "acc_1", "Website utama")

	if err := s.RevokeAPIKey(ctx, "acc_1", id); err != nil {
		t.Fatalf("RevokeAPIKey: %v", err)
	}

	_, err := s.VerifyAPIKey(ctx, rawKey)
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound (disamakan dengan key salah)", err)
	}
}

func TestRevokeAPIKeyIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	id, _ := createTestAPIKey(t, s, "acc_1", "Website utama")

	if err := s.RevokeAPIKey(ctx, "acc_1", id); err != nil {
		t.Fatalf("RevokeAPIKey pertama: %v", err)
	}
	if err := s.RevokeAPIKey(ctx, "acc_1", id); err != nil {
		t.Fatalf("RevokeAPIKey kedua (idempotent): %v", err)
	}
}

func TestRevokeAPIKeyTidakDitemukan(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")

	err := s.RevokeAPIKey(context.Background(), "acc_1", "key_tidak_ada")
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound", err)
	}
}

func TestRevokeAPIKeyMilikAccountLainDitolak(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	id, _ := createTestAPIKey(t, s, "acc_a", "Key A")

	err := s.RevokeAPIKey(context.Background(), "acc_b", id)
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound (key milik akun lain)", err)
	}
}

func TestListAPIKeysTidakMembocorkanHash(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")
	createTestAPIKey(t, s, "acc_1", "Website utama")

	keys, err := s.ListAPIKeys(context.Background(), "acc_1")
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len = %d, mau 1", len(keys))
	}
	// store.APIKey memang tidak punya field KeyHash sama sekali — tipe
	// hasil ListAPIKeys secara struktural tidak bisa membocorkannya.
	if keys[0].Name != "Website utama" {
		t.Fatalf("Name = %s, mau 'Website utama'", keys[0].Name)
	}
}

func TestListAPIKeysHanyaMilikAccountSendiri(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	createTestAPIKey(t, s, "acc_a", "Key A")
	createTestAPIKey(t, s, "acc_b", "Key B")

	keysA, err := s.ListAPIKeys(context.Background(), "acc_a")
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keysA) != 1 || keysA[0].Name != "Key A" {
		t.Fatalf("acc_a seharusnya cuma lihat Key A, dapat: %+v", keysA)
	}
}
