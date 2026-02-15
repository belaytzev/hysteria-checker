FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hysteria-checker .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 appuser

COPY --from=builder /hysteria-checker /usr/local/bin/hysteria-checker

USER appuser

EXPOSE 2112

ENTRYPOINT ["hysteria-checker"]
