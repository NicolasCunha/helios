// Package models defines domain models for Helios.
package models

import "time"

// HealthCheckLog represents a health check log entry for a container.
type HealthCheckLog struct {
	ID                  int64     `json:"id"`
	ContainerID         string    `json:"container_id"`
	ContainerName       string    `json:"container_name"`
	Status              string    `json:"status"` // healthy, unhealthy, resource_critical
	ResourceCPU         float64   `json:"resource_cpu"`
	ResourceMemory      uint64    `json:"resource_memory"`
	ResourceMemoryLimit uint64    `json:"resource_memory_limit"`
	ResourceNetworkRx   uint64    `json:"resource_network_rx"`
	ResourceNetworkTx   uint64    `json:"resource_network_tx"`
	ErrorMessage        string    `json:"error_message,omitempty"`
	CheckedAt           time.Time `json:"checked_at"`
}

// ActionLog represents an action performed on a Docker resource.
type ActionLog struct {
	ID           int64     `json:"id"`
	ActionType   string    `json:"action_type"`   // start, stop, restart, remove, create, etc.
	ResourceType string    `json:"resource_type"` // container, image, volume, network
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ExecutedAt   time.Time `json:"executed_at"`
}

// EventLog represents a system event log entry.
type EventLog struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"` // system, docker, api, health_check
	Level     string    `json:"level"`      // info, warning, error
	Message   string    `json:"message"`
	Metadata  string    `json:"metadata,omitempty"` // JSON-encoded additional data
	CreatedAt time.Time `json:"created_at"`
}
