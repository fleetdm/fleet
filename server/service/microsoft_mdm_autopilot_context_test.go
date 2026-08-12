package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Proves Fleet can read the Autopilot identifiers out of an enrollment request, so that when a real Autopilot device
// enrolls the only remaining unknown is whether Windows sends them. Verifying that needs physical hardware, which is
// slow to repeat, so everything testable is pinned here first.
func TestAutopilotEnrollmentContextItems(t *testing.T) {
	t.Parallel()

	const ztdGUID = "6f2b0d1a-6c3e-4f8a-9c1d-2b8a7e5f0c11"

	newRST := func(items ...fleet.ContextItem) *fleet.RequestSecurityToken {
		return &fleet.RequestSecurityToken{
			AdditionalContext: fleet.AdditionalContext{ContextItems: items},
		}
	}
	item := func(name, value string) fleet.ContextItem {
		return fleet.ContextItem{Name: name, Value: value}
	}

	t.Run("reads the Zero Touch Provisioning GUID when present", func(t *testing.T) {
		msg := newRST(
			item(syncml.ReqSecTokenContextItemDeviceID, "device-1"),
			item(syncml.ReqSecTokenContextItemZeroTouchProvisioning, ztdGUID),
		)
		got, err := GetContextItem(msg, syncml.ReqSecTokenContextItemZeroTouchProvisioning)
		require.NoError(t, err)
		assert.Equal(t, ztdGUID, got, "this is the value compared against windowsAutopilotDeviceIdentity.id")
	})

	t.Run("reads the offline registration correlator when present", func(t *testing.T) {
		msg := newRST(item(syncml.ReqSecTokenContextItemOfflineAutopilotCorrelator, ztdGUID))
		got, err := GetContextItem(msg, syncml.ReqSecTokenContextItemOfflineAutopilotCorrelator)
		require.NoError(t, err)
		assert.Equal(t, ztdGUID, got)
	})

	t.Run("absence is an error, not a panic, so a non-Autopilot enrollment is unaffected", func(t *testing.T) {
		msg := newRST(item(syncml.ReqSecTokenContextItemDeviceID, "device-1"))
		_, err := GetContextItem(msg, syncml.ReqSecTokenContextItemZeroTouchProvisioning)
		require.Error(t, err)
	})
}
