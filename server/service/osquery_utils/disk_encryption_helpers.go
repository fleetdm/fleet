package osquery_utils

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// IsDiskEncryptionEscrowEnabledForHost reports whether Fleet should store a
// recovery key for this host, based on its platform's escrow-relevant setting
// in the host's team, or globally if the host is not assigned to a team.
func IsDiskEncryptionEscrowEnabledForHost(ctx context.Context, logger *slog.Logger, ds fleet.Datastore, host *fleet.Host) bool {
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

	switch host.FleetPlatform() {
	case "darwin":
		// enforcement alone escrows nothing: the profile omits the escrow
		// payloads, so no key is ever produced for Fleet to store
		return cfg.MacOSEscrowEnabled
	case "windows":
		// BitLocker escrow is not separately settable; enforcement implies it
		return cfg.WindowsEnabled
	case "linux":
		return cfg.LinuxEscrowEnabled
	default:
		return false
	}
}
