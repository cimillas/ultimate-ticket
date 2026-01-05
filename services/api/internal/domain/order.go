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
