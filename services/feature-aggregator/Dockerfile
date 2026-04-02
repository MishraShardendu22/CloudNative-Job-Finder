FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/feature-aggregator ./services/feature-aggregator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/feature-aggregator ./services/feature-aggregator

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/feature-aggregator /app/service
EXPOSE 8092
ENTRYPOINT ["/app/service"]
