package usecase

import (
	"context"
	"order-service/internal/domain"
)

type OrderRepository interface {
	Save(ctx context.Context, order *domain.Order) error
	FindByID(ctx context.Context, id string) (*domain.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status string) error
	FindByStatus(ctx context.Context, status string) ([]*domain.Order, error)
}

type PaymentClient interface {
	// Authorize calls the Payment Service and returns the payment status ("Authorized" | "Declined")
	// or an error if the service is unreachable.
	Authorize(ctx context.Context, orderID string, amount int64) (string, error)
}
