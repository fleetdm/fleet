// Package bootstrap provides dependency injection for the MDM settings status aggregator.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus"
	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus/api"
	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus/internal/service"
)

// New creates a new MDM settings status aggregator service with all dependencies wired up.
func New(
	recoveryLock mdmsettingsstatus.RecoveryLockStatusProvider,
	profiles mdmsettingsstatus.ProfilesStatusProvider,
	declarations mdmsettingsstatus.DeclarationsStatusProvider,
	fileVault mdmsettingsstatus.FileVaultStatusProvider,
) api.Service {
	return service.New(recoveryLock, profiles, declarations, fileVault)
}
