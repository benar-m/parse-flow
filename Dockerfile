# Single container with Go app + nginx
FROM golang:1.24.4-alpine AS builder

# Build
RUN apk add --no-cache --update gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o parseflow cmd/server/main.go

FROM nginx:alpine

RUN apk add --no-cache supervisor

# Go application
COPY --from=builder /app/parseflow /usr/local/bin/parseflow
COPY --from=builder /app/data/IP2LOCATION-LITE-DB1.IPV6.BIN /app/data/

RUN mkdir -p /etc/nginx/ssl
COPY ssl/cert.pem /etc/nginx/ssl/cert.pem
COPY ssl/key.pem /etc/nginx/ssl/key.pem
COPY nginx.conf /etc/nginx/nginx.conf

# supervisor config
RUN mkdir -p /etc/supervisor/conf.d && \
    echo '[supervisord]' > /etc/supervisor/conf.d/supervisord.conf && \
    echo 'nodaemon=true' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'user=root' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo '' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo '[program:parseflow]' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'command=/usr/local/bin/parseflow' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'environment=PORT=8080,DATABASE_PATH=/app/data/logs.db,RAW_LOG_CHAN_SIZE=2000,PARSED_LOG_CHAN_SIZE=2000,METRIC_CHAN_SIZE=200,BATCH_SIZE=200,FLUSH_INTERVAL=3s,SNAPSHOT_INTERVAL=30s,METRICS_API_KEY=%(ENV_METRICS_API_KEY)s' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo '' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo '[program:nginx]' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'command=nginx -g "daemon off;"' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/supervisord.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/supervisord.conf

RUN mkdir -p /app/data && chmod 755 /app/data && \
    chmod 644 /etc/nginx/ssl/cert.pem && \
    chmod 600 /etc/nginx/ssl/key.pem

EXPOSE 80 443

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
