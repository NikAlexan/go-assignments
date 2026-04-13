package repository

import (
	"context"
	"database/sql"
	"order-service/internal/domain"
)

type PostgresOrderRepo struct {
	database *sql.DB
}

func NewPostgresOrderRepo(database *sql.DB) *PostgresOrderRepo {
	return &PostgresOrderRepo{database: database}
}

func (repository *PostgresOrderRepo) Save(ctx context.Context, order *domain.Order) error {
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO orders (id, customer_id, item_name, amount, status, created_at, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))`,
		order.ID, order.CustomerID, order.ItemName, order.Amount, order.Status, order.CreatedAt, order.IdempotencyKey,
	)
	return err
}

func (repository *PostgresOrderRepo) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	order := &domain.Order{}
	var idempotencyKey sql.NullString
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key
		 FROM orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.CustomerID, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt, &idempotencyKey)
	if err != nil {
		return nil, err
	}
	order.IdempotencyKey = idempotencyKey.String
	return order, nil
}

func (repository *PostgresOrderRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	order := &domain.Order{}
	var idempotencyKey sql.NullString
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key
		 FROM orders WHERE idempotency_key = $1`, key,
	).Scan(&order.ID, &order.CustomerID, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt, &idempotencyKey)
	if err != nil {
		return nil, err
	}
	order.IdempotencyKey = idempotencyKey.String
	return order, nil
}

func (repository *PostgresOrderRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`, status, id,
	)
	return err
}

func (repository *PostgresOrderRepo) FindByStatus(ctx context.Context, status string) ([]*domain.Order, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key
		 FROM orders WHERE status = $1`, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		var idempotencyKey sql.NullString
		if err := rows.Scan(&order.ID, &order.CustomerID, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt, &idempotencyKey); err != nil {
			return nil, err
		}
		order.IdempotencyKey = idempotencyKey.String
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
