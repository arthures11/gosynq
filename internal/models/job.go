package models

import (
	"encoding/json"
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

type JobPriority string

const (
	PriorityLow    JobPriority = "low"
	PriorityNormal JobPriority = "normal"
	PriorityHigh   JobPriority = "high"
)

type Job struct {
	ID             string
	Queue          string
	Payload        json.RawMessage
	MaxRetries     int
	RunAt          time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Status         JobStatus
	Priority       JobPriority
	IdempotencyKey string
	LockedBy       string
	LockedAt       *time.Time
}

type JobAttempt struct {
	ID            string
	JobID         string
	AttemptNumber int
	StartedAt     time.Time
	CompletedAt   *time.Time
	Status        JobStatus
	ErrorMessage  string
}

type EnqueueJobRequest struct {
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	MaxRetries     int             `json:"max_retries"`
	RunAt          time.Time       `json:"run_at"`
	Priority       JobPriority     `json:"priority"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type JobEvent struct {
	Type      string      `json:"type"`
	JobID     string      `json:"job_id"`
	Queue     string      `json:"queue"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func (e *JobEvent) ToJSON() string {
	data, err := json.Marshal(e)
	if err != nil {
		return `{"error":"failed to marshal event"}`
	}
	return string(data)
}

func (j *Job) IsRetryable() bool {
	return j.MaxRetries > 0
}

func (j *Job) ShouldRetry(attempts int) bool {
	return attempts < j.MaxRetries
}
