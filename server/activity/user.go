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
// Implemented by the ACL adapter that calls the legacy service.
type UserProvider interface {
	ListUsers(ctx context.Context, ids []uint) ([]*User, error)
	// SearchUsers searches for users by name/email prefix and returns their IDs.
	// Used for the `query` parameter search functionality.
	SearchUsers(ctx context.Context, query string) ([]uint, error)
}
