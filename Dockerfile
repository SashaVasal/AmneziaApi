FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/amnezia-api ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/amnezia-api /usr/local/bin/amnezia-api

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/amnezia-api"]
