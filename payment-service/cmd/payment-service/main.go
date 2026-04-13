package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"payment-service/internal/repository"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func main() {
	dataSourceName := os.Getenv("DATABASE_URL")
	if dataSourceName == "" {
		log.Fatal("DATABASE_URL is required")
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
	paymentRepository := repository.NewPostgresPaymentRepo(database)
	paymentUseCase := usecase.NewPaymentUseCase(paymentRepository)
	handler := transporthttp.NewHandler(paymentUseCase)
	router := transporthttp.SetupRouter(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("payment-service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run: %v", err)
	}
}
