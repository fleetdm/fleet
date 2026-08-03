package fleet

import "context"

// GoogleWorkspaceGroup is a group pulled from a Google Workspace directory along
// with the external IDs (Google user IDs) of its members. It is an intermediate
// representation used by the sync engine to populate the scim_groups and
// scim_user_group tables.
type GoogleWorkspaceGroup struct {
	// ExternalID is the Google group ID (maps to scim_groups.external_id).
	ExternalID string
	// DisplayName is the group's display name (maps to scim_groups.display_name).
	DisplayName string
	// MemberExternalIDs holds the Google user IDs of the group's direct members.
	MemberExternalIDs []string
}

// GoogleWorkspaceDirectory pulls users and groups from a Google Workspace
// directory via the Admin SDK Directory API so they can be synced into Fleet's
// IdP host vitals (the scim_* tables). Unlike SCIM — which the IdP pushes to Fleet
// over HTTP — this is a pull performed by Fleet on a schedule.
//
// The concrete implementation lives in ee/server/googleworkspace. Mapping from
// Google's data model to Fleet's ScimUser happens behind this interface so this
// package does not depend on the Google API client.
type GoogleWorkspaceDirectory interface {
	// ListUsers returns every user in the configured domain mapped to a ScimUser.
	// ExternalID is set to the Google user ID; the ScimUser's group membership is
	// not populated here — it is resolved from ListGroups by the sync engine.
	ListUsers(ctx context.Context) ([]*ScimUser, error)
	// ListGroups returns every group in the configured domain along with the
	// external IDs (Google user IDs) of each group's members.
	ListGroups(ctx context.Context) ([]*GoogleWorkspaceGroup, error)
}
