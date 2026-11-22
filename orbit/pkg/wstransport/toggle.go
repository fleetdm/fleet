// Package wstransport implements the fleetd side of the agent WebSocket
// notification transport (ADR-0011): orbit holds a persistent WebSocket to the
// Fleet server, receives tiny "check now" notifications and performs the usual
// HTTP distributed/read and distributed/write calls on behalf of osquery,
// acting as osquery's distributed plugin over the extension socket.
package wstransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// stateFileName persists the WebSocket transport toggle in orbit's root
// directory. Persistence is required (not just a nicety): orbit reads the
// toggle at startup before osquery is launched, and the early config fetch
// only runs when the server is reachable, so an offline start must still know
// which mode to run in.
const stateFileName = "websocket_transport.json"

// State is the server-directed WebSocket transport state, as of the last
// config fetch.
type State struct {
	Enabled bool `json:"enabled"`
	// OsqueryConfig is set when orbit serves osquery its config.
	OsqueryConfig bool `json:"osquery_config,omitempty"`
}

func stateFromConfig(cfg *fleet.OrbitConfig) State {
	ws := cfg.WebSocketTransport
	if ws == nil || !ws.Enabled {
		return State{}
	}
	return State{Enabled: true, OsqueryConfig: ws.OsqueryConfigPollInterval != nil}
}

// LoadState returns the persisted state. A missing or corrupt state file
// counts as disabled (the polling default).
func LoadState(rootDir string) State {
	data, err := os.ReadFile(filepath.Join(rootDir, stateFileName))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Info().Err(err).Msg("read websocket transport state; defaulting to disabled")
		}
		return State{}
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.Info().Err(err).Msg("parse websocket transport state; defaulting to disabled")
		return State{}
	}
	if !state.Enabled {
		return State{}
	}
	return state
}

// Enabled reports whether the WebSocket transport was enabled by the server
// directive as of the last config fetch.
func Enabled(rootDir string) bool {
	return LoadState(rootDir).Enabled
}

// ToggleReceiver watches the orbit config for the websocket_transport
// directive. When the server-directed state differs from the persisted one,
// it persists the new state and triggers an orbit restart: flipping osquery's
// --distributed_plugin or --config_plugin requires relaunching osquery, and
// orbit restarts handle that uniformly (same pattern as osquery flag updates).
type ToggleReceiver struct {
	rootDir             string
	triggerOrbitRestart func(reason string)
}

func NewToggleReceiver(rootDir string, triggerOrbitRestart func(reason string)) *ToggleReceiver {
	return &ToggleReceiver{
		rootDir:             rootDir,
		triggerOrbitRestart: triggerOrbitRestart,
	}
}

var _ fleet.OrbitConfigReceiver = (*ToggleReceiver)(nil)

func (r *ToggleReceiver) Run(cfg *fleet.OrbitConfig) error {
	want := stateFromConfig(cfg)
	if want == LoadState(r.rootDir) {
		return nil
	}

	state, err := json.Marshal(want)
	if err != nil {
		return fmt.Errorf("marshal websocket transport state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(r.rootDir, stateFileName), state, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write websocket transport state: %w", err)
	}

	log.Info().Bool("enabled", want.Enabled).Bool("osquery_config", want.OsqueryConfig).Msg("websocket transport toggled, restarting orbit")
	r.triggerOrbitRestart("websocket transport toggled")
	return nil
}
