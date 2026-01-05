package domain

import "time"

type EventStatus string

const (
	EventStatusActive    EventStatus = "active"
	EventStatusClosed    EventStatus = "closed"
	EventStatusCancelled EventStatus = "cancelled"
)

// Event represents a ticketed event (zone-based inventory).
type Event struct {
	ID          string
	Name        string
	StartsAt    time.Time
	Status      EventStatus
	CancelledAt *time.Time
	IsComplete  bool
}
