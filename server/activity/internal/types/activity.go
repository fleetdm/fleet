// Package types defines internal interfaces for the activity bounded context.
package types

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// ListOptions extends api.ListOptions with internal fields used by the datastore.
// The service layer populates these before calling the datastore.
type ListOptions struct {
	api.ListOptions

	// Internal fields - set programmatically by service, not from query params
	IncludeMetadata bool
	MatchingUserIDs []uint // User IDs matching MatchQuery (populated by service via ACL)
}

// Datastore is the datastore interface for the activity bounded context.
type Datastore interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*api.Activity, *api.PaginationMetadata, error)
}
