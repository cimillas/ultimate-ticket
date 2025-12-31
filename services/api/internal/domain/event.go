package domain

import "time"

// Event represents a ticketed event (zone-based inventory).
type Event struct {
	ID       string
	Name     string
	StartsAt time.Time
}
