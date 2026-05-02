FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/notification-service ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/notification-service /usr/local/bin/notification-service
ENTRYPOINT ["/usr/local/bin/notification-service"]
