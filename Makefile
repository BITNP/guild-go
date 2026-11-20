.PHONY: dev-backend dev-frontend build check test fmt tidy docker

dev-backend:
	go run ./cmd/guild

dev-frontend:
	pnpm dev

build:
	pnpm build
	CGO_ENABLED=0 go build -trimpath -o guild ./cmd/guild

test:
	go test ./...
	pnpm build

check:
	go test ./...
	pnpm check

fmt:
	gofmt -w cmd internal
	pnpm format
	pnpm lint:fix

tidy:
	go mod tidy

docker:
	docker build -t guild-go .
