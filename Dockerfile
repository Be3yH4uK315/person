FROM golang:1.27.2-alpine3.24@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/person-api ./cmd/person-api

FROM alpine:3.22.6@sha256:abd29214470819ed7667c87c1ceebc89aae766453a5b3cc09e8a52b9f796fd5a
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /app
COPY --from=builder --chown=app:app /out/person-api ./person-api

USER 10001:10001
EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/person-api"]
