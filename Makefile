include .env
export

.PHONY: run stop build restart logs logs-order logs-payment ps clean tidy stream help

## run: Start all services (build images if not present)
run:
	docker compose up -d

## stop: Stop and remove containers
stop:
	docker compose down

## build: Force rebuild all images
build:
	docker compose build --no-cache

## restart: Restart all services
restart: stop run

## logs: Follow logs of all services
logs:
	docker compose logs -f

## logs-order: Follow order-service logs
logs-order:
	docker compose logs -f order-service

## logs-payment: Follow payment-service logs
logs-payment:
	docker compose logs -f payment-service

## ps: Show container status
ps:
	docker compose ps

## clean: Stop containers and remove volumes (WARNING - deletes all DB data)
clean:
	docker compose down -v

## tidy: Run go mod tidy in all modules via Docker
tidy:
	docker run --rm -v "$(CURDIR)/order-service":/app -v "$(CURDIR)/proto-gen":/proto-gen -w /app golang:1.26-alpine go mod tidy
	docker run --rm -v "$(CURDIR)/payment-service":/app -v "$(CURDIR)/proto-gen":/proto-gen -w /app golang:1.26-alpine go mod tidy
	docker run --rm -v "$(CURDIR)/streaming-client":/app -v "$(CURDIR)/proto-gen":/proto-gen -w /app golang:1.26-alpine go mod tidy

## stream: Subscribe to order status updates via gRPC streaming. Usage: make stream ORDER_ID=<id>
stream:
	@test -n "$(ORDER_ID)" || (echo "Usage: make stream ORDER_ID=<order-id>" && exit 1)
	docker run --rm --network host \
	  -v "$(CURDIR)/streaming-client":/app \
	  -w /app golang:1.26-alpine \
	  go run main.go $(ORDER_ID)

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
