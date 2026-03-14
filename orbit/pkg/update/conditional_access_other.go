//go:build !linux

package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
)

// ConditionalAccessRunner is a no-op placeholder on non-Linux platforms.
type ConditionalAccessRunner struct{}

// NewConditionalAccessRunner returns a no-op runner on non-Linux platforms.
func NewConditionalAccessRunner(
	metadataDir string,
	fleetURL string,
	enrollSecret string,
	hardwareUUID string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) *ConditionalAccessRunner {
	return &ConditionalAccessRunner{}
}

// Run is a no-op on non-Linux platforms.
func (r *ConditionalAccessRunner) Run(_ *fleet.OrbitConfig) error {
	return nil
}
