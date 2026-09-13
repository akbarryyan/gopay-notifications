package store_test

import (
	"context"
	"testing"
)

func TestLogAuditDanListAuditLog(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.LogAudit(ctx, "akbar", "ACCOUNT_CREATED", "acc_1", map[string]string{"plan": "Business"}); err != nil {
		t.Fatalf("log audit: %v", err)
	}
	if err := s.LogAudit(ctx, "akbar", "ACCOUNT_SUSPENDED", "acc_1", nil); err != nil {
		t.Fatalf("log audit tanpa metadata: %v", err)
	}

	entries, err := s.ListAuditLog(ctx, 50, 0)
	if err != nil {
		t.Fatalf("list audit log: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, mau 2", len(entries))
	}
	// Terbaru dulu.
	if entries[0].Action != "ACCOUNT_SUSPENDED" || entries[1].Action != "ACCOUNT_CREATED" {
		t.Fatalf("urutan salah: %+v", entries)
	}
}
