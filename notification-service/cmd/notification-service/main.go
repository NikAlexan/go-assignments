package main

import (
	"log"
	"notification-service/internal/email"
	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"notification-service/internal/notifier"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/redis/go-redis/v9"
)

func main() {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	store := idempotency.NewRedisStore(rdb)

	maxRetries, _ := strconv.Atoi(os.Getenv("RETRY_MAX"))
	if maxRetries <= 0 {
		maxRetries = 3
	}
	baseDelayMs, _ := strconv.Atoi(os.Getenv("RETRY_BASE_DELAY_MS"))
	if baseDelayMs <= 0 {
		baseDelayMs = 2000
	}

	// Select email sender based on PROVIDER_MODE
	var sender notifier.EmailSender
	if os.Getenv("PROVIDER_MODE") == "SIMULATED" {
		sender = notifier.NewSimulatedSender()
		log.Println("notification-service: using SIMULATED email provider")
	} else {
		smtpSender := email.NewSender(os.Getenv("SMTP_FROM"), os.Getenv("SMTP_PASSWORD"))
		if smtpSender == nil {
			log.Println("notification-service: SMTP not configured, email sending disabled")
		} else {
			sender = smtpSender
			log.Println("notification-service: using REAL SMTP email provider")
		}
	}

	consumer, err := messaging.NewConsumer(rabbitmqURL, store, sender, maxRetries, baseDelayMs)
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
