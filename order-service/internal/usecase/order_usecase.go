package usecase

import (
	"context"
	"database/sql"
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

var ErrPaymentUnavailable = errors.New("payment service unavailable")

type OrderUseCase struct {
	repository    OrderRepository
	paymentClient PaymentClient
	cache         OrderCache
	cacheTTL      time.Duration
}

func NewOrderUseCase(repository OrderRepository, paymentClient PaymentClient, cache OrderCache, cacheTTL time.Duration) *OrderUseCase {
	return &OrderUseCase{repository: repository, paymentClient: paymentClient, cache: cache, cacheTTL: cacheTTL}
}

type CreateOrderInput struct {
	CustomerID     string
	ItemName       string
	Amount         int64
	IdempotencyKey string
}

func (useCase *OrderUseCase) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	// Idempotency check
	if input.IdempotencyKey != "" {
		existing, err := useCase.repository.FindByIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	order := &domain.Order{
		ID:             uuid.NewString(),
		CustomerID:     input.CustomerID,
		ItemName:       input.ItemName,
		Amount:         input.Amount,
		Status:         "Pending",
		CreatedAt:      time.Now().UTC(),
		IdempotencyKey: input.IdempotencyKey,
	}

	if err := useCase.repository.Save(ctx, order); err != nil {
		return nil, err
	}

	// Call Payment Service with a 2-second timeout
	paymentContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	paymentStatus, err := useCase.paymentClient.Authorize(paymentContext, order.ID, order.Amount)
	if err != nil {
		_ = useCase.repository.UpdateStatus(ctx, order.ID, "Failed")
		order.Status = "Failed"
		return order, ErrPaymentUnavailable
	}

	newStatus := "Failed"
	if paymentStatus == "Authorized" {
		newStatus = "Paid"
	}

	if err := useCase.repository.UpdateStatus(ctx, order.ID, newStatus); err != nil {
		return nil, err
	}
	order.Status = newStatus
	if useCase.cache != nil {
		_ = useCase.cache.Delete(ctx, order.ID)
	}
	return order, nil
}

func (useCase *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if useCase.cache != nil {
		if cached, err := useCase.cache.Get(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}
	order, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if useCase.cache != nil {
		_ = useCase.cache.Set(ctx, order, useCase.cacheTTL)
	}
	return order, nil
}

func (useCase *OrderUseCase) GetOrdersByStatus(ctx context.Context, status string) ([]*domain.Order, error) {
	return useCase.repository.FindByStatus(ctx, status)
}

func (useCase *OrderUseCase) GetPaymentStats(ctx context.Context) (*PaymentStats, error) {
	return useCase.paymentClient.GetPaymentStats(ctx)
}

func (useCase *OrderUseCase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !order.CanCancel() {
		return nil, domain.ErrCannotCancel
	}

	if err := useCase.repository.UpdateStatus(ctx, id, "Cancelled"); err != nil {
		return nil, err
	}
	order.Status = "Cancelled"
	if useCase.cache != nil {
		_ = useCase.cache.Delete(ctx, id)
	}
	return order, nil
}
