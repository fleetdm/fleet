package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailedPolicyAutomationActivities(t *testing.T) {
	t.Run("webhook", func(t *testing.T) {
		act := ActivityTypeFailedWebhookPolicyAutomation{
			PolicyID:      7,
			HostIDList:    []uint{10, 20, 30},
			StatusCode:    500,
			ErrorResponse: "internal server error",
		}

		assert.Equal(t, "failed_webhook_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{10, 20, 30}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 7, got["policy_id"])
		assert.EqualValues(t, 500, got["status_code"])
		assert.Equal(t, "internal server error", got["error_response"])
	})

	t.Run("jira", func(t *testing.T) {
		act := ActivityTypeFailedJiraPolicyAutomation{
			PolicyID:      8,
			HostIDList:    []uint{11},
			ErrorResponse: "401 Unauthorized",
		}

		assert.Equal(t, "failed_jira_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{11}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 8, got["policy_id"])
		assert.Equal(t, "401 Unauthorized", got["error_response"])
		// no status_code field for jira/zendesk
		_, hasStatus := got["status_code"]
		assert.False(t, hasStatus)
	})

	t.Run("zendesk", func(t *testing.T) {
		act := ActivityTypeFailedZendeskPolicyAutomation{
			PolicyID:      9,
			HostIDList:    []uint{12, 13},
			ErrorResponse: "422: {\"error\":\"RecordInvalid\"}",
		}

		assert.Equal(t, "failed_zendesk_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{12, 13}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 9, got["policy_id"])
		assert.Equal(t, "422: {\"error\":\"RecordInvalid\"}", got["error_response"])
		// no status_code field for zendesk
		_, hasStatus := got["status_code"]
		assert.False(t, hasStatus)
	})

	t.Run("calendar", func(t *testing.T) {
		act := ActivityTypeFailedCalendarPolicyAutomation{
			PolicyID:      14,
			HostIDList:    []uint{42},
			StatusCode:    403,
			ErrorResponse: "Rate Limit Exceeded",
		}

		assert.Equal(t, "failed_calendar_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{42}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 14, got["policy_id"])
		assert.EqualValues(t, 403, got["status_code"])
		assert.Equal(t, "Rate Limit Exceeded", got["error_response"])
	})

	t.Run("conditional access", func(t *testing.T) {
		act := ActivityTypeFailedConditionalAccessPolicyAutomation{
			PolicyID:      15,
			HostIDList:    []uint{43},
			StatusCode:    500,
			ErrorResponse: "500: upstream error",
		}

		assert.Equal(t, "failed_conditional_access_policy_automation", act.ActivityName())
		assert.Equal(t, []uint{43}, act.HostIDs())
		assert.True(t, act.WasFromAutomation())

		b, err := json.Marshal(act)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		assert.EqualValues(t, 15, got["policy_id"])
		assert.EqualValues(t, 500, got["status_code"])
		assert.Equal(t, "500: upstream error", got["error_response"])
	})
}

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
