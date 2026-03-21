FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/recommendation-service ./services/recommendation-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/recommendation-service ./services/recommendation-service

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/recommendation-service /app/service
EXPOSE 8087
ENTRYPOINT ["/app/service"]
