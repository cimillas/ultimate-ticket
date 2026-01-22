package domain

import "time"

type OrderStatus string

const (
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusRefunded  OrderStatus = "refunded"
)

// Order represents a confirmed purchase derived from a hold.
type Order struct {
	ID             string
	HoldID         string
	IdempotencyKey string
	Status         OrderStatus
	CreatedAt      time.Time
}

// OrderSummary represents a user-scoped view of an order with hold context.
type OrderSummary struct {
	ID        string
	HoldID    string
	EventID   string
	ZoneID    string
	Quantity  int
	Status    OrderStatus
	CreatedAt time.Time
}
