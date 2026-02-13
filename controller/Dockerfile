# Build stage
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=${VERSION}" -o /governance-api ./cmd/api/main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /governance-api /governance-api
EXPOSE 8090
ENTRYPOINT ["/governance-api"]