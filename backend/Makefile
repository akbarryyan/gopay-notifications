TEST_DATABASE_URL ?= postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable
DEV_DATABASE_URL  ?= postgres://gopay:gopay@localhost:5433/gopay_dev?sslmode=disable
GOOSE ?= $(shell go env GOPATH)/bin/goose

.PHONY: db-up db-down migrate test db-dev run-dev dev-device dev-key dev-url dev-admin

db-up:
	docker compose up -d --wait

# Menghapus volume — seluruh data dev ikut hilang, bukan hanya data test.
db-down:
	docker compose down -v

migrate:
	$(GOOSE) -dir migrations postgres "$(TEST_DATABASE_URL)" up

# -p 1 wajib: paket store dan httpapi sama-sama TRUNCATE database test yang
# sama. Tanpa ini Go menjalankan keduanya paralel dan mereka saling menghapus
# data di tengah jalan — gagal secara acak, bukan karena kodenya salah.
test: db-up migrate
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -count=1 -p 1

# --- Development ------------------------------------------------------------
# gopay_dev terpisah dari gopay_test dengan sengaja: `make test` menjalankan
# TRUNCATE, jadi event sungguhan dari HP akan terhapus bila keduanya disatukan.

db-dev: db-up
	@docker compose exec -T postgres psql -U gopay -d postgres -tAc \
	  "SELECT 1 FROM pg_database WHERE datname='gopay_dev'" | grep -q 1 \
	  || docker compose exec -T postgres createdb -U gopay gopay_dev
	$(GOOSE) -dir migrations postgres "$(DEV_DATABASE_URL)" up

dev-key:
	@go run ./cmd/devicetool -genkey

run-dev: db-dev
	@test -f .env.dev || { \
	  echo "Belum ada .env.dev. Jalankan:"; \
	  echo "  cp .env.dev.example .env.dev"; \
	  echo "  echo \"DEVICE_SECRET_KEY=\$$(make -s dev-key)\" >> .env.dev"; \
	  echo "  echo \"ADMIN_SESSION_KEY=\$$(make -s dev-key)\" >> .env.dev"; \
	  echo "  echo \"WEBHOOK_SECRET_KEY=\$$(make -s dev-key)\" >> .env.dev"; \
	  echo "  make dev-admin   # buat akun admin dashboard"; \
	  exit 1; }
	@port=$$(grep -E '^LISTEN_ADDR=' .env.dev | tail -1 | sed 's/.*://'); \
	 if ss -ltnH "sport = :$$port" 2>/dev/null | grep -q .; then \
	   echo "Port $$port sudah dipakai proses lain:"; \
	   docker ps --format '  container {{.Names}} ({{.Image}})' --filter "publish=$$port" 2>/dev/null; \
	   ss -ltnpH "sport = :$$port" 2>/dev/null | sed 's/^/  /'; \
	   echo; \
	   echo "Server tidak dijalankan. Ganti LISTEN_ADDR di .env.dev ke port lain,"; \
	   echo "lalu perbarui Backend URL di layar Pengaturan HP ('make dev-url')."; \
	   exit 1; \
	 fi
	@set -a; . ./.env.dev; set +a; go run ./cmd/server

dev-device: db-dev
	@test -f .env.dev || { echo "Belum ada .env.dev — lihat 'make run-dev'"; exit 1; }
	@set -a; . ./.env.dev; set +a; go run ./cmd/devicetool -name "HP GoPay Dev"

# Membuat atau mereset password admin dashboard. Interaktif — akan meminta
# password diketik dua kali tanpa ditampilkan di terminal.
dev-admin: db-dev
	@test -f .env.dev || { echo "Belum ada .env.dev — lihat 'make run-dev'"; exit 1; }
	@set -a; . ./.env.dev; set +a; go run ./cmd/admintool

# Alamat yang diketik ke layar Pengaturan di HP. Alamat LAN laptop berubah tiap
# ganti jaringan, jadi jangan dihafal.
dev-url:
	@ip=$$(ip -4 addr show 2>/dev/null | grep -oP 'inet \K[0-9.]+' \
	       | grep -vE '^(127\.|172\.1[6-9]\.|172\.2[0-9]\.|172\.3[01]\.)' | head -1); \
	 if [ -z "$$ip" ]; then echo "Alamat LAN tidak terdeteksi."; exit 1; fi; \
	 port=$$(grep -E '^LISTEN_ADDR=' .env.dev 2>/dev/null | tail -1 | sed 's/.*://'); \
	 port=$${port:-8090}; \
	 echo "Backend URL untuk layar Pengaturan di HP:"; \
	 echo; \
	 echo "    http://$$ip:$$port/api/v1"; \
	 echo; \
	 echo "Port dibaca dari .env.dev supaya tidak pernah menyimpang dari server."; \
	 echo
