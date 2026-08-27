package service

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// These tests cover the gate that decides whether the agent is asked to restore protection.
func TestShouldEnableBitLockerProtection(t *testing.T) {
	protectionOff := fleet.BitLockerProtectionStatusOff
	protectionOn := fleet.BitLockerProtectionStatusOn

	for _, tc := range []struct {
		name         string
		encrypted    *bool
		protection   *int
		tpmPINSet    bool
		pinRequired  bool
		wantNotified bool
	}{
		{
			name:         "encrypted and protection off is the case this exists for",
			encrypted:    new(true),
			protection:   &protectionOff,
			wantNotified: true,
		},
		{
			name:       "protection on needs nothing",
			encrypted:  new(true),
			protection: &protectionOn,
		},
		{
			name:       "not encrypted is the encryption flow's job, not this one",
			encrypted:  new(false),
			protection: &protectionOff,
		},
		{
			// An unknown status is not evidence of anything; acting on it is how protectors get destroyed.
			name:       "unknown protection status is never acted on",
			encrypted:  new(true),
			protection: nil,
		},
		{
			// Policy forbids a TPM-only protector, and only the end user can enrol a PIN, so Fleet cannot repair this
			// host. It stays "Action required" rather than the agent failing on every config poll.
			name:        "PIN required but not set cannot be repaired by Fleet",
			encrypted:   new(true),
			protection:  &protectionOff,
			tpmPINSet:   false,
			pinRequired: true,
		},
		{
			name:         "PIN required and set can be repaired",
			encrypted:    new(true),
			protection:   &protectionOff,
			tpmPINSet:    true,
			pinRequired:  true,
			wantNotified: true,
		},
		{
			name:       "host has not reported its disks yet",
			encrypted:  nil,
			protection: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &fleet.Host{
				DiskEncryptionEnabled:     tc.encrypted,
				BitLockerProtectionStatus: tc.protection,
				TPMPINSet:                 tc.tpmPINSet,
			}
			// Only BitLockerPINRequired is read here; the platform gate lives in the caller.
			diskEncryption := fleet.DiskEncryptionConfig{BitLockerPINRequired: tc.pinRequired}

			require.Equal(t, tc.wantNotified, shouldEnableBitLockerProtection(host, diskEncryption))
		})
	}
}

func TestSetOrUpdateDiskEncryptionProtection(t *testing.T) {
	// recorder captures what reached the datastore, so each case asserts on values rather than only on invocation.
	type recorder struct {
		storedError  string
		errorSet     bool
		refetchValue *bool
	}
	setup := func() (*Service, *mock.Store, *recorder) {
		ds := new(mock.Store)
		svcAuthz, err := authz.NewAuthorizer()
		require.NoError(t, err)
		svc := &Service{ds: ds, authz: svcAuthz}
		rec := &recorder{}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) { return true, nil }
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(_ context.Context, _ uint, e string) error {
			rec.errorSet, rec.storedError = true, e
			return nil
		}
		ds.UpdateHostRefetchRequestedFunc = func(_ context.Context, _ uint, value bool) error {
			rec.refetchValue = &value
			return nil
		}
		return svc, ds, rec
	}
	ctx := func() context.Context {
		return host.NewContext(t.Context(), &fleet.Host{ID: 1})
	}

	t.Run("restored clears the reason and asks for a refetch", func(t *testing.T) {
		svc, _, rec := setup()

		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionRestored, ""))
		require.True(t, rec.errorSet)
		require.Empty(t, rec.storedError)
		// osquery owns the protection status, so the status flips on evidence rather than on the agent's assertion.
		require.NotNil(t, rec.refetchValue, "must request a refetch")
		require.True(t, *rec.refetchValue)
	})

	t.Run("failure records the reason for the admin", func(t *testing.T) {
		svc, _, rec := setup()

		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed, "the TPM is not ready"))
		require.Equal(t, "the TPM is not ready", rec.storedError)
		require.Nil(t, rec.refetchValue, "a host that was not repaired has nothing new to observe")
	})

	t.Run("never writes the disk encryption key", func(t *testing.T) {
		for _, outcome := range []fleet.DiskEncryptionProtectionOutcome{
			fleet.DiskEncryptionProtectionRestored,
			fleet.DiskEncryptionProtectionDeferred,
			fleet.DiskEncryptionProtectionFailed,
		} {
			t.Run(string(outcome), func(t *testing.T) {
				svc, ds, _ := setup()
				reason := ""
				if outcome != fleet.DiskEncryptionProtectionRestored {
					reason = "a reason"
				}

				require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), outcome, reason))
				require.False(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked, "must not touch the escrowed recovery key")
			})
		}
	})

	t.Run("rejects an unknown outcome", func(t *testing.T) {
		svc, ds, rec := setup()

		err := svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionOutcome("nonsense"), "a reason")
		require.Error(t, err)
		require.False(t, rec.errorSet, "a rejected outcome must not reach the datastore")
		require.Nil(t, rec.refetchValue)
		require.False(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	})

	t.Run("rejects a failure with no reason", func(t *testing.T) {
		for _, outcome := range []fleet.DiskEncryptionProtectionOutcome{
			fleet.DiskEncryptionProtectionFailed,
			fleet.DiskEncryptionProtectionDeferred,
		} {
			t.Run(string(outcome), func(t *testing.T) {
				svc, _, rec := setup()

				require.Error(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), outcome, ""))
				// A blank reason would leave the admin with an "Action required" and nothing to act on.
				require.False(t, rec.errorSet)
			})
		}
	})

	t.Run("truncates an over-long reason to a whole number of runes", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			char string
		}{
			{name: "ascii", char: "x"},
			// A rune straddling the cut would be split by a byte-wise slice, and the utf8mb4 write would then fail.
			{name: "multibyte", char: "é"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc, _, rec := setup()

				require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed,
					strings.Repeat(tc.char, bitLockerProtectionErrorMaxLength+50)))
				require.True(t, utf8.ValidString(rec.storedError))
				// The column is bounded in characters, not bytes.
				require.Equal(t, bitLockerProtectionErrorMaxLength, utf8.RuneCountInString(rec.storedError))
			})
		}
	})
}
