package activity

import "context"

// Host represents minimal host info needed by the activity context for authorization.
type Host struct {
	ID     uint
	TeamID *uint
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
