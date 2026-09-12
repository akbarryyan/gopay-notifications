package store_test

import (
	"context"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func createTestAPIKey(t *testing.T, s *store.Store, name string) (id, rawKey string) {
	t.Helper()
	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(context.Background(), id, name, hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	return id, rawKey
}

func TestVerifyAPIKeyBenar(t *testing.T) {
	s := testStore(t)
	_, rawKey := createTestAPIKey(t, s, "Website utama")

	k, err := s.VerifyAPIKey(context.Background(), rawKey)
	if err != nil {
		t.Fatalf("VerifyAPIKey: %v", err)
	}
	if k.Name != "Website utama" {
		t.Fatalf("Name = %s, mau 'Website utama'", k.Name)
	}
}

func TestVerifyAPIKeySalahDitolak(t *testing.T) {
	s := testStore(t)
	createTestAPIKey(t, s, "Website utama")

	_, err := s.VerifyAPIKey(context.Background(), "sk_bukan-key-yang-benar")
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound", err)
	}
}

func TestVerifyAPIKeyYangSudahDicabutDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, rawKey := createTestAPIKey(t, s, "Website utama")

	if err := s.RevokeAPIKey(ctx, id); err != nil {
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
	id, _ := createTestAPIKey(t, s, "Website utama")

	if err := s.RevokeAPIKey(ctx, id); err != nil {
		t.Fatalf("RevokeAPIKey pertama: %v", err)
	}
	if err := s.RevokeAPIKey(ctx, id); err != nil {
		t.Fatalf("RevokeAPIKey kedua (idempotent): %v", err)
	}
}

func TestRevokeAPIKeyTidakDitemukan(t *testing.T) {
	s := testStore(t)

	err := s.RevokeAPIKey(context.Background(), "key_tidak_ada")
	if err != store.ErrAPIKeyNotFound {
		t.Fatalf("err = %v, mau ErrAPIKeyNotFound", err)
	}
}

func TestListAPIKeysTidakMembocorkanHash(t *testing.T) {
	s := testStore(t)
	createTestAPIKey(t, s, "Website utama")

	keys, err := s.ListAPIKeys(context.Background())
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
