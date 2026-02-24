FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/awg-api ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN mkdir -p /data /app/config

COPY --from=builder /out/awg-api /usr/local/bin/awg-api
COPY config /app/config

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/awg-api"]
