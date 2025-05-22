//go:build !linux
// +build !linux

package luks

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Run is a placeholder method for non-Linux builds.
func (lr *LuksRunner) Run(oc *fleet.OrbitConfig) error {
	return nil
}

// GetLuksDump is a placeholder method for non-Linux builds.
func GetLuksDump(ctx context.Context, devicePath string) (*LuksDump, error) {
	return nil, nil
}
