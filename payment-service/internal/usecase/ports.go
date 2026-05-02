package usecase

import (
	"context"
	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
	FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	GetStats(ctx context.Context) (*domain.PaymentStats, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event PaymentEvent) error
}

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}
