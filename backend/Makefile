TEST_DATABASE_URL ?= postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable
GOOSE ?= $(shell go env GOPATH)/bin/goose

.PHONY: db-up db-down migrate test

db-up:
	docker compose up -d --wait

db-down:
	docker compose down -v

migrate:
	$(GOOSE) -dir migrations postgres "$(TEST_DATABASE_URL)" up

# -p 1 wajib: paket store dan httpapi sama-sama TRUNCATE database test yang
# sama. Tanpa ini Go menjalankan keduanya paralel dan mereka saling menghapus
# data di tengah jalan — gagal secara acak, bukan karena kodenya salah.
test: db-up migrate
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -count=1 -p 1
