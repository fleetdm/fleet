// Package types defines internal interfaces for the activity bounded context.
package types

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// ListOptions extends api.ListOptions with internal fields used by the datastore.
type ListOptions struct {
	api.ListOptions

	// Internal fields - set programmatically by service, not from query params
	IncludeMetadata bool
	MatchingUserIDs []uint // User IDs matching MatchQuery (populated by service via ACL)
}

// GetPage returns the page number for pagination.
func (o *ListOptions) GetPage() uint { return o.Page }

// GetPerPage returns the number of items per page.
func (o *ListOptions) GetPerPage() uint { return o.PerPage }

// GetOrderKey returns the field to order by.
func (o *ListOptions) GetOrderKey() string { return o.OrderKey }

// IsDescending returns true if the order direction is descending.
func (o *ListOptions) IsDescending() bool { return o.OrderDirection == "desc" }

// GetCursorValue returns the cursor value for cursor-based pagination.
func (o *ListOptions) GetCursorValue() string { return o.After }

// WantsPaginationInfo returns true if pagination metadata should be included.
func (o *ListOptions) WantsPaginationInfo() bool { return o.IncludeMetadata }

// GetSecondaryOrderKey returns the secondary order key (not used for activities).
func (o *ListOptions) GetSecondaryOrderKey() string { return "" }

// IsSecondaryDescending returns true if the secondary order is descending (not used for activities).
func (o *ListOptions) IsSecondaryDescending() bool { return false }

// Datastore is the datastore interface for the activity bounded context.
type Datastore interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*api.Activity, *api.PaginationMetadata, error)
}
