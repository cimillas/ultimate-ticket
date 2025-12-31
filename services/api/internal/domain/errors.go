package domain

import "errors"

var (
	ErrZoneNotFound           = errors.New("zone not found")
	ErrInsufficientCapacity   = errors.New("insufficient capacity")
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency conflict")
	ErrHoldNotFound           = errors.New("hold not found")
	ErrHoldExpired            = errors.New("hold expired")
	ErrHoldAlreadyConfirmed   = errors.New("hold already confirmed")
	ErrInvalidID              = errors.New("invalid id")
)
