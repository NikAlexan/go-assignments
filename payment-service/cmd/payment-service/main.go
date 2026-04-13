package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	pb "github.com/nikalexan/go-proto-gen/payment"
	"google.golang.org/grpc"

	"payment-service/internal/repository"
	transportgrpc "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func main() {
	dataSourceName := os.Getenv("DATABASE_URL")
	if dataSourceName == "" {
		log.Fatal("DATABASE_URL is required")
	}

	grpcPort := os.Getenv("PAYMENT_GRPC_PORT")
	if grpcPort == "" {
		log.Fatal("PAYMENT_GRPC_PORT is required")
	}

	database, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Composition Root
	paymentRepository := repository.NewPostgresPaymentRepo(database)
	paymentUseCase := usecase.NewPaymentUseCase(paymentRepository)

	// gRPC Server
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("grpc listen: %v", err)
		}
		grpcServer := grpc.NewServer(grpc.UnaryInterceptor(transportgrpc.LoggingInterceptor))
		pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentServer(paymentUseCase))
		log.Printf("payment-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// HTTP Server (REST — kept for backwards compatibility)
	handler := transporthttp.NewHandler(paymentUseCase)
	router := transporthttp.SetupRouter(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("payment-service HTTP listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run: %v", err)
	}
}
