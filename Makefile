.PHONY: dev-infra-up dev-infra-down api frontend test

dev-infra-up:
	docker compose -f compose.dev.yml up -d

dev-infra-down:
	docker compose -f compose.dev.yml down

api:
	DATABASE_URL="$${DATABASE_URL:-postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable}" go run ./cmd/api

frontend:
	TG_GROUP_API_PROXY="$${TG_GROUP_API_PROXY:-http://127.0.0.1:8080}" pnpm dev

test:
	go test ./...
	go vet ./...
