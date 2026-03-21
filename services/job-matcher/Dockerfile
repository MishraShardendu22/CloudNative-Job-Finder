FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/job-matcher ./services/job-matcher
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/job-matcher ./services/job-matcher

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/job-matcher /app/service
EXPOSE 8086
ENTRYPOINT ["/app/service"]
