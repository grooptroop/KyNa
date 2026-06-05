package model

import "time"

type MachineStatus string

const (
	MachineStatusPending MachineStatus = "pending"
	MachineStatusReady   MachineStatus = "ready"
	MachineStatusFailed  MachineStatus = "failed"
)

type UserMachine struct {
	ID              int64
	Username        string
	Name            string
	Mode            string
	ServiceKind     string
	Status          MachineStatus
	ExternalIP      *string
	ClusterIP       *string
	IngressHost     *string
	ResourcesPreset string
	AccessScope     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
