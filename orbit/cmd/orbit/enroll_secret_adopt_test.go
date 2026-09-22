package main

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/stretchr/testify/require"
)

// adoptEnrollSecret is shared by the two delivery channels (the packaged secret file and, on
// Windows, the registry value Fleet MDM writes). onDelivered is what discards the delivery copy, so
// the contract that matters is when it does and does not fire: never while the secret might still be
// needed for another attempt.
func TestAdoptEnrollSecretDiscardsTheDeliveryCopyOnlyOnceStored(t *testing.T) {
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

			err := adoptEnrollSecret("delivered", tc.keystore, tc.disableKeystore,
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

func TestAdoptEnrollSecretPropagatesSetFailure(t *testing.T) {
	delivered := false

	err := adoptEnrollSecret("delivered", &fakeKeystore{supported: true}, false,
		func(string) error { return errors.New("boom") },
		func() { delivered = true },
	)

	require.Error(t, err)
	require.False(t, delivered, "a secret that could not be made active must not have its delivery copy discarded")
}

// The delivery channel is injected so this never reads or clears the host's real enroll secret. On
// Windows that lives in HKLM, so calling the production path here would make the result depend on
// whatever machine the test runs on, and could clear a secret the host still needs.
func TestAdoptDeliveredEnrollSecret(t *testing.T) {
	for _, tc := range []struct {
		name        string
		readErr     error
		wantAdopted bool
		wantAdds    int
		wantCleared bool
		wantErr     bool
	}{
		{
			// The ordinary state on a host that has already adopted its secret: nothing waiting is not
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
			wantAdopted: true,
			wantAdds:    1,
			wantCleared: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ks := &fakeKeystore{supported: true}
			cleared := false
			var set string

			adopted, err := adoptDeliveredEnrollSecret(
				func() (string, error) { return "delivered", tc.readErr },
				func() error { cleared = true; return nil },
				ks, false,
				func(secret string) error { set = secret; return nil },
			)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantAdopted, adopted)
			require.Equal(t, tc.wantAdds, ks.addCalls)
			require.Equal(t, tc.wantCleared, cleared, "the delivery copy is discarded only once stored")
			if tc.wantAdopted {
				require.Equal(t, "delivered", set)
			} else {
				require.Empty(t, set, "nothing should be set when no secret was adopted")
			}
		})
	}
}
