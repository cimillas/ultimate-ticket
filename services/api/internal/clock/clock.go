package clock

import "time"

// Clock allows injecting time in domain/services.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

// NewSystem returns a clock backed by time.Now.
func NewSystem() Clock {
	return systemClock{}
}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type fixedClock struct {
	now time.Time
}

// NewFixed returns a clock that always returns the same instant (useful for tests).
func NewFixed(t time.Time) Clock {
	return fixedClock{now: t.UTC()}
}

func (f fixedClock) Now() time.Time {
	return f.now
}
