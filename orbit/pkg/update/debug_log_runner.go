package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// DebugLogReceiver is an OrbitConfigReceiver that applies server-driven
// changes to orbit's zerolog global level at runtime, so an admin can toggle
// debug logging for a team (via agent options) or a single host (via the
// /api/_version_/fleet/hosts/{id}/debug-logging endpoint) without restarting
// orbit.
//
// See docs/Contributing/architecture/orbit-debug-logging.md.
type DebugLogReceiver struct {
	// startedInDebug captures whether orbit was launched with --debug /
	// ORBIT_DEBUG=1. When true the receiver treats the startup flag as a
	// floor: the server can keep debug on but cannot turn it off. This lets
	// operators pin a host to debug mode from the host side (e.g. during a
	// local investigation) without the server quietly silencing them.
	startedInDebug bool
}

// NewDebugLogReceiver returns a receiver that reacts to OrbitConfig.DebugLogging.
// Pass true for startedInDebug if orbit was launched with --debug /
// ORBIT_DEBUG=1 so that the startup flag acts as a floor.
func NewDebugLogReceiver(startedInDebug bool) *DebugLogReceiver {
	return &DebugLogReceiver{startedInDebug: startedInDebug}
}

// Run sets zerolog's global level to match config.DebugLogging. A nil value
// means the server is not expressing an opinion (older server or feature
// not wired): the current level is preserved. The level is only mutated
// when it differs from the current value, making this call idempotent and
// safe to invoke on every config tick.
//
// When the receiver was constructed with startedInDebug=true, a server
// request to turn debug OFF is ignored — the startup flag takes precedence.
// A server request to turn debug ON is always honored.
func (r *DebugLogReceiver) Run(config *fleet.OrbitConfig) error {
	if config == nil || config.DebugLogging == nil {
		return nil
	}

	desired := zerolog.InfoLevel
	if *config.DebugLogging {
		desired = zerolog.DebugLevel
	}

	// Startup flag is a floor. Server cannot lower below DebugLevel when
	// orbit was launched in debug mode.
	if r.startedInDebug && desired == zerolog.InfoLevel {
		return nil
	}

	current := zerolog.GlobalLevel()
	if current == desired {
		return nil
	}

	zerolog.SetGlobalLevel(desired)
	log.Info().
		Str("from", current.String()).
		Str("to", desired.String()).
		Msg("orbit log level changed by server config")
	return nil
}
