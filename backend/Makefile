TEST_DATABASE_URL ?= postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable
GOOSE ?= $(shell go env GOPATH)/bin/goose

.PHONY: db-up db-down migrate test

db-up:
	docker compose up -d --wait

db-down:
	docker compose down -v

migrate:
	$(GOOSE) -dir migrations postgres "$(TEST_DATABASE_URL)" up

test: db-up migrate
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -count=1
