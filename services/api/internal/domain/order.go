package domain

import "time"

// Order represents a confirmed purchase derived from a hold.
type Order struct {
	ID             string
	HoldID         string
	IdempotencyKey string
	CreatedAt      time.Time
}
