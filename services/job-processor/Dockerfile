FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY shared ./shared
COPY services/job-processor ./services/job-processor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/job-processor ./services/job-processor

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/job-processor /app/service
EXPOSE 8085
ENTRYPOINT ["/app/service"]
