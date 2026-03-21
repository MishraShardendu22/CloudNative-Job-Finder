FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/user-service ./services/user-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/user-service ./services/user-service

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/user-service /app/service
EXPOSE 8081
ENTRYPOINT ["/app/service"]
