package usecase

import (
	"context"
	"log"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

const maxAmount int64 = 100000 // $1000.00 in cents

type PaymentUseCase struct {
	repository    PaymentRepository
	publisher     EventPublisher
	defaultEmail  string
}

func NewPaymentUseCase(repository PaymentRepository, publisher EventPublisher, defaultEmail string) *PaymentUseCase {
	if defaultEmail == "" {
		defaultEmail = "user@example.com"
	}
	return &PaymentUseCase{repository: repository, publisher: publisher, defaultEmail: defaultEmail}
}

func (useCase *PaymentUseCase) Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	payment := &domain.Payment{
		ID:      uuid.NewString(),
		OrderID: orderID,
		Amount:  amount,
	}

	if amount > maxAmount {
		payment.Status = "Declined"
		payment.TransactionID = ""
	} else {
		payment.Status = "Authorized"
		payment.TransactionID = uuid.NewString()
	}

	if err := useCase.repository.Save(ctx, payment); err != nil {
		return nil, err
	}

	event := PaymentEvent{
		EventID:       uuid.NewString(),
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: useCase.defaultEmail,
		Status:        payment.Status,
	}
	if err := useCase.publisher.Publish(ctx, event); err != nil {
		log.Printf("publish payment event: %v", err)
	}

	return payment, nil
}

func (useCase *PaymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return useCase.repository.FindByOrderID(ctx, orderID)
}

func (useCase *PaymentUseCase) GetStats(ctx context.Context) (*domain.PaymentStats, error) {
	return useCase.repository.GetStats(ctx)
}
