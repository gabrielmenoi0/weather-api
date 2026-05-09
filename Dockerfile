# ── Stage 1: build ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate swagger docs before building
RUN go install github.com/swaggo/swag/cmd/swag@latest && swag init -g cmd/api/main.go -o docs

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /weather-api ./cmd/api

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /weather-api /weather-api

EXPOSE 8080

ENTRYPOINT ["/weather-api"]
