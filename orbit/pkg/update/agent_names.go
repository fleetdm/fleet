package update

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// agentNamesStateFile, in orbit's root directory, exists while the server has fleet.CapabilityFleetAgentNames. Linux
// resets the process name at every start, so orbit applies it from this file before it reaches the server.
const agentNamesStateFile = "agent_names_enabled"

// AgentNamesReceiver renames the agent once the server has fleet.CapabilityFleetAgentNames: "Fleet" in Windows
// Installed apps and "fleet" as the Linux process name. Service names, labels, units, packages and paths don't change, so
// scripts, MDM profiles and EDR rules that match them keep working.
type AgentNamesReceiver struct {
	serverCapabilities func() fleet.CapabilityMap
	apply              func() error
	stateFile          string // empty where the names outlive a restart

	done           bool
	failed         bool
	rememberFailed bool
}

func NewAgentNamesReceiver(rootDir string, serverCapabilities func() fleet.CapabilityMap) *AgentNamesReceiver {
	r := &AgentNamesReceiver{serverCapabilities: serverCapabilities, apply: applyAgentDisplayNames}
	if runtime.GOOS == "linux" {
		r.stateFile = filepath.Join(rootDir, agentNamesStateFile)
	}
	return r
}

// Run applies the names once per orbit run, retrying on later config fetches until it succeeds.
func (r *AgentNamesReceiver) Run(*fleet.OrbitConfig) error {
	enabled := r.serverCapabilities().Has(fleet.CapabilityFleetAgentNames)
	r.rememberForNextStart(enabled)
	if !enabled || r.done {
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

func (r *AgentNamesReceiver) rememberForNextStart(enabled bool) {
	if r.stateFile == "" {
		return
	}
	_, err := os.Stat(r.stateFile)
	switch exists := err == nil; {
	case enabled && !exists:
		err = os.WriteFile(r.stateFile, nil, constant.DefaultFileMode)
	case !enabled && exists:
		err = os.Remove(r.stateFile)
	default:
		return
	}
	if err == nil {
		return
	}
	// As with applying the names, only the first failure is logged at the default level.
	if r.rememberFailed {
		log.Debug().Err(err).Msg("remember the agent names for the next start")
	} else {
		log.Info().Err(err).Msg("remember the agent names for the next start")
		r.rememberFailed = true
	}
}

// ApplyRememberedAgentNames applies the Linux process name at start if the server had the capability at the last config
// fetch. Elsewhere the names outlive a restart.
func ApplyRememberedAgentNames(rootDir string) {
	if runtime.GOOS != "linux" {
		return
	}
	applyRememberedAgentNames(filepath.Join(rootDir, agentNamesStateFile), applyAgentDisplayNames)
}

func applyRememberedAgentNames(stateFile string, apply func() error) {
	if _, err := os.Stat(stateFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Info().Err(err).Msg("read the remembered agent names")
		}
		return
	}
	// AgentNamesReceiver tries again after the first config fetch.
	if err := apply(); err != nil {
		log.Debug().Err(err).Msg("show the agent as Fleet at start")
	}
}
