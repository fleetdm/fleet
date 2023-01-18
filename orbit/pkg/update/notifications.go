package update

import (
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/rs/zerolog/log"
)

// RenewEnrollmentProfileConfigFetcher is a kind of middleware that wraps an
// OrbitConfigFetcher and detects if the fleet server sent a notification to
// renew the enrollment profile. If so, it runs the command (as root) to
// bootstrap the renewal of the profile on the device (the user still needs to
// execute some manual steps to accept the new profile).
//
// It ensures only one renewal command is executed at any given time, and that
// it doesn't re-execute the command until a certain amount of time has passed.
type RenewEnrollmentProfileConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher
	// Frequency is the minimum amount of time that must pass between two
	// executions of the profile renewal command.
	Frequency time.Duration

	// ensures only one command runs at a time, protects access to lastRun
	cmdMu   sync.Mutex
	lastRun time.Time
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server set the renew enrollment profile flag to true, executes the command
// to renew the enrollment profile.
func (h *RenewEnrollmentProfileConfigFetcher) GetConfig() (*service.OrbitConfig, error) {
	cfg, err := h.Fetcher.GetConfig()
	if err == nil && cfg.Notifications.RenewEnrollmentProfile {
		if h.cmdMu.TryLock() {
			defer h.cmdMu.Unlock()

			// TODO: what's a good delay after the last "renew enrollment profile"
			// command has run where we can assume Fleet has been notified that this
			// host is enrolled? AFAICS there's still a manual step that the user
			// must execute after the renew command has been executed - they will
			// receive a pop-up notification and then must follow some steps to
			// accept the new enrollment profile, captured in this figma:
			// https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-(dev-ready%2C-scratchpad)?node-id=10566%3A315998&t=jSV3l5p3I5gxX4EW-0
			//
			// So the host could be actually enrolled only much later, resulting in
			// this command being executed multiple times (presumably with the
			// associated macOS notification popup). Is that ok? Is that command
			// idempotent apart from the notification popup?
			if time.Since(h.lastRun) > h.Frequency {
				if err := runRenewEnrollmentProfile(); err != nil {
					log.Info().Err(err).Msg("calling /usr/bin/profiles to renew enrollment profile failed")
				} else {
					h.lastRun = time.Now()
					log.Info().Msg("successfully called /usr/bin/profiles to renew enrollment profile")
				}
			} else {
				log.Debug().Msg("skipped calling /usr/bin/profiles to renew enrollment profile, last run was too recent")
			}
		}
	}
	return cfg, err
}
