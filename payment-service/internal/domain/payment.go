package domain

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64  // Amount in cents
	Status        string // "Authorized" | "Declined"
}

type PaymentStats struct {
	TotalCount      int64
	AuthorizedCount int64
	DeclinedCount   int64
	TotalAmount     int64 // sum of all amounts in cents
}
