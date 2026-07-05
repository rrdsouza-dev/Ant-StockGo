# ── Build stage ────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache de dependências: só reexecuta `go mod download` quando go.mod/go.sum mudam.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wms-backend main.go

# ── Runtime stage ──────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /app/wms-backend .
COPY database/migrations ./database/migrations

EXPOSE 8000
ENTRYPOINT ["./wms-backend"]
