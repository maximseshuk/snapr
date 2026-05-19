# syntax=docker/dockerfile:1

# Frontend build stage
FROM node:22-alpine AS frontend-builder

RUN corepack enable

WORKDIR /app

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json ./web/

RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter web...

COPY web/ ./web/

RUN pnpm --filter web build

# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static' -X main.version=${VERSION}" \
    -o /out/snapr ./cmd/snapr

# Runtime stage
FROM alpine:3.22

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    tar \
    postgresql-client \
    mariadb-client \
    mongodb-tools \
    redis \
    sqlite \
    wget \
    zip \
    openssl \
    pigz \
    zstd \
    xz \
    && addgroup -g 1001 -S snapr \
    && adduser -u 1001 -S snapr -G snapr \
    && mkdir -p /app/data /app/logs \
    && chown -R snapr:snapr /app

WORKDIR /app

COPY --from=builder --chown=snapr:snapr /out/snapr ./snapr
COPY --from=frontend-builder --chown=snapr:snapr /app/web/dist ./web/dist

USER snapr

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/v1/status || exit 1

ENTRYPOINT ["./snapr"]
