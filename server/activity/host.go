package activity

import "context"

// Host represents minimal host info needed by the activity context for authorization.
// The json tags must match the field names referenced by the OPA policy (see
// server/authz/policy.rego), otherwise authorization checks that depend on
// object.team_id (e.g. team-scoped admins reading host activities) will fail.
type Host struct {
	ID     uint  `json:"id"`
	TeamID *uint `json:"team_id"`
}

// AuthzType returns the authorization type for hosts.
func (h *Host) AuthzType() string {
	return "host"
}

// HostProvider is the interface for fetching host data.
type HostProvider interface {
	// GetHostLite returns minimal host information for authorization.
	// If the host doesn't exist, returns a NotFoundError.
	GetHostLite(ctx context.Context, hostID uint) (*Host, error)
}
