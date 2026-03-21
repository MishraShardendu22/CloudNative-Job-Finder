FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/job-scraper ./services/job-scraper
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/job-scraper ./services/job-scraper

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/job-scraper /app/service
EXPOSE 8084
ENTRYPOINT ["/app/service"]
