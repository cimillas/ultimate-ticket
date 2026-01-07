package domain

import "time"

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type User struct {
	ID        string
	Username  string
	Email     string
	Role      UserRole
	CreatedAt time.Time
}
