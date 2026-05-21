package android_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/require"
)

// TestAndroidCommandStatusAcknowledgedStringMatches guards against silent divergence between
// the duplicated "acknowledged" status string in server/fleet and this package. The duplication
// exists because server/fleet imports server/mdm/android (so the reverse would be an import
// cycle); HostLockWipeStatus.IsLocked/IsWiped on server/fleet read mdm_android_commands.status
// and must compare against the same literal that this package writes. Lives in package
// android_test (external) to break the cycle the in-package test variant would create.
func TestAndroidCommandStatusAcknowledgedStringMatches(t *testing.T) {
	require.Equal(t, fleet.AndroidMDMCommandStatusAcknowledged, string(android.MDMAndroidCommandStatusAcknowledged),
		"android.MDMAndroidCommandStatusAcknowledged must equal fleet.AndroidMDMCommandStatusAcknowledged")
}
