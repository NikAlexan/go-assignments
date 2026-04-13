FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/payment-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service .
EXPOSE 8081 50051
CMD ["./payment-service"]