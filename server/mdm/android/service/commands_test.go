package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// TestIssueCustomCommandOwnership covers the pre-flight check that rejects AMAPI command types Google does not
// support on a personally-owned work profile. AMAPI accepts REBOOT there and reports the operation as done with no
// error while the device ignores it, so Fleet has to refuse before issuing the command.
func TestIssueCustomCommandOwnership(t *testing.T) {
	const hostID = uint(42)

	testCases := []struct {
		name                 string
		rawCommand           string
		isPersonalEnrollment bool
		hostMDMNotFound      bool
		hostMDMErr           bool
		wantErrContains      string
		// wantRejected distinguishes a refusal (a client error naming the command type) from a failure to
		// determine ownership, which must not be reported to the caller as an unsupported command.
		wantRejected bool
	}{
		{
			name:                 "reboot on personally-owned host is rejected",
			rawCommand:           `{"type":"REBOOT"}`,
			isPersonalEnrollment: true,
			wantErrContains:      "REBOOT is not supported for personally-owned Android hosts.",
			wantRejected:         true,
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
			wantRejected:         true,
		},
		{
			name:                 "relinquish ownership on personally-owned host is rejected",
			rawCommand:           `{"type":"RELINQUISH_OWNERSHIP"}`,
			isPersonalEnrollment: true,
			wantErrContains:      "RELINQUISH_OWNERSHIP is not supported for personally-owned Android hosts.",
			wantRejected:         true,
		},
		{
			name:                 "start lost mode implied by params on personally-owned host is rejected",
			rawCommand:           `{"startLostModeParams":{"lostMessage":{"defaultMessage":"call me"}}}`,
			isPersonalEnrollment: true,
			wantErrContains:      "START_LOST_MODE is not supported for personally-owned Android hosts.",
			wantRejected:         true,
		},
		{
			name:                 "stop lost mode implied by params on personally-owned host is rejected",
			rawCommand:           `{"stopLostModeParams":{}}`,
			isPersonalEnrollment: true,
			wantErrContains:      "STOP_LOST_MODE is not supported for personally-owned Android hosts.",
			wantRejected:         true,
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
			// Ownership cannot be determined without the row, so the command must not be issued.
			name:            "reboot is refused when the host has no host_mdm row",
			rawCommand:      `{"type":"REBOOT"}`,
			hostMDMNotFound: true,
			wantErrContains: "Can't run the MDM command because the host doesn't have MDM turned on.",
			wantRejected:    true,
		},
		{
			// Failing to read ownership must fail closed rather than fall through to issuing the command.
			name:            "reboot fails when the ownership lookup fails",
			rawCommand:      `{"type":"REBOOT"}`,
			hostMDMErr:      true,
			wantErrContains: "getting host_mdm for android custom command",
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
			fleetDS.Store.GetHostMDMFunc = func(ctx context.Context, _ uint) (*fleet.HostMDM, error) {
				assert.True(t, ctxdb.IsPrimaryRequired(ctx),
					"ownership must be read from the primary: a lagging replica reports a freshly enrolled BYOD host as company-owned")
				if tt.hostMDMErr {
					return nil, errors.New("host_mdm read failed")
				}
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

			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
			cmd, err := svc.IssueCustomCommand(ctx, hostID, []byte(tt.rawCommand))

			if tt.wantErrContains != "" {
				require.Error(t, err)
				assert.Nil(t, cmd)
				// Either way the command must never reach AMAPI, so no row is written and the caller
				// gets no command UUID to report as successfully run.
				assert.False(t, androidAPIClient.EnterprisesDevicesIssueCommandFuncInvoked)
				assert.False(t, fleetDS.Store.InsertMDMAndroidCommandFuncInvoked)

				var badRequestErr *fleet.BadRequestError
				if tt.wantRejected {
					require.ErrorAs(t, err, &badRequestErr)
					assert.Contains(t, badRequestErr.Message, tt.wantErrContains)
				} else {
					// A lookup failure is a server error, not a verdict on the command.
					assert.NotErrorAs(t, err, &badRequestErr)
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.True(t, androidAPIClient.EnterprisesDevicesIssueCommandFuncInvoked)
			assert.True(t, fleetDS.Store.InsertMDMAndroidCommandFuncInvoked)
			if companyOwnedOnlyCommandType(&androidmanagement.Command{Type: cmd.CommandType}) == "" {
				// Unrestricted types must not pay for an ownership lookup on every custom command.
				assert.False(t, fleetDS.Store.GetHostMDMFuncInvoked)
			}
		})
	}
}

// TestCompanyOwnedOnlyCommandType covers the type normalization on its own, including the AMAPI behavior of inferring
// the command type from the params when type is omitted.
func TestCompanyOwnedOnlyCommandType(t *testing.T) {
	testCases := []struct {
		name string
		cmd  androidmanagement.Command
		want android.MDMAndroidCommandType
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
