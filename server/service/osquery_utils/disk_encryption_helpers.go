package osquery_utils

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// IsDiskEncryptionEnabledForHost checks if disk encryption is enabled for the
// host's platform, in the host's team or globally if the host is not assigned
// to a team.
func IsDiskEncryptionEnabledForHost(ctx context.Context, logger *slog.Logger, ds fleet.Datastore, host *fleet.Host) bool {
	var cfg fleet.DiskEncryptionConfig

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
		cfg = teamMDM.DiskEncryptionConfig()
	} else {
		// global
		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			logger.DebugContext(ctx, "failed to get app config for disk encryption check",
				"host_id", host.ID,
				"err", err,
			)
			return false
		}
		cfg = appConfig.MDM.DiskEncryptionConfig()
	}

	// the FileVault flow treats the macOS pair as one unit until the
	// per-payload split ships
	switch host.FleetPlatform() {
	case "darwin":
		return cfg.MacOSEnabled || cfg.MacOSEscrowEnabled
	case "windows":
		return cfg.WindowsEnabled
	case "linux":
		return cfg.LinuxEscrowEnabled
	default:
		return false
	}
}
