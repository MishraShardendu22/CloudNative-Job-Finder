FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/scheduler ./services/scheduler
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/scheduler ./services/scheduler

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/scheduler /app/service
EXPOSE 8088
ENTRYPOINT ["/app/service"]
