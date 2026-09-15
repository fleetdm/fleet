package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// TestIssueCustomCommandManagementMode covers the pre-flight check that rejects AMAPI command types Google only
// supports on company-owned hosts. AMAPI accepts REBOOT on a personally-owned work profile and reports the operation
// as done with no error while the device ignores it, so Fleet has to refuse before issuing the command.
func TestIssueCustomCommandManagementMode(t *testing.T) {
	const hostID = uint(42)

	testCases := []struct {
		name                 string
		rawCommand           string
		isPersonalEnrollment bool
		hostMDMNotFound      bool
		wantErrContains      string
	}{
		{
			name:                 "reboot on personally-owned host is rejected",
			rawCommand:           `{"type":"REBOOT"}`,
			isPersonalEnrollment: true,
			wantErrContains:      "REBOOT is not supported for personally-owned Android hosts.",
		},
		{
			name:       "reboot on company-owned host is issued",
			rawCommand: `{"type":"REBOOT"}`,
		},
		{
			name:                 "lowercase reboot on personally-owned host is rejected",
			rawCommand:           `{"type":" reboot "}`,
			isPersonalEnrollment: true,
			wantErrContains:      "REBOOT is not supported for personally-owned Android hosts.",
		},
		{
			name:                 "relinquish ownership on personally-owned host is rejected",
			rawCommand:           `{"type":"RELINQUISH_OWNERSHIP"}`,
			isPersonalEnrollment: true,
			wantErrContains:      "RELINQUISH_OWNERSHIP is not supported for personally-owned Android hosts.",
		},
		{
			name:                 "start lost mode implied by params on personally-owned host is rejected",
			rawCommand:           `{"startLostModeParams":{"lostMessage":{"defaultMessage":"call me"}}}`,
			isPersonalEnrollment: true,
			wantErrContains:      "START_LOST_MODE is not supported for personally-owned Android hosts.",
		},
		{
			name:                 "stop lost mode implied by params on personally-owned host is rejected",
			rawCommand:           `{"stopLostModeParams":{}}`,
			isPersonalEnrollment: true,
			wantErrContains:      "STOP_LOST_MODE is not supported for personally-owned Android hosts.",
		},
		{
			name:                 "clear app data on personally-owned host is issued",
			rawCommand:           `{"type":"CLEAR_APP_DATA","clearAppsDataParams":{"packageNames":["com.example"]}}`,
			isPersonalEnrollment: true,
		},
		{
			name:                 "lock on personally-owned host is issued",
			rawCommand:           `{"type":"LOCK"}`,
			isPersonalEnrollment: true,
		},
		{
			name:            "reboot is issued when the host has no host_mdm row",
			rawCommand:      `{"type":"REBOOT"}`,
			hostMDMNotFound: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			androidAPIClient := android_mock.Client{}
			androidAPIClient.InitCommonMocks()
			androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, _ string,
				_ *androidmanagement.Command,
			) (*androidmanagement.Operation, error) {
				return &androidmanagement.Operation{Name: "enterprises/LC01/devices/dev1/operations/1"}, nil
			}

			fleetDS := InitCommonDSMocks()
			fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
				return &android.Enterprise{ID: 1, EnterpriseID: "LC01"}, nil
			}
			fleetDS.Store.HostLiteFunc = func(_ context.Context, id uint) (*fleet.Host, error) {
				return &fleet.Host{ID: id, UUID: "host-uuid", Platform: "android"}, nil
			}
			fleetDS.Store.AndroidHostLiteByHostUUIDFunc = func(_ context.Context, uuid string) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{
					Host:   &fleet.Host{ID: hostID, UUID: uuid, Platform: "android"},
					Device: &android.Device{HostID: hostID, DeviceID: "dev1"},
				}, nil
			}
			fleetDS.Store.GetHostMDMFunc = func(_ context.Context, _ uint) (*fleet.HostMDM, error) {
				if tt.hostMDMNotFound {
					return nil, &notFoundError{}
				}
				return &fleet.HostMDM{
					HostID:               hostID,
					Enrolled:             true,
					IsPersonalEnrollment: tt.isPersonalEnrollment,
				}, nil
			}
			fleetDS.Store.InsertMDMAndroidCommandFunc = func(_ context.Context, _ *android.MDMAndroidCommand) error {
				return nil
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore,
				noopNewActivity, config.AndroidAgentConfig{})
			require.NoError(t, err)

			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
			cmd, err := svc.IssueCustomCommand(ctx, hostID, []byte(tt.rawCommand))

			if tt.wantErrContains != "" {
				require.Error(t, err)
				var badRequestErr *fleet.BadRequestError
				require.ErrorAs(t, err, &badRequestErr)
				assert.Contains(t, badRequestErr.Message, tt.wantErrContains)
				assert.Nil(t, cmd)
				// The command must never reach AMAPI, so no row is written and no
				// "ran command" activity can be recorded for it.
				assert.False(t, androidAPIClient.EnterprisesDevicesIssueCommandFuncInvoked)
				assert.False(t, fleetDS.Store.InsertMDMAndroidCommandFuncInvoked)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.True(t, androidAPIClient.EnterprisesDevicesIssueCommandFuncInvoked)
			assert.True(t, fleetDS.Store.InsertMDMAndroidCommandFuncInvoked)
		})
	}
}

// TestCompanyOwnedOnlyCommandType covers the type normalization on its own, including the AMAPI behavior of inferring
// the command type from the params when type is omitted.
func TestCompanyOwnedOnlyCommandType(t *testing.T) {
	testCases := []struct {
		name string
		cmd  androidmanagement.Command
		want string
	}{
		{"explicit reboot", androidmanagement.Command{Type: "REBOOT"}, "REBOOT"},
		{"lowercase with spaces", androidmanagement.Command{Type: " reboot "}, "REBOOT"},
		{"relinquish ownership", androidmanagement.Command{Type: "RELINQUISH_OWNERSHIP"}, "RELINQUISH_OWNERSHIP"},
		{"explicit start lost mode", androidmanagement.Command{Type: "START_LOST_MODE"}, "START_LOST_MODE"},
		{
			"start lost mode implied by params",
			androidmanagement.Command{StartLostModeParams: &androidmanagement.StartLostModeParams{}},
			"START_LOST_MODE",
		},
		{
			"stop lost mode implied by params",
			androidmanagement.Command{StopLostModeParams: &androidmanagement.StopLostModeParams{}},
			"STOP_LOST_MODE",
		},
		{"lock is unrestricted", androidmanagement.Command{Type: "LOCK"}, ""},
		{"reset password is unrestricted", androidmanagement.Command{Type: "RESET_PASSWORD"}, ""},
		{"wipe is unrestricted", androidmanagement.Command{Type: "WIPE"}, ""},
		{
			"clear app data implied by params is unrestricted",
			androidmanagement.Command{ClearAppsDataParams: &androidmanagement.ClearAppsDataParams{}},
			"",
		},
		{"empty command", androidmanagement.Command{}, ""},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, companyOwnedOnlyCommandType(&tt.cmd))
		})
	}
}
