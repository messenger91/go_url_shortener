FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO is disabled: the modernc.org/sqlite driver is pure Go,
# so the binary is fully static
RUN CGO_ENABLED=0 GOOS=linux go build -o url-shortener ./cmd/url-shortener

FROM alpine:3.20

WORKDIR /app

RUN adduser -D -H appuser

# writable location for the SQLite file; STORAGE_PATH defaults to it
RUN mkdir /data && chown appuser /data
USER appuser

COPY --from=builder /app/url-shortener ./url-shortener
COPY --from=builder /app/config ./config

ENV CONFIG_PATH=./config/local.yaml
ENV STORAGE_PATH=/data/storage.db
# bind all interfaces so the published port is reachable
ENV HTTP_SERVER_ADDRESS=0.0.0.0:8082

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8082/health || exit 1

ENTRYPOINT ["./url-shortener"]
