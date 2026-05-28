package model

import "time"

type ProvisionStatus string

const (
	StatusPending ProvisionStatus = "pending"
	StatusReady   ProvisionStatus = "ready"
	StatusFailed  ProvisionStatus = "failed"
)

type UserProvision struct {
	ID         int64
	Username   string
	Domain     string
	Mode       string
	Status     ProvisionStatus
	ExternalIP *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AdminUserView struct {
	Username   string
	Email      string
	Role       Role
	Domain     *string
	Mode       *string
	Status     *ProvisionStatus
	ExternalIP *string
}
