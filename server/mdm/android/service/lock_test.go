package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestLockAndroidHost(t *testing.T) {
	const hostID = uint(42)

	type lockSetup struct {
		svc        *Service
		apiClient  *android_mock.Client
		fleetDS    *AndroidMockDS
		issued     []*androidmanagement.Command
		storedCmds []*android.MDMAndroidCommand
		storedPINs []string
	}

	newSetup := func(t *testing.T, device *androidmanagement.Device, getErr error) *lockSetup {
		s := &lockSetup{apiClient: &android_mock.Client{}}
		s.apiClient.InitCommonMocks()
		s.apiClient.EnterprisesDevicesGetFunc = func(_ context.Context, deviceName string) (*androidmanagement.Device, error) {
			assert.Equal(t, "enterprises/LC01/devices/dev1", deviceName)
			return device, getErr
		}
		s.apiClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, _ string, cmd *androidmanagement.Command) (*androidmanagement.Operation, error) {
			s.issued = append(s.issued, cmd)
			return &androidmanagement.Operation{Name: "enterprises/LC01/devices/dev1/operations/1"}, nil
		}

		s.fleetDS = InitCommonDSMocks()
		s.fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
			return &android.Enterprise{ID: 1, EnterpriseID: "LC01"}, nil
		}
		s.fleetDS.Store.HostLiteFunc = func(_ context.Context, id uint) (*fleet.Host, error) {
			return &fleet.Host{ID: id, UUID: "host-uuid", Platform: "android"}, nil
		}
		s.fleetDS.Store.AndroidHostLiteByHostUUIDFunc = func(_ context.Context, uuid string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host:   &fleet.Host{ID: hostID, UUID: uuid, Platform: "android"},
				Device: &android.Device{HostID: hostID, DeviceID: "dev1"},
			}, nil
		}
		s.fleetDS.Store.LockHostViaAndroidMDMFunc = func(_ context.Context, _ *fleet.Host, cmd *android.MDMAndroidCommand, unlockPIN string) error {
			s.storedCmds = append(s.storedCmds, cmd)
			s.storedPINs = append(s.storedPINs, unlockPIN)
			return nil
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		svc, err := NewServiceWithClient(logger, s.fleetDS, s.apiClient, "test-private-key", &s.fleetDS.DataStore,
			noopNewActivity, config.AndroidAgentConfig{})
		require.NoError(t, err)
		s.svc = svc.(*Service)
		return s
	}
	adminCtx := func(t *testing.T) context.Context {
		return viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	}
	sixDigitPIN := regexp.MustCompile(`^[0-9]{6}$`)

	t.Run("issues RESET_PASSWORD with a policy-fitting password and LOCK_NOW", func(t *testing.T) {
		s := newSetup(t, &androidmanagement.Device{
			AppliedPasswordPolicies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "ALPHANUMERIC", PasswordMinimumLength: 10, PasswordScope: "SCOPE_PROFILE"},
			},
		}, nil)

		pin, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(pin), 10)
		require.Regexp(t, `[a-z]`, pin)
		require.Regexp(t, `[0-9]`, pin)

		require.Len(t, s.issued, 1)
		issued := s.issued[0]
		assert.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), issued.Type)
		assert.Equal(t, pin, issued.NewPassword)
		assert.Equal(t, []string{"LOCK_NOW"}, issued.ResetPasswordFlags)
		assert.Equal(t, longCommandDuration, issued.Duration)

		require.Len(t, s.storedCmds, 1)
		stored := s.storedCmds[0]
		assert.Equal(t, pin, s.storedPINs[0])
		assert.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), stored.CommandType)
		assert.Equal(t, string(android.MDMAndroidCommandStatusPending), stored.Status)
		assert.Equal(t, "enterprises/LC01/devices/dev1/operations/1", stored.OperationName)
		require.True(t, stored.RawCommand.Valid)
		assert.NotContains(t, stored.RawCommand.V, pin)
		var raw map[string]any
		require.NoError(t, json.Unmarshal([]byte(stored.RawCommand.V), &raw))
		assert.NotContains(t, raw, "newPassword")
		assert.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), raw["type"])
		assert.Equal(t, []any{"LOCK_NOW"}, raw["resetPasswordFlags"])
	})

	t.Run("falls back to a 6-digit PIN when the device can't be read", func(t *testing.T) {
		s := newSetup(t, nil, errors.New("amapi unavailable"))

		pin, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		require.NoError(t, err)
		require.Regexp(t, sixDigitPIN, pin)
		require.False(t, hasUnlockPasswordSequence([]byte(pin)), pin)
		require.Len(t, s.issued, 1)
		assert.Equal(t, pin, s.issued[0].NewPassword)
		assert.Equal(t, []string{pin}, s.storedPINs)
	})

	t.Run("no applied policies gives a 6-digit PIN, different each lock", func(t *testing.T) {
		s := newSetup(t, &androidmanagement.Device{}, nil)

		first, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		require.NoError(t, err)
		require.Regexp(t, sixDigitPIN, first)
		second, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		require.NoError(t, err)
		require.Regexp(t, sixDigitPIN, second)

		// 1 in ~10^6 chance of a false failure.
		assert.NotEqual(t, first, second)
		assert.Equal(t, []string{first, second}, s.storedPINs)
	})

	t.Run("a policy that needs over 32 characters fails before issuing", func(t *testing.T) {
		s := newSetup(t, &androidmanagement.Device{
			AppliedPasswordPolicies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "NUMERIC", PasswordMinimumLength: 40},
			},
		}, nil)

		_, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		var badRequest *fleet.BadRequestError
		require.ErrorAs(t, err, &badRequest)
		assert.Empty(t, s.issued)
		assert.False(t, s.fleetDS.Store.LockHostViaAndroidMDMFuncInvoked)
	})

	t.Run("an AMAPI rejection stores nothing", func(t *testing.T) {
		s := newSetup(t, &androidmanagement.Device{}, nil)
		s.apiClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, _ string, _ *androidmanagement.Command) (*androidmanagement.Operation, error) {
			return nil, errors.New("boom")
		}

		pin, err := s.svc.LockAndroidHost(adminCtx(t), hostID)
		require.Error(t, err)
		assert.Empty(t, pin)
		assert.False(t, s.fleetDS.Store.LockHostViaAndroidMDMFuncInvoked)
	})
}
