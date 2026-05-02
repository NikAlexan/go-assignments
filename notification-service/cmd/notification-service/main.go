package main

import (
	"log"
	"notification-service/internal/email"
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

	mailer := email.NewSender(os.Getenv("SMTP_FROM"), os.Getenv("SMTP_PASSWORD"))
	if mailer == nil {
		log.Println("notification-service: SMTP not configured, email sending disabled")
	}

	store := idempotency.NewStore()

	consumer, err := messaging.NewConsumer(rabbitmqURL, store, mailer)
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
