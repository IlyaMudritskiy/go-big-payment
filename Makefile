-include .env
export

.PHONY: run-ledger test lint migrate-up migrate-down migrate-status

run-ledger:
	go run ./services/ledger/cmd/ledger

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

MIGRATIONS_DIR := services/ledger/migrations

migrate-up:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status
