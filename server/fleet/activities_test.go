package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailedPolicyAutomationActivities(t *testing.T) {
	t.Run("ticket (jira)", func(t *testing.T) {
		act := ActivityTypeFailedTicketPolicyAutomation{
			PolicyID:      8,
			HostIDList:    []uint{11},
			Type:          "jira",
			ErrorResponse: "401 Unauthorized",
		}

		assert.Equal(t, "failed_ticket_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{11}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 8, got["policy_id"])
		assert.Equal(t, "jira", got["type"])
		assert.Equal(t, "401 Unauthorized", got["error_response"])
		_, hasStatus := got["status_code"]
		assert.False(t, hasStatus)
	})

	t.Run("ticket (zendesk)", func(t *testing.T) {
		act := ActivityTypeFailedTicketPolicyAutomation{
			PolicyID:      9,
			HostIDList:    []uint{12, 13},
			Type:          "zendesk",
			ErrorResponse: "422: {\"error\":\"RecordInvalid\"}",
		}

		assert.Equal(t, "failed_ticket_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{12, 13}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 9, got["policy_id"])
		assert.Equal(t, "zendesk", got["type"])
		assert.Equal(t, "422: {\"error\":\"RecordInvalid\"}", got["error_response"])
		_, hasStatus := got["status_code"]
		assert.False(t, hasStatus)
	})
}

func TestSuccessPolicyAutomationActivities(t *testing.T) {
	// assertNoHostIDsOrPolicyName fails if the marshaled details leak the
	// host ID list or a policy name (both are intentionally omitted; hosts
	// live one-per-row in activity_host_past).
	assertNoHostIDsOrPolicyName := func(t *testing.T, got map[string]any) {
		t.Helper()
		_, hasHostIDs := got["host_ids"]
		assert.False(t, hasHostIDs)
		_, hasPolicyName := got["policy_name"]
		assert.False(t, hasPolicyName)
	}

	t.Run("ticket queued (jira)", func(t *testing.T) {
		act := ActivityTypeQueuedTicketPolicyAutomation{
			PolicyID:   8,
			HostIDList: []uint{11},
			Type:       "jira",
			TicketKey:  "ABC-123",
		}

		assert.Equal(t, "queued_ticket_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{11}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 8, got["policy_id"])
		assert.Equal(t, "jira", got["type"])
		assert.Equal(t, "ABC-123", got["ticket_key"])
		// ticket_id omitted for jira
		_, hasTicketID := got["ticket_id"]
		assert.False(t, hasTicketID)
		assertNoHostIDsOrPolicyName(t, got)
	})

	t.Run("ticket queued (zendesk)", func(t *testing.T) {
		act := ActivityTypeQueuedTicketPolicyAutomation{
			PolicyID:   9,
			HostIDList: []uint{12, 13},
			Type:       "zendesk",
			TicketID:   4567,
		}

		assert.Equal(t, "queued_ticket_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{12, 13}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 9, got["policy_id"])
		assert.Equal(t, "zendesk", got["type"])
		assert.EqualValues(t, 4567, got["ticket_id"])
		// ticket_key omitted for zendesk
		_, hasTicketKey := got["ticket_key"]
		assert.False(t, hasTicketKey)
		assertNoHostIDsOrPolicyName(t, got)
	})
}

// TestVPPInstallFailureEmptyCommandUUIDDoesNotActivateNext exercises the
// scenario where a VPP install is attempted during setup experience for a
// host that has other upcoming activities queued. If the VPP call fails
// before an MDM command is sent (e.g. no available licenses), the
// CommandUUID is empty. In that case the next upcoming activity must NOT
// be activated, because the current activity was never truly started —
// activating the next one would break the intended sequential ordering.
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
