package repository

import (
	"context"
	"database/sql"
	"payment-service/internal/domain"
)

type PostgresPaymentRepo struct {
	database *sql.DB
}

func NewPostgresPaymentRepo(database *sql.DB) *PostgresPaymentRepo {
	return &PostgresPaymentRepo{database: database}
}

func (repository *PostgresPaymentRepo) Save(ctx context.Context, payment *domain.Payment) error {
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO payments (id, order_id, transaction_id, amount, status) VALUES ($1, $2, $3, $4, $5)`,
		payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status,
	)
	return err
}

func (repository *PostgresPaymentRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	payment := &domain.Payment{}
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, order_id, transaction_id, amount, status FROM payments WHERE order_id = $1`,
		orderID,
	).Scan(&payment.ID, &payment.OrderID, &payment.TransactionID, &payment.Amount, &payment.Status)
	if err != nil {
		return nil, err
	}
	return payment, nil
}
