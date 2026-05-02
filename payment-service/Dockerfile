FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/payment-service ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/payment-service /usr/local/bin/payment-service
ENTRYPOINT ["/usr/local/bin/payment-service"]
