.PHONY: dev-backend dev-frontend build test fmt tidy docker

dev-backend:
	go run ./cmd/guild

dev-frontend:
	cd web && pnpm dev

build:
	cd web && pnpm build
	CGO_ENABLED=0 go build -trimpath -o guild ./cmd/guild

test:
	go test ./...
	cd web && pnpm build

fmt:
	gofmt -w cmd internal

tidy:
	go mod tidy

docker:
	docker build -t guild-go .