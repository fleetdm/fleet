//go:build linux

package update

import (
	"context"
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/condaccess"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConditionalAccessRunner reacts to the RunConditionalAccessEnrollment orbit
// notification by enrolling or renewing the SCEP client certificate used for
// Okta conditional access mTLS on Linux.
type ConditionalAccessRunner struct {
	isRunning    atomic.Bool
	metadataDir  string
	scepURL      string
	enrollSecret string
	hardwareUUID string
	rootCA       string
	insecure     bool
	logger       zerolog.Logger

	// enrollFn is the enrollment function, overridable in tests.
	enrollFn func(ctx context.Context, metadataDir, scepURL, challenge, uuid, rootCA string, insecure bool, logger zerolog.Logger) error
}

// NewConditionalAccessRunner creates a runner for Linux conditional access SCEP enrollment.
func NewConditionalAccessRunner(
	metadataDir string,
	fleetURL string,
	enrollSecret string,
	hardwareUUID string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) *ConditionalAccessRunner {
	r := &ConditionalAccessRunner{
		metadataDir:  metadataDir,
		scepURL:      fleetURL + "/api/fleet/conditional_access/scep",
		enrollSecret: enrollSecret,
		hardwareUUID: hardwareUUID,
		rootCA:       rootCA,
		insecure:     insecure,
		logger:       logger,
	}
	r.enrollFn = func(ctx context.Context, metadataDir, scepURL, challenge, uuid, rootCA string, insecure bool, logger zerolog.Logger) error {
		_, err := condaccess.Enroll(ctx, metadataDir, scepURL, challenge, uuid, rootCA, insecure, logger)
		return err
	}
	return r
}

// Run implements fleet.OrbitConfigReceiver. It launches a goroutine to perform
// SCEP enrollment when the server notification is set. Concurrent calls while
// a goroutine is already running are silently ignored (idempotent).
func (r *ConditionalAccessRunner) Run(cfg *fleet.OrbitConfig) error {
	if !cfg.Notifications.RunConditionalAccessEnrollment {
		return nil
	}
	if r.isRunning.Swap(true) {
		// already running, skip
		return nil
	}
	go func() {
		defer r.isRunning.Store(false)
		if err := r.enrollFn(
			context.Background(),
			r.metadataDir,
			r.scepURL,
			r.enrollSecret,
			r.hardwareUUID,
			r.rootCA,
			r.insecure,
			r.logger,
		); err != nil {
			log.Error().Err(err).Msg("conditional access SCEP enrollment failed")
		}
	}()
	return nil
}
