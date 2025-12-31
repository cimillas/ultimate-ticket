package domain

import "time"

type HoldStatus string

const (
	HoldStatusActive    HoldStatus = "active"
	HoldStatusConfirmed HoldStatus = "confirmed"
	HoldStatusExpired   HoldStatus = "expired"
)

// Hold represents reserved inventory for a limited time.
type Hold struct {
	ID             string
	EventID        string
	ZoneID         string
	Quantity       int
	Status         HoldStatus
	ExpiresAt      time.Time
	IdempotencyKey string
	// IdempotencyHash can be stored when using hashed keys; not used in logic yet.
	IdempotencyHash string
	CreatedAt       time.Time
}
