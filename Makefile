.PHONY: dev-backend dev-frontend build check test fmt tidy docker

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

check:
	go test ./...
	cd web && pnpm check

fmt:
	gofmt -w cmd internal
	cd web && pnpm format
	cd web && pnpm lint:fix

tidy:
	go mod tidy

docker:
	docker build -t guild-go .
