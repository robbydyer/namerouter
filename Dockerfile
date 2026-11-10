FROM golang:1.27.3 as builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 \
    GOCACHE=/app/.cache \
    go build -mod=vendor -o /usr/local/bin/namerouter ./cmd/namerouter

FROM gcr.io/distroless/base-debian13

WORKDIR /

COPY --from=builder /usr/local/bin/namerouter /usr/local/bin/namerouter
ENTRYPOINT ["/usr/local/bin/namerouter"]
