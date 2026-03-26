# Build stage
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=${VERSION}" -o /governance-api ./cmd/api/main.go

# Runtime stage — Tier 2 #15: Hardened controller image (USER nonroot, read-only fs)
FROM alpine:3.19
RUN apk add --no-cache ca-certificates && \
    addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot
COPY --from=builder /governance-api /governance-api
# Ensure the binary is owned by nonroot and not world-writable
RUN chown 65532:65532 /governance-api && chmod 550 /governance-api
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/governance-api"]