FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/outbox-relay ./services/outbox-relay
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/outbox-relay ./services/outbox-relay

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/outbox-relay /app/service
EXPOSE 8091
ENTRYPOINT ["/app/service"]
