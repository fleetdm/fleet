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
type DebugLogReceiver struct{}

// NewDebugLogReceiver returns a receiver that reacts to OrbitConfig.DebugLogging.
func NewDebugLogReceiver() *DebugLogReceiver {
	return &DebugLogReceiver{}
}

// Run sets zerolog's global level to match config.DebugLogging. A nil value
// means the server is not expressing an opinion (older server or feature
// not wired): the current level is preserved. The level is only mutated
// when it differs from the current value, making this call idempotent and
// safe to invoke on every config tick.
func (r *DebugLogReceiver) Run(config *fleet.OrbitConfig) error {
	if config == nil || config.DebugLogging == nil {
		return nil
	}

	desired := zerolog.InfoLevel
	if *config.DebugLogging {
		desired = zerolog.DebugLevel
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
