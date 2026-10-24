# syntax=docker/dockerfile:1

FROM node:24-alpine AS web
WORKDIR /src
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./web/
RUN cd web && pnpm install --frozen-lockfile
COPY web ./web
RUN mkdir -p internal/web && cd web && pnpm build

FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=web /src/internal/web/dist ./internal/web/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/guild ./cmd/guild

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app
WORKDIR /app
COPY --from=build /out/guild /usr/local/bin/guild
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/healthz >/dev/null || exit 1
ENTRYPOINT ["guild"]
