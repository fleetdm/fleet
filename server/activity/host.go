package activity

import "context"

// Host represents minimal host info needed by the activity context for authorization.
// Populated via ACL from the legacy host service.
type Host struct {
	ID     uint
	TeamID *uint
}

// AuthzType returns the authorization type for hosts.
// This allows the activity bounded context to perform host-level authorization
// without importing the fleet.Host type.
func (h *Host) AuthzType() string {
	return "host"
}

// HostProvider is the interface for fetching host data.
// Implemented by the ACL adapter that calls the Fleet service.
type HostProvider interface {
	// GetHostLite returns minimal host information for authorization.
	// If the host doesn't exist, returns a NotFoundError.
	GetHostLite(ctx context.Context, hostID uint) (*Host, error)
}
