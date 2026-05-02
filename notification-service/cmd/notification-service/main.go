package main

import (
	"log"
	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	store := idempotency.NewStore()

	consumer, err := messaging.NewConsumer(rabbitmqURL, store)
	if err != nil {
		log.Fatalf("create consumer: %v", err)
	}
	defer consumer.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		<-quit
		log.Println("notification-service: shutting down...")
		close(done)
	}()

	if err := consumer.Start(done); err != nil {
		log.Fatalf("consumer: %v", err)
	}

	log.Println("notification-service: stopped")
}
