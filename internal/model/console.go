package model

import (
	"time"
)

// OpsDependencyStatus describes a single dependency probe.
type OpsDependencyStatus struct {
	Status    string `json:"status"` // ok | error | not_configured
	LatencyMs int64  `json:"latencyMs,omitempty"`
	Error     string `json:"error,omitempty"`
}

// OpsQueueStatus describes one RabbitMQ queue.
type OpsQueueStatus struct {
	Name      string `json:"name"`
	Depth     uint32 `json:"depth"`
	Consumers uint32 `json:"consumers"`
	Error     string `json:"error,omitempty"`
}

// OpsRabbitMQStatus describes the RabbitMQ connection and its queues.
type OpsRabbitMQStatus struct {
	Status string           `json:"status"` // ok | error | not_configured
	Queues []OpsQueueStatus `json:"queues"`
}

// OpsEventStats holds cumulative and trailing-hour counters for a family of
// events (ALE, auth, HTTP).
type OpsEventStats struct {
	Total    map[string]int64 `json:"total"`
	LastHour map[string]int64 `json:"lastHour"`
}

// OpsReconView is the last reconciliation run known to the API process.
type OpsReconView struct {
	At     time.Time `json:"at"`
	Source string    `json:"source"` // startup | manual
	Report *Report   `json:"report"`
}

// OpsSignupView is one recently registered user.
type OpsSignupView struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// OpsUsersView summarizes user activity for the observation checklist.
type OpsUsersView struct {
	Total         int             `json:"total"`
	RecentSignups []OpsSignupView `json:"recentSignups"`
}

// OpsServerInfo identifies the API instance serving the console.
type OpsServerInfo struct {
	StartedAt time.Time `json:"startedAt"`
	UptimeSec int64     `json:"uptimeSec"`
	GoVersion string    `json:"goVersion"`
	Env       string    `json:"env"`
	Version   string    `json:"version"`
}

// OpsStatus is the full payload of GET /v1/<code>/status.
type OpsStatus struct {
	Server         OpsServerInfo                  `json:"server"`
	Database       OpsDependencyStatus            `json:"database"`
	Redis          map[string]OpsDependencyStatus `json:"redis"`
	RabbitMQ       OpsRabbitMQStatus              `json:"rabbitmq"`
	HTTP           OpsEventStats                  `json:"http"`
	ALE            OpsEventStats                  `json:"ale"`
	Auth           OpsEventStats                  `json:"auth"`
	Reconciliation *OpsReconView                  `json:"reconciliation"`
	Users          OpsUsersView                   `json:"users"`
	RecentEvents   []OpsEvent                     `json:"recentEvents"`
}
