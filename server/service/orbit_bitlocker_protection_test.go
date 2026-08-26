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

// Fleet used to detect that a host was encrypted but unprotected, display it, and never act. These cover the gate that
// decides whether the agent is asked to restore protection.
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
			// The LEFT JOIN on host_disks yields NULLs when the host has never reported a disk.
			name:       "host has not reported its disks yet",
			encrypted:  nil,
			protection: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &fleet.Host{
				ID:                        1,
				TeamID:                    new(uint(1)),
				DiskEncryptionEnabled:     tc.encrypted,
				BitLockerProtectionStatus: tc.protection,
				TPMPINSet:                 tc.tpmPINSet,
			}
			diskEncryption := fleet.DiskEncryptionConfig{WindowsEnabled: true, BitLockerPINRequired: tc.pinRequired}

			require.Equal(t, tc.wantNotified, shouldEnableBitLockerProtection(host, diskEncryption))
		})
	}
}

// Reporting an outcome must never touch host_disk_encryption_keys: that write path also owns base64_encrypted and
// would blank the escrowed recovery key of the hosts that still need it.
func TestSetOrUpdateDiskEncryptionProtection(t *testing.T) {
	setup := func() (*Service, *mock.Store) {
		ds := new(mock.Store)
		svcAuthz, err := authz.NewAuthorizer()
		require.NoError(t, err)
		svc := &Service{ds: ds, authz: svcAuthz}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) { return true, nil }
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(ctx context.Context, hostID uint, e string) error { return nil }
		ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, hostID uint, value bool) error { return nil }
		return svc, ds
	}
	ctx := func() context.Context {
		return host.NewContext(t.Context(), &fleet.Host{ID: 1})
	}

	t.Run("restored clears the reason and asks for a refetch", func(t *testing.T) {
		svc, ds := setup()
		var cleared string
		var clearCalled bool
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(_ context.Context, _ uint, e string) error {
			clearCalled, cleared = true, e
			return nil
		}

		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionRestored, ""))
		require.True(t, clearCalled)
		require.Empty(t, cleared)
		// osquery owns the protection status, so the status flips on evidence rather than on the agent's assertion.
		require.True(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	})

	t.Run("failure records the reason for the admin", func(t *testing.T) {
		svc, ds := setup()
		var stored string
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(_ context.Context, _ uint, e string) error {
			stored = e
			return nil
		}

		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed, "the TPM is not ready"))
		require.Equal(t, "the TPM is not ready", stored)
		require.False(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	})

	t.Run("never writes the disk encryption key", func(t *testing.T) {
		svc, ds := setup()
		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed, "boom"))
		require.False(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked, "must not touch the escrowed recovery key")
	})

	t.Run("rejects an unknown outcome", func(t *testing.T) {
		svc, _ := setup()
		err := svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionOutcome("nonsense"), "a reason")
		require.Error(t, err)
	})

	t.Run("rejects a failure with no reason", func(t *testing.T) {
		svc, ds := setup()
		for _, outcome := range []fleet.DiskEncryptionProtectionOutcome{
			fleet.DiskEncryptionProtectionFailed,
			fleet.DiskEncryptionProtectionDeferred,
		} {
			err := svc.SetOrUpdateDiskEncryptionProtection(ctx(), outcome, "")
			require.Error(t, err, outcome)
			// A blank reason would leave the admin with an "Action required" and nothing to act on.
			require.False(t, ds.SetOrUpdateHostBitLockerProtectionErrorFuncInvoked, outcome)
		}
	})

	t.Run("truncates a reason too long for the column", func(t *testing.T) {
		svc, ds := setup()
		var stored string
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(_ context.Context, _ uint, e string) error {
			stored = e
			return nil
		}

		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed,
			strings.Repeat("x", bitLockerProtectionErrorMaxLength+50)))
		require.Len(t, stored, bitLockerProtectionErrorMaxLength)
	})

	t.Run("truncates a multibyte reason without corrupting it", func(t *testing.T) {
		svc, ds := setup()
		var stored string
		ds.SetOrUpdateHostBitLockerProtectionErrorFunc = func(_ context.Context, _ uint, e string) error {
			stored = e
			return nil
		}

		// A rune that straddles the cut would be split by a byte-wise slice, and the utf8mb4 write would then fail.
		require.NoError(t, svc.SetOrUpdateDiskEncryptionProtection(ctx(), fleet.DiskEncryptionProtectionFailed,
			strings.Repeat("é", bitLockerProtectionErrorMaxLength+50)))
		require.True(t, utf8.ValidString(stored))
		require.Equal(t, bitLockerProtectionErrorMaxLength, utf8.RuneCountInString(stored))
	})
}
