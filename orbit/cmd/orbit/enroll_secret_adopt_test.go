package main

import (
	"errors"
	"testing"

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

// On platforms with no MDM delivery channel the absence of a secret is the ordinary state, not an
// error, so orbit must fall through to the file and keystore rather than failing to start.
func TestAdoptMDMDeliveredEnrollSecretReportsNothingWaiting(t *testing.T) {
	ks := &fakeKeystore{supported: true}

	adopted, err := adoptMDMDeliveredEnrollSecret(ks, false, func(string) error {
		t.Fatal("no secret should have been set")
		return nil
	})

	require.NoError(t, err)
	require.False(t, adopted)
	require.Zero(t, ks.addCalls)
}
