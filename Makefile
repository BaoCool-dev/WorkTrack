# WorkTrack — lệnh chuẩn. CLAUDE.md §4 tham chiếu các target này.
# Các thư mục backend/ và frontend/ sẽ có khi scaffold; target nào chưa có code sẽ báo lỗi bình thường.

SHELL := /bin/bash
BACKEND := backend
FRONTEND := frontend
DATABASE_URL ?= postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable
CORE_PKGS ?= ./internal/domain/... ./internal/application/...

.PHONY: dev dev-local seed test test-integration lint sqlc migrate-up migrate-down migrate-new cover tools

dev:
	docker compose up --build

# Không Docker (B12): API go run + web npm run dev, đọc .env.
dev-local:
	./scripts/dev-local.sh

# Dữ liệu mẫu (B11): 1 tài khoản mỗi vai trò, mật khẩu in ra màn hình.
seed:
	./scripts/seed.sh

test:
	cd $(BACKEND) && go test ./...
	cd $(FRONTEND) && npm run test

test-integration:
	cd $(BACKEND) && go test -tags=integration ./...

lint:
	cd $(BACKEND) && golangci-lint run
	cd $(FRONTEND) && npm run lint && npm run typecheck && npm run format:check

sqlc:
	cd $(BACKEND) && sqlc generate

migrate-up:
	cd $(BACKEND) && goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	cd $(BACKEND) && goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-new:
	@test -n "$(name)" || (echo "Dùng: make migrate-new name=<ten_migration>" && exit 1)
	cd $(BACKEND) && goose -dir db/migrations create $(name) sql

cover:
	cd $(BACKEND) && go test -coverprofile=coverage.out $(CORE_PKGS) && go tool cover -func=coverage.out | tail -1

tools:
	./scripts/setup-dev.sh
