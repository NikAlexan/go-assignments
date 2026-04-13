package usecase

import (
	"context"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

const maxAmount int64 = 100000 // $1000.00 in cents

type PaymentUseCase struct {
	repository PaymentRepository
}

func NewPaymentUseCase(repository PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repository: repository}
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
	return payment, nil
}

func (useCase *PaymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return useCase.repository.FindByOrderID(ctx, orderID)
}
