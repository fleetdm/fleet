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

// Run sets the global zerolog level to match config.DebugLogging. Nil
// returns the value to the default, either info level or debug if the
// agent was started in debug mode.
func (r *DebugLogReceiver) Run(config *fleet.OrbitConfig) error {
	if config == nil {
		return nil
	}

	currentGlobalLevel := zerolog.GlobalLevel()

	desired := zerolog.InfoLevel
	if (config.DebugLogging != nil && *config.DebugLogging) || r.startedInDebug {
		desired = zerolog.DebugLevel
	}

	if currentGlobalLevel == desired {
		return nil
	}

	zerolog.SetGlobalLevel(desired)
	log.Info().
		Str("from", currentGlobalLevel.String()).
		Str("to", desired.String()).
		Msg("orbit log level changed by server config")
	return nil
}
