// Package activity is the root package for the activity bounded context.
// It contains public types that need to be shared with the ACL layer.
package activity

import "context"

// User represents minimal user info needed by the activity context.
// Populated via ACL from the legacy user service.
type User struct {
	ID       uint
	Name     string
	Email    string
	Gravatar string
	APIOnly  bool
}

// UserProvider is the interface for fetching user data.
// Implemented by the ACL adapter that calls the Fleet service.
type UserProvider interface {
	// UsersByIDs returns users for the given IDs (for enriching activities).
	UsersByIDs(ctx context.Context, ids []uint) ([]*User, error)
	// FindUserIDs searches for users by name/email prefix and returns their IDs.
	// Used for the `query` parameter search functionality.
	FindUserIDs(ctx context.Context, query string) ([]uint, error)
}
