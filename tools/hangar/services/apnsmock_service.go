package services

import "github.com/fleetdm/fleet/tools/hangar/internal/apnsmock"

// ApnsMockService reports on the per-server mock APNs servers and checks what
// one needs before it starts. Starting and stopping them goes through the
// process engine like the rest of a server's stack (see the frontend's
// orchestration).
type ApnsMockService struct{}

// ApnsMockStatus probes the mock listening on port.
func (s *ApnsMockService) ApnsMockStatus(port uint16) apnsmock.Status {
	return apnsmock.Probe(port)
}

// ApnsMockPreflight checks what a mock run from repo needs before it starts:
// its source, the server's Redis at redisPort, and a free port.
func (s *ApnsMockService) ApnsMockPreflight(repo string, redisPort, port uint16) apnsmock.Preflight {
	return apnsmock.Check(repo, redisPort, port)
}
