package osquery_utils

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// IsDiskEncryptionEnabledForHost checks if disk encryption is enabled for the
// host team or globally if the host is not assigned to a team.
func IsDiskEncryptionEnabledForHost(ctx context.Context, logger log.Logger, ds fleet.Datastore, host *fleet.Host) bool {
	// team
	if host.TeamID != nil {
		teamMDM, err := ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			level.Debug(logger).Log(
				"msg", "failed to get team MDM config for disk encryption check",
				"host_id", host.ID,
				"team_id", *host.TeamID,
				"err", err,
			)
			return false
		}
		if teamMDM == nil {
			return false
		}
		return teamMDM.EnableDiskEncryption
	}

	// global
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Debug(logger).Log(
			"msg", "failed to get app config for disk encryption check",
			"host_id", host.ID,
			"err", err,
		)
		return false
	}
	return appConfig.MDM.EnableDiskEncryption.Value
}
