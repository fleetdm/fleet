//go:build windows

package update

import "github.com/fleetdm/fleet/v4/orbit/pkg/constant"

// applyAgentDisplayNames renames fleetd's Installed apps entry. The installer keeps its "Fleet osquery" product name, so
// an installer repair brings that name back until fleetd's next start.
func applyAgentDisplayNames() error {
	return renameUninstallFleetdRegistryEntry(constant.AgentDisplayName)
}
