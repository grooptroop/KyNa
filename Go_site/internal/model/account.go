package model

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Account struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}
