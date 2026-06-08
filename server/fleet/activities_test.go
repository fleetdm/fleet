package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVPPInstallFailureEmptyCommandUUIDDoesNotActivateNext exercises the
// scenario where a VPP install is attempted during setup experience for a
// host that has other upcoming activities queued. If the VPP call fails
// before an MDM command is sent (e.g. no available licenses), the
// CommandUUID is empty. In that case the next upcoming activity must NOT
// be activated, because the current activity was never truly started —
// activating the next one would break the intended sequential ordering.
//
// See commit 159194acc9d92843bb2de933309f159c84a501aa for the fix.
func TestVPPInstallFailureEmptyCommandUUIDDoesNotActivateNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		activity       ActivityInstalledAppStoreApp
		expectActivate bool
	}{
		{
			name: "failed VPP install with empty command UUID must not activate next upcoming activity",
			activity: ActivityInstalledAppStoreApp{
				HostID:              42,
				HostDisplayName:     "ios-host",
				SoftwareTitle:       "Licensed App",
				AppStoreID:          "99999",
				CommandUUID:         "", // no MDM command was sent
				Status:              string(SoftwareInstallFailed),
				FromSetupExperience: true,
			},
			expectActivate: false,
		},
		{
			name: "failed VPP install with command UUID activates next upcoming activity",
			activity: ActivityInstalledAppStoreApp{
				HostID:              42,
				HostDisplayName:     "ios-host",
				SoftwareTitle:       "Licensed App",
				AppStoreID:          "99999",
				CommandUUID:         "cmd-uuid-abc",
				Status:              string(SoftwareInstallFailed),
				FromSetupExperience: true,
			},
			expectActivate: true,
		},
		{
			name: "successful VPP install must not activate next (handled by install verification)",
			activity: ActivityInstalledAppStoreApp{
				HostID:              42,
				HostDisplayName:     "ios-host",
				SoftwareTitle:       "Licensed App",
				AppStoreID:          "99999",
				CommandUUID:         "cmd-uuid-abc",
				Status:              string(SoftwareInstalled),
				FromSetupExperience: true,
			},
			expectActivate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectActivate, tt.activity.MustActivateNextUpcomingActivity(),
				"MustActivateNextUpcomingActivity() = %v, want %v",
				tt.activity.MustActivateNextUpcomingActivity(), tt.expectActivate)

			if tt.expectActivate {
				hostID, cmdUUID := tt.activity.ActivateNextUpcomingActivityArgs()
				assert.Equal(t, tt.activity.HostID, hostID)
				assert.Equal(t, tt.activity.CommandUUID, cmdUUID)
			}
		})
	}
}
