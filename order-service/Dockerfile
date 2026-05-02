FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/order-service ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/order-service /usr/local/bin/order-service
ENTRYPOINT ["/usr/local/bin/order-service"]
