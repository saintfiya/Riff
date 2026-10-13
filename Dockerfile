FROM golang:1.25.8-alpine3.23 AS builder

ARG GOPROXY=https://goproxy.cn,direct

ENV CGO_ENABLED=0 \
    GOTOOLCHAIN=local \
    GOPROXY=${GOPROXY}

WORKDIR /src

# Cache dependencies separately from the application source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /tmp/riff .

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S riff \
    && adduser -S -G riff riff \
    && mkdir -p /app \
    && chown riff:riff /app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder --chown=riff:riff /tmp/riff ./riff
COPY --chown=riff:riff conf ./conf
COPY --chown=riff:riff static ./static
COPY --chown=riff:riff templates ./templates

USER riff

EXPOSE 8888

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8888/ping || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["./riff"]
