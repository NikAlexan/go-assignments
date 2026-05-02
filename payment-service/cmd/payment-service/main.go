package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	pb "github.com/nikalexan/go-proto-gen/payment"
	"google.golang.org/grpc"

	"payment-service/internal/messaging"
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

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	database, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	publisher, err := messaging.NewRabbitMQPublisher(rabbitmqURL)
	if err != nil {
		log.Fatalf("create rabbitmq publisher: %v", err)
	}
	defer publisher.Close()

	// Composition Root
	paymentRepository := repository.NewPostgresPaymentRepo(database)
	paymentUseCase := usecase.NewPaymentUseCase(paymentRepository, publisher, os.Getenv("DEFAULT_CUSTOMER_EMAIL"))

	// gRPC Server
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(transportgrpc.LoggingInterceptor))
	pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentServer(paymentUseCase))

	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("grpc listen: %v", err)
		}
		log.Printf("payment-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// HTTP Server
	handler := transporthttp.NewHandler(paymentUseCase)
	router := transporthttp.SetupRouter(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("payment-service HTTP listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("payment-service: shutting down...")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("http shutdown: %v", err)
	}

	log.Println("payment-service: stopped")
}
