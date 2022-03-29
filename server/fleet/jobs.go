package fleet

import (
	"encoding/json"
	"time"
)

type JobState int

// The possible states for a job
//
//     ┌─────────┐
//     │         │
//     ▼         │
//  Queued───►Running───►Success
//               │
//               │
//               └──────►Failure
//
const (
	JobStateQueued JobState = iota + 1
	JobStateSuccess
	JobStateFailure
)

// Job describes an asynchronous job started via the queue package.
type Job struct {
	ID        uint             `json:"id" db:"id"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time       `json:"updated_at" db:"updated_at"`
	Name      string           `json:"name" db:"name"`
	Args      *json.RawMessage `json:"args" db:"args"`
	State     JobState         `json:"state" db:"state"`
	Retries   int              `json:"retries" db:"retries"`
	Error     string           `json:"error" db:"error"`
}
