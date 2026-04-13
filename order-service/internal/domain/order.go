package domain

import (
	"errors"
	"time"
)

var ErrCannotCancel = errors.New("only pending orders can be cancelled")

type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64  // Amount in cents
	Status         string // "Pending" | "Paid" | "Failed" | "Cancelled"
	CreatedAt      time.Time
	IdempotencyKey string
}

func (order *Order) CanCancel() bool {
	return order.Status == "Pending"
}
