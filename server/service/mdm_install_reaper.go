package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ReapStuckMDMInstalls fails App Store and in-house app installs that have been activated for
// longer than olderThan without reaching a terminal state, releasing the activity queue of every
// host holding one.
//
// Fleet records these installs as successful only on verification, so one that never verifies holds
// the head of the host's queue for good and nothing behind it runs, scripts and package installs
// included. UnblockHostsUpcomingActivityQueue cannot rescue such a host because it looks for hosts
// where nothing is activated.
func ReapStuckMDMInstalls(ctx context.Context, ds fleet.Datastore, logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc, olderThan time.Duration, maxHosts int,
) error {
	// A non-positive age would make every activated install on the fleet reapable, which is the
	// opposite of what anyone setting it to zero intends. The cron leaves the job unregistered in
	// that case; this guard is here so no caller can reach the query with it.
	if olderThan <= 0 {
		return nil
	}

	// A partially successful run still returns the installs it did fail, and those have to be
	// recorded, so this error waits until they have been.
	reaped, err := ds.ReapStuckActivatedMDMInstalls(ctx, olderThan, maxHosts)
	var errs []error
	if err != nil {
		errs = append(errs, ctxerr.Wrap(ctx, err, "reap stuck activated MDM installs"))
	}

	for _, install := range reaped {
		// Warn rather than debug: this only fires on a host that has been unable to run any
		// activity for at least olderThan, which is never normal.
		logger.WarnContext(ctx, "failed a stuck app install to release the host's activity queue",
			"host_id", install.HostID, "command_uuid", install.CommandUUID)

		if err := recordReapedMDMInstall(ctx, ds, install, newActivityFn); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// recordReapedMDMInstall does what the verify result handler does when an install times out, and in
// the same order, so a reaped install ends up in the same state as one Fleet gave up on normally.
//
// The install is already failed and its queue row deleted by now, so it can no longer match the reap
// predicate and none of this is retried. One failure must therefore not cost two things. The setup
// experience step goes first, having the more lasting effect, and its error does not skip the
// activity: maybeUpdateSetupExperienceStatus reports `updated` alongside an error from the
// cancel-the-rest step that runs after its own commit.
func recordReapedMDMInstall(ctx context.Context, ds fleet.Datastore, install fleet.ReapedMDMInstall,
	newActivityFn fleet.NewActivityFunc,
) error {
	// A reaped install can be a setup experience step — App Store apps on any Apple platform,
	// in-house apps on iOS/iPadOS. Skipping the update would leave the step running for good
	// and, on macOS, never cancel the rest of the setup experience.
	var errs []error
	var act fleet.ActivityDetails
	switch {
	case install.AppStoreActivity != nil:
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, fleet.SetupExperienceVPPInstallResult{
			HostUUID:      install.HostUUID,
			CommandUUID:   install.CommandUUID,
			CommandStatus: fleet.MDMAppleStatusError,
		}, newActivityFn)
		if err != nil {
			errs = append(errs, ctxerr.Wrap(ctx, err, "updating setup experience status from reaped app store app install"))
		}
		install.AppStoreActivity.FromSetupExperience = updated
		act = install.AppStoreActivity

	case install.InHouseActivity != nil:
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, fleet.SetupExperienceVPPInstallResult{
			HostUUID:      install.HostUUID,
			CommandUUID:   install.CommandUUID,
			CommandStatus: fleet.MDMAppleStatusError,
		}, newActivityFn)
		if err != nil {
			errs = append(errs, ctxerr.Wrap(ctx, err, "updating setup experience status from reaped in-house app install"))
		}
		install.InHouseActivity.FromSetupExperience = updated
		act = install.InHouseActivity

	default:
		return nil
	}

	if err := newActivityFn(ctx, install.User, act); err != nil {
		errs = append(errs, ctxerr.Wrap(ctx, err, "creating activity for reaped app install"))
	}
	return errors.Join(errs...)
}
