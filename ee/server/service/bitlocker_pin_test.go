package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

// testBitLockerPINPrivateKey is a 32-byte key so AES-256 encryption of the PIN works in tests.
const testBitLockerPINPrivateKey = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456"

// newBitLockerPINTestService builds a service for a Windows host that needs a PIN and whose fleetd can set one.
func newBitLockerPINTestService(t *testing.T) (*Service, *mock.Store, *svcmock.Service, *fleet.Host, context.Context) {
	t.Helper()

	ds := new(mock.Store)
	svc, base := newTestServiceWithMock(t, ds)
	svc.logger = slog.New(slog.DiscardHandler)
	svc.config.Server.PrivateKey = testBitLockerPINPrivateKey

	ds.GetMDMWindowsHostConfigStateFunc = func(context.Context, string) (*fleet.MDMWindowsHostConfigState, error) {
		return &fleet.MDMWindowsHostConfigState{FleetdBitLockerPINCapable: true}, nil
	}
	ds.GetMDMWindowsBitLockerStatusFunc = func(context.Context, *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
		return &fleet.HostMDMDiskEncryption{ActionRequired: new(fleet.ActionRequiredCreatePIN)}, nil
	}

	host := &fleet.Host{ID: 42, UUID: "host-uuid", Platform: "windows", Hostname: "MU-TH-UR"}
	return svc, ds, base, host, test.HostContext(t.Context(), host)
}

type recordedBitLockerPINOutcome struct {
	requestUUID string
	outcome     fleet.BitLockerPINRequestStatus
	clientError string
}

// recordBitLockerPINOutcomes makes the datastore accept any outcome and returns what it last recorded.
func recordBitLockerPINOutcomes(ds *mock.Store) *recordedBitLockerPINOutcome {
	rec := &recordedBitLockerPINOutcome{}
	ds.SetBitLockerPINRequestOutcomeFunc = func(
		_ context.Context, _ *fleet.Host, requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string,
	) error {
		*rec = recordedBitLockerPINOutcome{requestUUID, outcome, clientError}
		return nil
	}
	return rec
}

func TestSubmitBitLockerPIN(t *testing.T) {
	t.Run("queues the PIN encrypted", func(t *testing.T) {
		svc, ds, _, host, ctx := newBitLockerPINTestService(t)
		var queued string
		ds.QueueBitLockerPINRequestFunc = func(_ context.Context, h *fleet.Host, encryptedPIN string) error {
			require.Equal(t, host.ID, h.ID)
			queued = encryptedPIN
			return nil
		}

		require.NoError(t, svc.SubmitBitLockerPIN(ctx, host, "123456"))
		require.NotEqual(t, "123456", queued)
		decrypted, err := mdm.DecodeAndDecrypt(queued, testBitLockerPINPrivateKey)
		require.NoError(t, err)
		require.Equal(t, "123456", decrypted)
	})

	t.Run("rejects an invalid PIN before any lookup", func(t *testing.T) {
		svc, ds, _, host, ctx := newBitLockerPINTestService(t)

		require.ErrorContains(t, svc.SubmitBitLockerPIN(ctx, host, "12ab"), microsoft_mdm.BitLockerPINLengthMessage)
		require.False(t, ds.GetMDMWindowsHostConfigStateFuncInvoked)
	})

	for _, tc := range []struct {
		name              string
		setup             func(ds *mock.Store, host *fleet.Host)
		wantMessage       string
		wantStatusChecked bool
	}{
		{
			name:        "a non-Windows host",
			setup:       func(_ *mock.Store, host *fleet.Host) { host.Platform = "darwin" },
			wantMessage: bitLockerPINNotNeededMessage,
		},
		{
			name: "a host not enrolled in Windows MDM",
			setup: func(ds *mock.Store, _ *fleet.Host) {
				ds.GetMDMWindowsHostConfigStateFunc = func(context.Context, string) (*fleet.MDMWindowsHostConfigState, error) {
					return nil, &notFoundError{}
				}
			},
			wantMessage: bitLockerPINNotNeededMessage,
		},
		{
			// Capability is checked first, so the BitLocker status is never read for an agent that can't apply a PIN.
			name: "a host whose fleetd cannot apply a PIN",
			setup: func(ds *mock.Store, _ *fleet.Host) {
				ds.GetMDMWindowsHostConfigStateFunc = func(context.Context, string) (*fleet.MDMWindowsHostConfigState, error) {
					return &fleet.MDMWindowsHostConfigState{}, nil
				}
			},
			wantMessage: bitLockerPINAgentTooOldMessage,
		},
		{
			// The page may be stale: another session set the PIN, or the fleet stopped requiring one.
			name: "a host that does not need a PIN",
			setup: func(ds *mock.Store, _ *fleet.Host) {
				ds.GetMDMWindowsBitLockerStatusFunc = func(context.Context, *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
					return &fleet.HostMDMDiskEncryption{}, nil
				}
			},
			wantMessage:       bitLockerPINNotNeededMessage,
			wantStatusChecked: true,
		},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			svc, ds, _, host, ctx := newBitLockerPINTestService(t)
			tc.setup(ds, host)

			require.ErrorContains(t, svc.SubmitBitLockerPIN(ctx, host, "123456"), tc.wantMessage)
			require.Equal(t, tc.wantStatusChecked, ds.GetMDMWindowsBitLockerStatusFuncInvoked)
		})
	}
}

func TestGetBitLockerPINForHost(t *testing.T) {
	t.Run("returns the decrypted PIN", func(t *testing.T) {
		svc, ds, _, host, ctx := newBitLockerPINTestService(t)
		encrypted, err := mdm.EncryptAndEncode("654321", testBitLockerPINPrivateKey)
		require.NoError(t, err)
		ds.TakeBitLockerPINRequestFunc = func(_ context.Context, h *fleet.Host) (string, string, error) {
			require.Equal(t, host.ID, h.ID)
			return encrypted, "req-1", nil
		}

		pin, requestUUID, err := svc.GetBitLockerPINForHost(ctx)
		require.NoError(t, err)
		require.Equal(t, "654321", pin)
		require.Equal(t, "req-1", requestUUID)
	})

	t.Run("passes through nothing to collect", func(t *testing.T) {
		svc, ds, _, _, ctx := newBitLockerPINTestService(t)
		ds.TakeBitLockerPINRequestFunc = func(context.Context, *fleet.Host) (string, string, error) {
			return "", "", &notFoundError{}
		}

		_, _, err := svc.GetBitLockerPINForHost(ctx)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("a missing private key does not consume the submission", func(t *testing.T) {
		svc, ds, _, _, ctx := newBitLockerPINTestService(t)
		svc.config.Server.PrivateKey = ""
		ds.TakeBitLockerPINRequestFunc = func(context.Context, *fleet.Host) (string, string, error) {
			return "ciphertext", "req-1", nil
		}

		_, _, err := svc.GetBitLockerPINForHost(ctx)
		require.Error(t, err)
		// Collecting is destructive, so the key is checked before the submission is taken.
		require.False(t, ds.TakeBitLockerPINRequestFuncInvoked)
	})

	t.Run("an unreadable PIN retires the submission as failed", func(t *testing.T) {
		svc, ds, _, _, ctx := newBitLockerPINTestService(t)
		ds.TakeBitLockerPINRequestFunc = func(context.Context, *fleet.Host) (string, string, error) {
			return "not-ciphertext", "req-1", nil
		}
		rec := recordBitLockerPINOutcomes(ds)

		_, _, err := svc.GetBitLockerPINForHost(ctx)
		require.Error(t, err)
		// The ciphertext is already gone, so the page is told the submission failed rather than left waiting.
		require.Equal(t, recordedBitLockerPINOutcome{"req-1", fleet.BitLockerPINRequestFailed, bitLockerPINUnreadableError}, *rec)
	})
}

func TestSetBitLockerPINOutcome(t *testing.T) {
	createdPINActivity := fleet.ActivityTypeCreatedDiskEncryptionPIN{}.ActivityName()

	t.Run("success records the outcome, the PIN, a refetch and the activity", func(t *testing.T) {
		svc, ds, base, _, ctx := newBitLockerPINTestService(t)
		rec := recordBitLockerPINOutcomes(ds)
		var bootProtectorSet, pinSet, refetch bool
		ds.SetOrUpdateHostDiskBitLockerProtectorsFunc = func(_ context.Context, _ uint, bootProtector, tpmPIN bool) error {
			bootProtectorSet, pinSet = bootProtector, tpmPIN
			return nil
		}
		ds.UpdateHostRefetchRequestedFunc = func(_ context.Context, _ uint, requested bool) error {
			refetch = requested
			return nil
		}
		var activityName string
		base.NewActivityFunc = func(_ context.Context, user *fleet.User, a fleet.ActivityDetails) error {
			// The end user chose the PIN, so the activity has no actor rather than being Fleet-initiated.
			require.Nil(t, user)
			activityName = a.ActivityName()
			return nil
		}

		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestSet, ""))
		require.Equal(t, recordedBitLockerPINOutcome{"req-1", fleet.BitLockerPINRequestSet, ""}, *rec)
		require.True(t, bootProtectorSet)
		require.True(t, pinSet)
		require.True(t, refetch)
		require.Equal(t, createdPINActivity, activityName)
	})

	t.Run("failure records the reason trimmed and truncated by characters", func(t *testing.T) {
		svc, ds, _, _, ctx := newBitLockerPINTestService(t)
		rec := recordBitLockerPINOutcomes(ds)

		reason := "  " + strings.Repeat("é", 300) + "  "
		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestFailed, reason))
		want := strings.Repeat("é", fleet.BitLockerPINClientErrorMaxLength)
		require.Equal(t, recordedBitLockerPINOutcome{"req-1", fleet.BitLockerPINRequestFailed, want}, *rec)
	})

	t.Run("host updates failing after the outcome is recorded do not fail the report", func(t *testing.T) {
		svc, ds, base, _, ctx := newBitLockerPINTestService(t)
		recordBitLockerPINOutcomes(ds)
		ds.SetOrUpdateHostDiskBitLockerProtectorsFunc = func(context.Context, uint, bool, bool) error {
			return errors.New("host_disks write failed")
		}
		ds.UpdateHostRefetchRequestedFunc = func(context.Context, uint, bool) error {
			return errors.New("refetch write failed")
		}
		var activityName string
		base.NewActivityFunc = func(_ context.Context, _ *fleet.User, a fleet.ActivityDetails) error {
			activityName = a.ActivityName()
			return nil
		}

		require.NoError(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestSet, ""))
		require.True(t, ds.UpdateHostRefetchRequestedFuncInvoked, "a failed protector write must not skip the refetch")
		require.Equal(t, createdPINActivity, activityName)
	})

	t.Run("an outcome that matches no collected submission records nothing else", func(t *testing.T) {
		svc, ds, _, _, ctx := newBitLockerPINTestService(t)
		ds.SetBitLockerPINRequestOutcomeFunc = func(context.Context, *fleet.Host, string, fleet.BitLockerPINRequestStatus, string) error {
			return &notFoundError{}
		}

		require.True(t, fleet.IsNotFound(svc.SetBitLockerPINOutcome(ctx, "forged", fleet.BitLockerPINRequestSet, "")))
	})

	t.Run("rejects a failure without a reason", func(t *testing.T) {
		svc, _, _, _, ctx := newBitLockerPINTestService(t)

		require.ErrorContains(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestFailed, "   "), "client_error")
	})

	t.Run("rejects an unknown outcome", func(t *testing.T) {
		svc, _, _, _, ctx := newBitLockerPINTestService(t)

		var badRequest *fleet.BadRequestError
		require.ErrorAs(t, svc.SetBitLockerPINOutcome(ctx, "req-1", fleet.BitLockerPINRequestPending, ""), &badRequest)
	})
}
