package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

// testBitLockerPINPrivateKey is a 32-byte key so AES-256 encryption of the PIN works in tests.
const testBitLockerPINPrivateKey = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456"

// newBitLockerPINTestService builds a service whose host is a Windows host that needs a PIN and whose fleetd can set
// one. Individual tests override the mocks they care about.
func newBitLockerPINTestService(
	t *testing.T, host *fleet.Host,
) (fleet.Service, *mock.Store, context.Context, *TestServerOpts) {
	t.Helper()

	ds := new(mock.Store)
	cfg := config.TestConfig()
	cfg.Server.PrivateKey = testBitLockerPINPrivateKey
	opts := &TestServerOpts{SkipCreateTestUsers: true, License: &fleet.LicenseInfo{Tier: fleet.TierPremium}}
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)

	ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
		return &fleet.MDMWindowsHostConfigState{FleetdBitLockerPINCapable: true}, nil
	}
	ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, h *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
		return &fleet.HostMDMDiskEncryption{ActionRequired: new(fleet.ActionRequiredCreatePIN)}, nil
	}

	return svc, ds, test.HostContext(ctx, host), opts
}

func windowsPINHost() *fleet.Host {
	return &fleet.Host{ID: 42, UUID: "host-uuid", Platform: "windows", Hostname: "MU-TH-UR"}
}

func TestSubmitBitLockerPIN(t *testing.T) {
	t.Run("queues the PIN encrypted", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		var queued string
		ds.QueueBitLockerPINRequestFunc = func(ctx context.Context, h *fleet.Host, encryptedPIN string) error {
			require.Equal(t, host.ID, h.ID)
			queued = encryptedPIN
			return nil
		}

		require.NoError(t, svc.SubmitBitLockerPIN(ctx, host, "123456"))
		require.True(t, ds.QueueBitLockerPINRequestFuncInvoked)

		// The stored value must not be the PIN itself, and must decrypt back to it.
		require.NotEmpty(t, queued)
		require.NotEqual(t, "123456", queued)
		decrypted, err := mdm.DecodeAndDecrypt(queued, testBitLockerPINPrivateKey)
		require.NoError(t, err)
		require.Equal(t, "123456", decrypted)
	})

	t.Run("rejects an invalid PIN before touching the datastore", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		err := svc.SubmitBitLockerPIN(ctx, host, "12ab")
		require.Error(t, err)
		require.Contains(t, err.Error(), microsoft_mdm.BitLockerPINLengthMessage)
		require.False(t, ds.QueueBitLockerPINRequestFuncInvoked)
	})

	t.Run("rejects a host that does not need a PIN", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)
		// The page may be showing a stale view: another session set the PIN, or the fleet stopped requiring one.
		ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, h *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
			return &fleet.HostMDMDiskEncryption{}, nil
		}

		err := svc.SubmitBitLockerPIN(ctx, host, "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "doesn't need a BitLocker PIN")
		require.False(t, ds.QueueBitLockerPINRequestFuncInvoked)
	})

	t.Run("rejects a host whose fleetd cannot apply a PIN", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{FleetdBitLockerPINCapable: false}, nil
		}

		err := svc.SubmitBitLockerPIN(ctx, host, "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "too old")
		require.False(t, ds.QueueBitLockerPINRequestFuncInvoked)
	})

	t.Run("rejects a non-Windows host", func(t *testing.T) {
		host := &fleet.Host{ID: 7, UUID: "mac-uuid", Platform: "darwin"}
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		err := svc.SubmitBitLockerPIN(ctx, host, "123456")
		require.Error(t, err)
		require.False(t, ds.QueueBitLockerPINRequestFuncInvoked)
	})
}

func TestGetBitLockerPINForHost(t *testing.T) {
	t.Run("returns the decrypted PIN", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		encrypted, err := mdm.EncryptAndEncode("654321", testBitLockerPINPrivateKey)
		require.NoError(t, err)
		ds.TakeBitLockerPINRequestFunc = func(ctx context.Context, h *fleet.Host) (string, string, error) {
			require.Equal(t, host.ID, h.ID)
			return encrypted, "req-1", nil
		}

		pin, requestUUID, err := svc.GetBitLockerPINForHost(ctx)
		require.NoError(t, err)
		require.Equal(t, "654321", pin)
		require.Equal(t, "req-1", requestUUID)
	})

	t.Run("passes through nothing-to-collect", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)
		ds.TakeBitLockerPINRequestFunc = func(ctx context.Context, h *fleet.Host) (string, string, error) {
			return "", "", newNotFoundError()
		}

		_, _, err := svc.GetBitLockerPINForHost(ctx)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})
}

func TestSetBitLockerPINOutcome(t *testing.T) {
	t.Run("success records the PIN, the activity and a refetch", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, opts := newBitLockerPINTestService(t, host)

		var pinSet, refetch bool
		ds.SetOrUpdateHostDiskTpmPINFunc = func(ctx context.Context, hostID uint, set bool) error {
			pinSet = set
			return nil
		}
		ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, hostID uint, requested bool) error {
			refetch = requested
			return nil
		}
		var outcome fleet.BitLockerPINRequestStatus
		ds.SetBitLockerPINRequestOutcomeFunc = func(
			ctx context.Context, h *fleet.Host, requestUUID string, o fleet.BitLockerPINRequestStatus, clientError string,
		) error {
			outcome = o
			require.Empty(t, clientError)
			return nil
		}
		var activityName string
		opts.ActivityMock.NewActivityFunc = func(
			_ context.Context, user *activity_api.User, a activity_api.ActivityDetails,
		) error {
			// The end user chose the PIN, so the activity deliberately has no actor rather than being Fleet-initiated.
			require.Nil(t, user)
			activityName = a.ActivityName()
			return nil
		}

		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestSet, ""))
		require.True(t, pinSet)
		require.True(t, refetch)
		require.Equal(t, fleet.BitLockerPINRequestSet, outcome)
		require.Equal(t, "created_disk_encryption_pin", activityName)
	})

	t.Run("failure records the reason and leaves the PIN flag alone", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		var gotError string
		ds.SetBitLockerPINRequestOutcomeFunc = func(
			ctx context.Context, h *fleet.Host, requestUUID string, o fleet.BitLockerPINRequestStatus, clientError string,
		) error {
			require.Equal(t, fleet.BitLockerPINRequestFailed, o)
			gotError = clientError
			return nil
		}

		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestFailed, "  PIN already set  "))
		require.Equal(t, "PIN already set", gotError)
		require.False(t, ds.SetOrUpdateHostDiskTpmPINFuncInvoked)
		require.False(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	})

	t.Run("failure reason is truncated by characters, not bytes", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		var gotError string
		ds.SetBitLockerPINRequestOutcomeFunc = func(
			ctx context.Context, h *fleet.Host, requestUUID string, o fleet.BitLockerPINRequestStatus, clientError string,
		) error {
			gotError = clientError
			return nil
		}

		// A localized Windows error is multi-byte. 300 two-byte characters is 600 bytes, so a byte slice at 255 would
		// cut a character in half and hand MySQL invalid UTF-8.
		reason := strings.Repeat("é", 300)
		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestFailed, reason))

		require.True(t, utf8.ValidString(gotError), "truncated reason must stay valid UTF-8")
		require.Equal(t, bitLockerPINClientErrorMaxLength, utf8.RuneCountInString(gotError))
	})

	t.Run("host updates failing after the outcome is recorded do not fail the report", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, opts := newBitLockerPINTestService(t, host)

		ds.SetBitLockerPINRequestOutcomeFunc = func(
			ctx context.Context, h *fleet.Host, requestUUID string, o fleet.BitLockerPINRequestStatus, clientError string,
		) error {
			return nil
		}
		// Both follow-on writes fail. Returning an error would make the agent retry into a 404, since the submission
		// is already settled, and the activity would never be written.
		ds.SetOrUpdateHostDiskTpmPINFunc = func(ctx context.Context, hostID uint, set bool) error {
			return errors.New("host_disks write failed")
		}
		ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, hostID uint, requested bool) error {
			return errors.New("refetch write failed")
		}
		var activityName string
		opts.ActivityMock.NewActivityFunc = func(
			_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails,
		) error {
			activityName = a.ActivityName()
			return nil
		}

		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestSet, ""))
		require.True(t, ds.SetOrUpdateHostDiskTpmPINFuncInvoked)
		require.True(t, ds.UpdateHostRefetchRequestedFuncInvoked, "a failed tpm_pin_set write must not skip the refetch")
		require.Equal(t, "created_disk_encryption_pin", activityName)
	})

	t.Run("an outcome that matches no collected submission records nothing", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		ds.SetBitLockerPINRequestOutcomeFunc = func(
			ctx context.Context, h *fleet.Host, requestUUID string, o fleet.BitLockerPINRequestStatus, clientError string,
		) error {
			return newNotFoundError()
		}

		err := svc.SetBitLockerPINOutcome(ctx, "forged", fleet.BitLockerPINRequestSet, "")
		require.True(t, fleet.IsNotFound(err))
		// The whole point of recording the outcome first: a forged success must not mark the host as having a PIN.
		require.False(t, ds.SetOrUpdateHostDiskTpmPINFuncInvoked)
		require.False(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	})

	t.Run("failure without a reason is rejected", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		err := svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestFailed, "   ")
		require.Error(t, err)
		require.Contains(t, err.Error(), "client_error")
		require.False(t, ds.SetBitLockerPINRequestOutcomeFuncInvoked)
	})

	t.Run("unknown outcome is rejected", func(t *testing.T) {
		host := windowsPINHost()
		svc, ds, ctx, _ := newBitLockerPINTestService(t, host)

		err := svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestPending, "")
		require.Error(t, err)
		require.False(t, ds.SetBitLockerPINRequestOutcomeFuncInvoked)
	})
}
