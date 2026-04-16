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

func (repository *PostgresPaymentRepo) GetStats(ctx context.Context) (*domain.PaymentStats, error) {
	stats := &domain.PaymentStats{}
	err := repository.database.QueryRowContext(ctx, `
		SELECT
			COUNT(*)                                      AS total_count,
			COUNT(*) FILTER (WHERE status = 'Authorized') AS authorized_count,
			COUNT(*) FILTER (WHERE status = 'Declined')   AS declined_count,
			COALESCE(SUM(amount), 0)                      AS total_amount
		FROM payments
	`).Scan(&stats.TotalCount, &stats.AuthorizedCount, &stats.DeclinedCount, &stats.TotalAmount)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
