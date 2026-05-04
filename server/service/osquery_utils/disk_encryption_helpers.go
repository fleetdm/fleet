package osquery_utils

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// IsDiskEncryptionEnabledForHost checks if disk encryption is enabled for the
// host team or globally if the host is not assigned to a team.
func IsDiskEncryptionEnabledForHost(ctx context.Context, logger *slog.Logger, ds fleet.Datastore, host *fleet.Host) bool {
	// team
	if host.TeamID != nil {
		teamMDM, err := ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			logger.DebugContext(ctx, "failed to get team MDM config for disk encryption check",
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
		logger.DebugContext(ctx, "failed to get app config for disk encryption check",
			"host_id", host.ID,
			"err", err,
		)
		return false
	}
	return appConfig.MDM.EnableDiskEncryption.Value
}
