package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"order-service/internal/repository"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

func main() {
	dataSourceName := os.Getenv("DATABASE_URL")
	if dataSourceName == "" {
		log.Fatal("DATABASE_URL is required")
	}

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		log.Fatal("PAYMENT_SERVICE_URL is required")
	}

	database, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Composition Root — manual dependency injection
	httpClient := &http.Client{Timeout: 2 * time.Second}
	paymentClient := repository.NewPaymentHTTPClient(httpClient, paymentServiceURL)
	orderRepository := repository.NewPostgresOrderRepo(database)
	orderUseCase := usecase.NewOrderUseCase(orderRepository, paymentClient)
	handler := transporthttp.NewHandler(orderUseCase)
	router := transporthttp.SetupRouter(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("order-service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run: %v", err)
	}
}
