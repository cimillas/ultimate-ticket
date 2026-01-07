package domain

import "time"

type Session struct {
	TokenHash  string
	UserID     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastUsedAt time.Time
}
