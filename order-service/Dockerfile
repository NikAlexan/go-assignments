FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
ENV GONOSUMDB='*'
ENV GOFLAGS='-mod=mod'
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8080 50052
CMD ["./order-service"]