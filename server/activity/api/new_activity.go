package api

import (
	"context"
)

// User represents user information for activity recording.
type User struct {
	ID      uint
	Name    string
	Email   string
	Deleted bool
}

// ActivityDetails defines the interface for activity detail types.
type ActivityDetails interface {
	ActivityName() string
}

// NewActivityService is for creating activities.
type NewActivityService interface {
	// NewActivity creates a new activity record and fires the webhook if configured.
	// user can be nil for automation-initiated activities.
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
}
