FROM golang:1.24.5 as builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 \
    GOCACHE=/app/.cache \
    go build -mod=vendor -o /usr/local/bin/namerouter ./cmd/namerouter

FROM alpine:3.22.0

WORKDIR /

COPY --from=builder /usr/local/bin/namerouter /usr/local/bin/namerouter
RUN chmod +x  /usr/local/bin/namerouter
