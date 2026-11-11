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

	done   bool
	failed bool
}

// agentNamesMu serializes the receiver's renames with undoAgentNames, after which this orbit run doesn't rename again.
var (
	agentNamesMu     sync.Mutex
	agentNamesUndone bool
)

func NewAgentNamesReceiver(serverCapabilities func() fleet.CapabilityMap) *AgentNamesReceiver {
	return &AgentNamesReceiver{serverCapabilities: serverCapabilities, apply: applyAgentDisplayNames}
}

// Run applies the names once per orbit run, retrying on later config fetches until it succeeds.
func (r *AgentNamesReceiver) Run(*fleet.OrbitConfig) error {
	if !r.serverCapabilities().Has(fleet.CapabilityFleetAgentNames) {
		return nil
	}
	agentNamesMu.Lock()
	defer agentNamesMu.Unlock()
	if r.done || agentNamesUndone {
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

// undoAgentNames runs before a downgrade and restores the installer's name: an older fleetd finds its Windows Installed apps
// entry only by that name, and without it doesn't restart into its next update. A version that renames does so again.
func undoAgentNames(restore func() error) {
	agentNamesMu.Lock()
	defer agentNamesMu.Unlock()
	agentNamesUndone = true
	if err := restore(); err != nil {
		log.Error().Err(err).Msg("restore the agent's installer name before the downgrade")
	}
}
