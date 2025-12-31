package domain

import "errors"

var (
	ErrEventNotFound          = errors.New("event not found")
	ErrZoneNotFound           = errors.New("zone not found")
	ErrZoneAlreadyExists      = errors.New("zone already exists")
	ErrInsufficientCapacity   = errors.New("insufficient capacity")
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrInvalidCapacity        = errors.New("invalid capacity")
	ErrEventNameRequired      = errors.New("event name required")
	ErrZoneNameRequired       = errors.New("zone name required")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency conflict")
	ErrHoldNotFound           = errors.New("hold not found")
	ErrHoldExpired            = errors.New("hold expired")
	ErrHoldAlreadyConfirmed   = errors.New("hold already confirmed")
	ErrInvalidID              = errors.New("invalid id")
)
