package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/stretchr/testify/require"
)

// loadEnrollSecret is shared by the two delivery channels (the packaged secret file and, on
// Windows, the registry value Fleet MDM writes). onDelivered is what discards the delivery copy, so
// the contract that matters is when it does and does not fire: never while the secret might still be
// needed for another attempt.
func TestLoadEnrollSecretDiscardsTheDeliveryCopyOnlyOnceStored(t *testing.T) {
	for _, tc := range []struct {
		name            string
		keystore        *fakeKeystore
		disableKeystore bool
		wantDelivered   bool
		wantAdds        int
		wantUpdates     int
	}{
		{
			name:          "keystore empty, secret added",
			keystore:      &fakeKeystore{supported: true},
			wantDelivered: true,
			wantAdds:      1,
		},
		{
			name:          "keystore holds a different secret, so it is updated",
			keystore:      &fakeKeystore{supported: true, secret: "stale"},
			wantDelivered: true,
			wantUpdates:   1,
		},
		{
			name:          "keystore already holds this secret",
			keystore:      &fakeKeystore{supported: true, secret: "delivered"},
			wantDelivered: true,
		},
		{
			// The delivery copy has to survive so the next start can retry.
			name:          "add fails",
			keystore:      &fakeKeystore{supported: true, addErr: errors.New("boom")},
			wantDelivered: false,
			wantAdds:      1,
		},
		{
			name:          "update fails",
			keystore:      &fakeKeystore{supported: true, secret: "stale", updateErr: errors.New("boom")},
			wantDelivered: false,
			wantUpdates:   1,
		},
		{
			name:          "keystore unreadable",
			keystore:      &fakeKeystore{supported: true, getErr: errors.New("boom")},
			wantDelivered: false,
		},
		{
			// With no keystore there is nothing to sync into, so the delivery copy is left alone.
			name:          "keystore unsupported",
			keystore:      &fakeKeystore{supported: false},
			wantDelivered: false,
		},
		{
			name:            "keystore disabled",
			keystore:        &fakeKeystore{supported: true},
			disableKeystore: true,
			wantDelivered:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var set string
			delivered := false

			err := loadEnrollSecret("delivered", tc.keystore, tc.disableKeystore,
				func(secret string) error { set = secret; return nil },
				func() { delivered = true },
			)

			require.NoError(t, err)
			require.Equal(t, "delivered", set, "the secret must become active regardless of keystore outcome")
			require.Equal(t, tc.wantDelivered, delivered)
			require.Equal(t, tc.wantAdds, tc.keystore.addCalls)
			require.Equal(t, tc.wantUpdates, tc.keystore.updateCalls)
		})
	}
}

func TestLoadEnrollSecretPropagatesSetFailure(t *testing.T) {
	delivered := false

	err := loadEnrollSecret("delivered", &fakeKeystore{supported: true}, false,
		func(string) error { return errors.New("boom") },
		func() { delivered = true },
	)

	require.Error(t, err)
	require.False(t, delivered, "a secret that could not be made active must not have its delivery copy discarded")
}

// The delivery channel is injected so this never reads or clears the host's real enroll secret. On
// Windows that lives in HKLM, so calling the production path here would make the result depend on
// whatever machine the test runs on, and could clear a secret the host still needs.
func TestLoadDeliveredEnrollSecret(t *testing.T) {
	for _, tc := range []struct {
		name        string
		readErr     error
		wantLoaded  bool
		wantAdds    int
		wantCleared bool
		wantErr     bool
	}{
		{
			// The ordinary state on a host that has already loaded its secret: nothing waiting is not
			// an error, so orbit falls through to the file and keystore rather than failing to start.
			name:    "nothing waiting",
			readErr: profiles.ErrEnrollSecretNotFound,
		},
		{
			// Platforms with no delivery channel at all take the same path.
			name:    "no delivery channel on this platform",
			readErr: profiles.ErrNotImplemented,
		},
		{
			name:    "delivery channel fails",
			readErr: errors.New("registry exploded"),
			wantErr: true,
		},
		{
			name:        "secret waiting",
			wantLoaded:  true,
			wantAdds:    1,
			wantCleared: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ks := &fakeKeystore{supported: true}
			cleared := false
			var set string

			// Fleet may write the same secret to the installer's file as well, so that an older fleetd
			// that cannot read the registry still enrolls. Loading has to retire that copy too.
			secretPath := filepath.Join(t.TempDir(), "secret.txt")
			require.NoError(t, os.WriteFile(secretPath, []byte("delivered"), 0o600))

			loaded, err := loadDeliveredEnrollSecret(
				func() (string, error) { return "delivered", tc.readErr },
				func() error { cleared = true; return nil },
				secretPath, ks, false,
				func(secret string) error { set = secret; return nil },
			)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantLoaded, loaded)
			require.Equal(t, tc.wantAdds, ks.addCalls)
			require.Equal(t, tc.wantCleared, cleared, "the delivery copy is discarded only once stored")

			_, statErr := os.Stat(secretPath)
			if tc.wantCleared {
				require.ErrorIs(t, statErr, os.ErrNotExist, "the installer's copy must not outlive loading")
			} else {
				require.NoError(t, statErr, "a secret that was not loaded leaves the file for the file path to read")
			}

			if tc.wantLoaded {
				require.Equal(t, "delivered", set)
			} else {
				require.Empty(t, set, "nothing should be set when no secret was loaded")
			}
		})
	}
}
