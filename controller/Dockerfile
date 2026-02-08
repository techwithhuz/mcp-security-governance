# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /governance-api ./cmd/api/main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /governance-api /governance-api
EXPOSE 8090
ENTRYPOINT ["/governance-api"]