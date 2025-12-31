package domain

// Zone represents a sellable area for an event (no seat-level selection).
type Zone struct {
	ID       string
	EventID  string
	Name     string
	Capacity int
}
