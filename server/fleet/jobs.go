package fleet

import (
	"encoding/json"
	"time"
)

type JobState string

// The possible states for a job
//
//	Queued ───► Success
//	  │
//	  │
//	  └──────►Failure
const (
	JobStateQueued  JobState = "queued"
	JobStateSuccess JobState = "success"
	JobStateFailure JobState = "failure"
)

// Job describes an asynchronous job started via the worker package.
type Job struct {
	ID        uint             `json:"id" db:"id"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time       `json:"updated_at" db:"updated_at"`
	Name      string           `json:"name" db:"name"`
	Args      *json.RawMessage `json:"args" db:"args"`
	State     JobState         `json:"state" db:"state"`
	Retries   int              `json:"retries" db:"retries"`
	Error     string           `json:"error" db:"error"`
	NotBefore time.Time        `json:"not_before" db:"not_before"`
}
