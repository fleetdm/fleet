package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// DebugLogReceiver toggles orbit's zerolog level in response to
// OrbitConfig.DebugLogging.
type DebugLogReceiver struct {
	// startedInDebug acts as a floor: when orbit was launched with --debug
	// or ORBIT_DEBUG=1, the server can raise the level but cannot lower it.
	startedInDebug bool
}

func NewDebugLogReceiver(startedInDebug bool) *DebugLogReceiver {
	return &DebugLogReceiver{startedInDebug: startedInDebug}
}

// Run sets the global zerolog level to match config.DebugLogging. A nil
// value preserves the current level (server has no opinion).
func (r *DebugLogReceiver) Run(config *fleet.OrbitConfig) error {
	if config == nil || config.DebugLogging == nil {
		return nil
	}

	desired := zerolog.InfoLevel
	if *config.DebugLogging {
		desired = zerolog.DebugLevel
	}

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
