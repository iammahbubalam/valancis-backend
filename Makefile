# ==========================================
# Valancis Backend Makefile
# ==========================================

# Default env file
ENV_FILE ?= .env

# Load environment variables from the specified file
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

# Database URL (uses DB_DSN from loaded env)
DB_URL = $(DB_DSN)

# ==========================================
# Application
# ==========================================

run:
	CONFIG_FILE=$(ENV_FILE) go run cmd/api/main.go

# Production Run (Shortcut)
prod:
	$(MAKE) run ENV_FILE=.env.prod

# Development Run (Shortcut)
# Note: 'dev' previously ran tidy & sqlc. We keep that behavior but force .env.dev
dev:
	$(MAKE) tidy sqlc
	$(MAKE) run ENV_FILE=.env.dev

build:
	go build -o bin/api cmd/api/main.go

tidy:
	go mod tidy

test:
	go test ./... -v

# ==========================================
# Database Migrations
# ==========================================

migrateup:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" up

migrateup-dev:
	$(MAKE) migrateup ENV_FILE=.env.dev

migrateup-prod:
	$(MAKE) migrateup ENV_FILE=.env.prod

migratedown:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" down 1

migratedown-dev:
	$(MAKE) migratedown ENV_FILE=.env.dev

migratedown-prod:
	$(MAKE) migratedown ENV_FILE=.env.prod

migrateforce:
	@read -p "Enter version to force: " version; \
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" force $$version

migratestatus:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" version

migratestatus-dev:
	$(MAKE) migratestatus ENV_FILE=.env.dev

migratestatus-prod:
	$(MAKE) migratestatus ENV_FILE=.env.prod

migratecreate:
	@read -p "Enter migration name: " name; \
	$(HOME)/go/bin/migrate create -ext sql -dir db/migrations -seq $$name

# ==========================================
# Code Generation
# ==========================================

sqlc:
	$(HOME)/go/bin/sqlc generate

generate: sqlc
	@echo "Code generation complete."



lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/

# ==========================================
# Docker (if using)
# ==========================================

docker-build:
	docker build -t valancis-backend .

docker-run:
	docker run -p 8080:8080 --env-file .env valancis-backend

# ==========================================
# Help
# ==========================================

help:
	@echo "Available commands:"
	@echo ""
	@echo "  Application:"
	@echo "    make run            - Run the application"
	@echo "    make build          - Build the binary"
	@echo "    make dev            - Tidy, generate, and run"
	@echo "    make tidy           - Run go mod tidy"
	@echo "    make test           - Run tests"
	@echo ""
	@echo "  Database:"
	@echo "    make migrateup      - Apply all pending migrations"
	@echo "    make migratedown    - Rollback last migration"
	@echo "    make migrateforce   - Force migration version"
	@echo "    make migratestatus  - Show current migration version"
	@echo "    make migratecreate  - Create a new migration"
	@echo ""
	@echo "  Code Generation:"
	@echo "    make sqlc           - Generate SQLC code"
	@echo "    make generate       - Run all code generators"
	@echo ""
	@echo "  Quality:"
	@echo "    make lint           - Run linter"
	@echo "    make fmt            - Format code"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-build   - Build Docker image"
	@echo "    make docker-run     - Run Docker container"

.PHONY: run build tidy test migrateup migratedown migrateforce migratestatus migratecreate sqlc generate dev lint fmt clean docker-build docker-run help
