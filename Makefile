.PHONY: help docker-up docker-down docker-logs proto migrate test test-unit test-e2e run dev

SPANNER_EMULATOR_HOST ?= localhost:9010
SPANNER_PROJECT_ID ?= test-project
SPANNER_INSTANCE_ID ?= emulator-instance
SPANNER_DATABASE_ID ?= test-db

SPANNER_DATABASE := projects/$(SPANNER_PROJECT_ID)/instances/$(SPANNER_INSTANCE_ID)/databases/$(SPANNER_DATABASE_ID)

help:
	@echo "Targets:"
	@echo "  docker-up     Start the Spanner emulator (and init container)"
	@echo "  docker-down   Stop the emulator"
	@echo "  docker-logs   Tail emulator logs"
	@echo "  proto         Generate Go code from proto"
	@echo "  migrate       Apply Spanner DDL (requires emulator running)"
	@echo "  test          Run all tests"
	@echo "  test-unit     Run unit tests only"
	@echo "  test-e2e      Run E2E tests only"
	@echo "  run           Start gRPC server"
	@echo "  dev           docker-up + migrate + proto"

docker-up:
	docker compose up -d spanner-emulator spanner-init

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f spanner-emulator

proto:
	protoc -I proto \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  proto/product/v1/product_service.proto

migrate:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	SPANNER_DATABASE=$(SPANNER_DATABASE) \
	go run ./cmd/migrate

test:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) go test ./...

test-unit:
	go test ./internal/...

test-e2e:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) go test ./tests/e2e/...

run:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	SPANNER_DATABASE=$(SPANNER_DATABASE) \
	go run ./cmd/server

dev: docker-up migrate proto
