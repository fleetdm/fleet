package services

import "github.com/fleetdm/fleet/tools/hangar/internal/apnsmock"

// ApnsMockService reports on the per-server mock APNs servers. Starting and
// stopping them goes through the process engine like the rest of a server's
// stack (see the frontend's orchestration), so this only probes.
type ApnsMockService struct{}

// ApnsMockStatus probes the mock listening on port.
func (s *ApnsMockService) ApnsMockStatus(port uint16) apnsmock.Status {
	return apnsmock.Probe(port)
}
