package update

import (
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// AgentNamesReceiver shows the agent as "Fleet" once the server has fleet.CapabilityFleetAgentNames: the Windows Installed
// apps name and the Linux process name. Service names, labels, units, packages and paths don't change, so scripts, MDM
// profiles and EDR rules that match them keep working.
type AgentNamesReceiver struct {
	serverCapabilities func() fleet.CapabilityMap
	apply              func() error

	mu     sync.Mutex
	done   bool
	failed bool
}

func NewAgentNamesReceiver(serverCapabilities func() fleet.CapabilityMap) *AgentNamesReceiver {
	return &AgentNamesReceiver{serverCapabilities: serverCapabilities, apply: applyAgentDisplayNames}
}

// Run applies the names once per orbit run, retrying on later config fetches until it succeeds.
func (r *AgentNamesReceiver) Run(*fleet.OrbitConfig) error {
	if !r.serverCapabilities().Has(fleet.CapabilityFleetAgentNames) {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return nil
	}
	if err := r.apply(); err != nil {
		// Only the first failure is an error, so a host where it can't succeed doesn't log one on every config fetch.
		if r.failed {
			log.Debug().Err(err).Msg("show the agent as Fleet")
		} else {
			log.Error().Err(err).Msg("show the agent as Fleet")
			r.failed = true
		}
		return nil
	}
	r.done = true
	return nil
}
