package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	pb "github.com/nikalexan/go-proto-gen/order"
	"google.golang.org/grpc"

	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

func main() {
	dataSourceName := os.Getenv("DATABASE_URL")
	if dataSourceName == "" {
		log.Fatal("DATABASE_URL is required")
	}

	paymentGRPCAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if paymentGRPCAddr == "" {
		log.Fatal("PAYMENT_GRPC_ADDR is required")
	}

	orderGRPCPort := os.Getenv("ORDER_GRPC_PORT")
	if orderGRPCPort == "" {
		log.Fatal("ORDER_GRPC_PORT is required")
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
	paymentClient, err := repository.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("create payment grpc client: %v", err)
	}

	orderRepository := repository.NewPostgresOrderRepo(database)
	orderUseCase := usecase.NewOrderUseCase(orderRepository, paymentClient)

	// gRPC Server (Order streaming)
	go func() {
		lis, err := net.Listen("tcp", ":"+orderGRPCPort)
		if err != nil {
			log.Fatalf("grpc listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterOrderServiceServer(grpcServer, transportgrpc.NewOrderServer(orderUseCase))
		log.Printf("order-service gRPC listening on :%s", orderGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// HTTP Server (REST — external API)
	handler := transporthttp.NewHandler(orderUseCase)
	router := transporthttp.SetupRouter(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("order-service HTTP listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run: %v", err)
	}
}
