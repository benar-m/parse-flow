FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o parseflow cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite

RUN addgroup -g 1001 -S parseflow && \
    adduser -u 1001 -S parseflow -G parseflow

WORKDIR /app
RUN mkdir -p /app/data /app/logs && \
    chown -R parseflow:parseflow /app

COPY --from=builder /app/parseflow .
COPY --chown=parseflow:parseflow data/IP2LOCATION-LITE-DB1.IPV6.BIN ./data/

USER parseflow
EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:5000/metrics || exit 1

ENV PORT=5000 \
    DATABASE_PATH=/app/logs/logs.db \
    RAW_LOG_CHAN_SIZE=1000 \
    PARSED_LOG_CHAN_SIZE=1000 \
    METRIC_CHAN_SIZE=100 \
    BATCH_SIZE=100 \
    FLUSH_INTERVAL=5s \
    SNAPSHOT_INTERVAL=1m

CMD ["./parseflow"]
