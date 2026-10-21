// Package core defines Asteriade's platform-independent domain contracts.
package core

import "time"

type Limits struct {
	MaxDepth         int `json:"max_depth"`
	MaxConcurrent    int `json:"max_concurrent"`
	MaxRetries       int `json:"max_retries"`
	MaxTasks         int `json:"max_tasks"`
	ExecutionMinutes int `json:"execution_minutes"`
	ProjectMinutes   int `json:"project_minutes"`
}

type Project struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Goal              string     `json:"goal"`
	LifecycleStatus   string     `json:"lifecycle_status"`
	Status            string     `json:"status"`
	ExecutionMode     string     `json:"execution_mode"`
	Version           int64      `json:"version"`
	Limits            Limits     `json:"limits"`
	CreatedAt         time.Time  `json:"created_at"`
	CancelRequestedAt *time.Time `json:"cancel_requested_at,omitempty"`
}

type Task struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Objective string `json:"objective"`
	Status    string `json:"status"`
	Version   int64  `json:"version"`
}

type Execution struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
	Attempt   int    `json:"attempt"`
	Epoch     int64  `json:"epoch"`
	Version   int64  `json:"version"`
}

type Hold struct {
	ID         string     `json:"id"`
	Scope      string     `json:"scope"`
	ProjectID  string     `json:"project_id,omitempty"`
	Reason     string     `json:"reason"`
	Actor      string     `json:"actor"`
	Version    int64      `json:"version"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
}

type Event struct {
	ID               string    `json:"id"`
	Sequence         int64     `json:"sequence"`
	ProjectID        string    `json:"project_id,omitempty"`
	Type             string    `json:"type"`
	AggregateID      string    `json:"aggregate_id"`
	AggregateVersion int64     `json:"aggregate_version"`
	Payload          string    `json:"payload"`
	OccurredAt       time.Time `json:"occurred_at"`
}

type Operation struct {
	Kind                  string `json:"kind"`
	Destination           string `json:"destination"`
	Account               string `json:"account"`
	Target                string `json:"target"`
	PayloadHash           string `json:"payload_hash"`
	PolicyRevision        int64  `json:"policy_revision"`
	EstimatedCostMicroJPY int64  `json:"estimated_cost_micro_jpy"`
}

type Approval struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	OperationHash string    `json:"operation_hash"`
	Operation     Operation `json:"operation"`
	Scope         string    `json:"scope"`
	Status        string    `json:"status"`
	Version       int64     `json:"version"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type RunnerProfile struct {
	OS           string          `json:"os"`
	Status       string          `json:"status"`
	Fingerprint  string          `json:"fingerprint"`
	Capabilities map[string]bool `json:"capabilities"`
	Reason       string          `json:"reason"`
}
