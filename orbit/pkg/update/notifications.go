package update

import (
	"errors"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type runCmdFunc func() error

// renewEnrollmentProfileConfigFetcher is a kind of middleware that wraps an
// OrbitConfigFetcher and detects if the fleet server sent a notification to
// renew the enrollment profile. If so, it runs the command (as root) to
// bootstrap the renewal of the profile on the device (the user still needs to
// execute some manual steps to accept the new profile).
//
// It ensures only one renewal command is executed at any given time, and that
// it doesn't re-execute the command until a certain amount of time has passed.
type renewEnrollmentProfileConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher
	// Frequency is the minimum amount of time that must pass between two
	// executions of the profile renewal command.
	Frequency time.Duration

	// for tests, to be able to mock command execution. If nil, will use
	// runRenewEnrollmentProfile.
	runCmdFn runCmdFunc

	// ensures only one command runs at a time, protects access to lastRun
	cmdMu   sync.Mutex
	lastRun time.Time
}

func ApplyRenewEnrollmentProfileConfigFetcherMiddleware(fetcher OrbitConfigFetcher, frequency time.Duration) OrbitConfigFetcher {
	return &renewEnrollmentProfileConfigFetcher{Fetcher: fetcher, Frequency: frequency}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server set the renew enrollment profile flag to true, executes the command
// to renew the enrollment profile.
func (h *renewEnrollmentProfileConfigFetcher) GetConfig() (*fleet.OrbitConfig, error) {
	cfg, err := h.Fetcher.GetConfig()

	// TODO: download and use swiftDialog following the same patterns we
	// use for Nudge.
	//
	// updaterHasTarget := h.UpdateRunner.HasRunnerOptTarget("swiftDialog")
	// runnerHasLocalHash := h.UpdateRunner.HasLocalHash("swiftDialog")
	// if !updaterHasTarget || !runnerHasLocalHash {
	//         log.Info().Msg("refreshing the update runner config with swiftDialog targets and hashes")
	//         log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
	//         return cfg, h.setTargetsAndHashes()
	// }

	if err == nil && cfg.Notifications.RenewEnrollmentProfile {
		if h.cmdMu.TryLock() {
			defer h.cmdMu.Unlock()

			// Note that the macOS notification popup will be shown periodically
			// until the Fleet server gets notified that the device is now properly
			// enrolled (after the user's manual steps, and osquery reporting the
			// updated mdm enrollment).
			// See https://github.com/fleetdm/fleet/pull/9409#discussion_r1084382455
			if time.Since(h.lastRun) > h.Frequency {
				fn := h.runCmdFn
				if fn == nil {
					fn = runRenewEnrollmentProfile
				}
				if err := fn(); err != nil {
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

type execWinAPIFunc func(WindowsMDMEnrollmentArgs) error

type windowsMDMEnrollmentConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher
	// Frequency is the minimum amount of time that must pass between two
	// executions of the windows MDM enrollment attempt.
	Frequency time.Duration
	// HostUUID is the current host's UUID.
	HostUUID string

	// for tests, to be able to mock API commands. If nil, will use
	// RunWindowsMDMEnrollment and RunWindowsMDMUnenrollment respectively.
	execEnrollFn   execWinAPIFunc
	execUnenrollFn execWinAPIFunc

	// ensures only one command runs at a time, protects access to lastXxxRun and
	// isWindowsServer.
	mu              sync.Mutex
	lastEnrollRun   time.Time
	lastUnenrollRun time.Time
	isWindowsServer bool
}

func ApplyWindowsMDMEnrollmentFetcherMiddleware(
	fetcher OrbitConfigFetcher,
	frequency time.Duration,
	hostUUID string,
) OrbitConfigFetcher {
	return &windowsMDMEnrollmentConfigFetcher{
		Fetcher:   fetcher,
		Frequency: frequency,
		HostUUID:  hostUUID,
	}
}

var errIsWindowsServer = errors.New("device is a Windows Server")

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server set the "needs windows enrollment" flag to true, executes the command
// to enroll into Windows MDM (or not, if the device is a Windows Server).
func (w *windowsMDMEnrollmentConfigFetcher) GetConfig() (*fleet.OrbitConfig, error) {
	cfg, err := w.Fetcher.GetConfig()

	if err == nil {
		if cfg.Notifications.NeedsProgrammaticWindowsMDMEnrollment {
			w.attemptEnrollment(cfg.Notifications)
		} else if cfg.Notifications.NeedsProgrammaticWindowsMDMUnenrollment {
			w.attemptUnenrollment()
		}
	}
	return cfg, err
}

func (w *windowsMDMEnrollmentConfigFetcher) attemptEnrollment(notifs fleet.OrbitConfigNotifications) {
	if notifs.WindowsMDMDiscoveryEndpoint == "" {
		log.Info().Err(errors.New("discovery endpoint is missing")).Msg("skipping enrollment, discovery endpoint is empty")
		return
	}

	if w.mu.TryLock() {
		defer w.mu.Unlock()

		// do not enroll Windows Servers, and do not attempt enrollment if the last
		// run is not at least Frequency ago.
		if w.isWindowsServer {
			log.Debug().Msg("skipped calling RegisterDeviceWithManagement to enroll Windows device, device is a server")
			return
		}
		if time.Since(w.lastEnrollRun) <= w.Frequency {
			log.Debug().Msg("skipped calling RegisterDeviceWithManagement to enroll Windows device, last run was too recent")
			return
		}

		fn := w.execEnrollFn
		if fn == nil {
			fn = RunWindowsMDMEnrollment
		}
		args := WindowsMDMEnrollmentArgs{
			DiscoveryURL: notifs.WindowsMDMDiscoveryEndpoint,
			HostUUID:     w.HostUUID,
		}
		if err := fn(args); err != nil {
			if errors.Is(err, errIsWindowsServer) {
				w.isWindowsServer = true
				log.Info().Msg("device is a Windows Server, skipping enrollment")
			} else {
				log.Info().Err(err).Msg("calling RegisterDeviceWithManagement to enroll Windows device failed")
			}
			return
		}

		w.lastEnrollRun = time.Now()
		log.Info().Msg("successfully called RegisterDeviceWithManagement to enroll Windows device")
	}
}

func (w *windowsMDMEnrollmentConfigFetcher) attemptUnenrollment() {
	if w.mu.TryLock() {
		defer w.mu.Unlock()

		// do not unenroll Windows Servers, and do not attempt unenrollment if the
		// last run is not at least Frequency ago.
		if w.isWindowsServer {
			log.Debug().Msg("skipped calling UnregisterDeviceWithManagement to enroll Windows device, device is a server")
			return
		}
		if time.Since(w.lastUnenrollRun) <= w.Frequency {
			log.Debug().Msg("skipped calling UnregisterDeviceWithManagement to unenroll Windows device, last run was too recent")
			return
		}

		fn := w.execUnenrollFn
		if fn == nil {
			fn = RunWindowsMDMUnenrollment
		}
		args := WindowsMDMEnrollmentArgs{
			HostUUID: w.HostUUID,
		}
		if err := fn(args); err != nil {
			if errors.Is(err, errIsWindowsServer) {
				w.isWindowsServer = true
				log.Info().Msg("device is a Windows Server, skipping unenrollment")
			} else {
				log.Info().Err(err).Msg("calling UnregisterDeviceWithManagement to unenroll Windows device failed")
			}
			return
		}

		w.lastUnenrollRun = time.Now()
		log.Info().Msg("successfully called UnregisterDeviceWithManagement to unenroll Windows device")
	}
}
