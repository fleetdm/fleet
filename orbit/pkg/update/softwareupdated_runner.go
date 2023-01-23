package update

import (
	"time"

	"github.com/rs/zerolog/log"
)

// SoftwareUpdatedRunner is a specialized runner to periodically kickstart the
// softwareupdated service on macOS, to work around a bug where the service
// hangs from time to time and prevents updates from being downloaded or update
// notifications from being shown.
//
// It is designed with Execute and Interrupt functions to be compatible with
// oklog/run.
type SoftwareUpdatedRunner struct {
	opt    SoftwareUpdatedOptions
	cancel chan struct{}
}

// SoftwareUpdatedOptions defines the options provided for the softwareupdated
// runner.
type SoftwareUpdatedOptions struct {
	// Interval is the interval at which to run the the kickstart softwareupdated
	// command.
	Interval time.Duration

	// runCmdFn can be set in tests to mock the command executed to kickstart
	// softwareupdated. If nil, defaults to runKickstartSoftwareUpdated.
	runCmdFn runCmdFunc
}

// NewSoftwareUpdatedRunner creates a new runner with the provided options. The
// runner must be started with Execute.
func NewSoftwareUpdatedRunner(opt SoftwareUpdatedOptions) *SoftwareUpdatedRunner {
	return &SoftwareUpdatedRunner{
		opt:    opt,
		cancel: make(chan struct{}),
	}
}

// Execute starts the loop to periodically run the kickstart command.
func (r *SoftwareUpdatedRunner) Execute() error {
	log.Debug().Msg("starting softwareupdated runner")

	// ensure it runs ~immediately the first time (e.g. on startup)
	firstInterval := 10 * time.Second
	if r.opt.Interval < firstInterval {
		firstInterval = r.opt.Interval
	}
	ticker := time.NewTicker(firstInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.cancel:
			return nil

		case <-ticker.C:
			log.Info().Msg("executing launchctl kickstart -k softwareupdated")
			fn := r.opt.runCmdFn
			if fn == nil {
				fn = runKickstartSoftwareUpdated
			}
			if err := fn(); err != nil {
				log.Info().Err(err).Msg("executing launchctl kickstart -k softwareupdated failed")
			}
			// run at the defined interval the next time around
			ticker.Reset(r.opt.Interval)
		}
	}
}

// Interrupt is the oklog/run interrupt method that stops the runner when
// called.
func (r *SoftwareUpdatedRunner) Interrupt(err error) {
	close(r.cancel)
	log.Debug().Err(err).Msg("interrupt for softwareupdated runner")
}
