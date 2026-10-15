package update

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/client"
	"github.com/fleetdm/fleet/v4/orbit/pkg/bitlocker"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsMDMBitlockerPIN(t *testing.T) {
	// Not parallel: the subtests capture the global logger.
	var logBuf bytes.Buffer
	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	const (
		pin                    = `open "sesame" \ 42`
		requestUUID            = "request-uuid"
		resubmittedRequestUUID = "resubmitted-request-uuid"
	)
	notFound := &client.NotFoundErr{}
	networkErr := errors.New("connection refused")

	pinPending := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{BitLockerPINRequestPending: true}}
	nothingPending := &fleet.OrbitConfig{}

	type fixture struct {
		receiver  *windowsMDMBitlockerConfigReceiver
		server    *mockDiskEncryptionKeySetter
		appliedTo []string
	}
	newFixture := func(t *testing.T, applyErr error) *fixture {
		logBuf.Reset()
		// Every path through the receiver logs, and none of it may carry the PIN.
		t.Cleanup(func() { assert.NotContains(t, logBuf.String(), pin) })
		f := &fixture{server: &mockDiskEncryptionKeySetter{
			GetPINDetailsImpl: func() (string, string, error) { return pin, requestUUID, nil },
			SetPINResultImpl:  func(pinOutcome) error { return nil },
		}}
		f.receiver = &windowsMDMBitlockerConfigReceiver{
			EncryptionResult: f.server,
			execSetTPMAndPINProtectorFn: func(volumeID, gotPIN string) error {
				// Not require: this runs inside the receiver, and a failure here must not stop the subtest's goroutine.
				assert.Equal(t, pin, gotPIN)
				f.appliedTo = append(f.appliedTo, volumeID)
				return applyErr
			},
		}
		return f
	}
	setOutcome := pinOutcome{requestUUID: requestUUID, outcome: fleet.BitLockerPINRequestSet}

	t.Run("a pending PIN is applied and reported", func(t *testing.T) {
		f := newFixture(t, nil)
		require.NoError(t, f.receiver.Run(pinPending))

		require.Equal(t, []string{"C:"}, f.appliedTo)
		require.Equal(t, []pinOutcome{setOutcome}, f.server.PINResults)
		require.Nil(t, f.receiver.heldPINOutcome)
	})

	t.Run("nothing is collected without the pending flag", func(t *testing.T) {
		f := newFixture(t, nil)
		require.NoError(t, f.receiver.Run(nothingPending))

		require.Zero(t, f.server.GetPINDetailsCalls)
		require.Empty(t, f.server.PINResults)
	})

	t.Run("nothing is collected while another BitLocker operation holds the lock", func(t *testing.T) {
		f := newFixture(t, nil)
		f.receiver.mu.Lock()
		require.NoError(t, f.receiver.Run(pinPending))
		f.receiver.mu.Unlock()

		require.Zero(t, f.server.GetPINDetailsCalls)
		require.Empty(t, f.appliedTo)
	})

	t.Run("nothing is collected while encryption is being enforced", func(t *testing.T) {
		f := newFixture(t, nil)
		f.receiver.encryptionRetryAfter = time.Now().Add(time.Hour)
		cfg := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			BitLockerPINRequestPending: true,
			EnforceBitLockerEncryption: true,
		}}
		require.NoError(t, f.receiver.Run(cfg))

		require.Zero(t, f.server.GetPINDetailsCalls)
	})

	requireNothingHappened := func(t *testing.T, f *fixture) {
		t.Helper()
		require.Empty(t, f.appliedTo)
		require.Empty(t, f.server.PINResults)
		require.Nil(t, f.receiver.heldPINOutcome)
	}

	t.Run("a collect 404 means nothing to apply", func(t *testing.T) {
		f := newFixture(t, nil)
		f.server.GetPINDetailsImpl = func() (string, string, error) { return "", "", notFound }
		require.NoError(t, f.receiver.Run(pinPending))

		requireNothingHappened(t, f)
	})

	t.Run("a failed collect is retried on the next poll", func(t *testing.T) {
		f := newFixture(t, nil)
		f.server.GetPINDetailsImpl = func() (string, string, error) { return "", "", networkErr }
		require.NoError(t, f.receiver.Run(pinPending))
		requireNothingHappened(t, f)

		// The server still has the PIN, so a later poll collects it. Nothing from the failure may stand in the way.
		f.server.GetPINDetailsImpl = func() (string, string, error) { return pin, requestUUID, nil }
		require.NoError(t, f.receiver.Run(pinPending))

		require.Equal(t, []string{"C:"}, f.appliedTo)
		require.Equal(t, []pinOutcome{setOutcome}, f.server.PINResults)
		require.Nil(t, f.receiver.heldPINOutcome)
	})

	for _, tc := range []struct {
		name       string
		applyErr   error
		wantReason string
	}{
		{
			name:       "an apply failure reports its reason",
			applyErr:   &bitlocker.PINError{Reason: bitlocker.PINReasonAlreadySet, Err: errors.New("a protector of type 4 exists")},
			wantReason: bitlocker.PINReasonAlreadySet,
		},
		{
			name:       "an apply failure without a reason still reports one",
			applyErr:   errors.New("COM worker is closed"),
			wantReason: bitlocker.PINReasonNotFinished,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, tc.applyErr)
			require.NoError(t, f.receiver.Run(pinPending))

			require.Equal(t, []pinOutcome{{
				requestUUID: requestUUID,
				outcome:     fleet.BitLockerPINRequestFailed,
				clientError: tc.wantReason,
			}}, f.server.PINResults)
		})
	}

	t.Run("an unreported outcome is held and retried on a later poll without the bitlocker_pin_request_pending flag", func(t *testing.T) {
		f := newFixture(t, nil)
		f.server.SetPINResultImpl = func(pinOutcome) error { return networkErr }
		require.NoError(t, f.receiver.Run(pinPending))
		require.Equal(t, &setOutcome, f.receiver.heldPINOutcome)

		f.server.SetPINResultImpl = func(pinOutcome) error { return nil }
		require.NoError(t, f.receiver.Run(nothingPending))

		require.Equal(t, []pinOutcome{setOutcome, setOutcome}, f.server.PINResults)
		require.Equal(t, 1, f.server.GetPINDetailsCalls)
		require.Len(t, f.appliedTo, 1)
		require.Nil(t, f.receiver.heldPINOutcome)
	})

	t.Run("a held outcome is retried while the server asks for a repair", func(t *testing.T) {
		f := newFixture(t, nil)
		f.server.SetPINResultImpl = func(pinOutcome) error { return networkErr }
		require.NoError(t, f.receiver.Run(pinPending))
		require.Equal(t, &setOutcome, f.receiver.heldPINOutcome)

		// The repair branches return early, and the server times a collected PIN out an hour after the agent took it.
		f.server.SetPINResultImpl = func(pinOutcome) error { return nil }
		f.receiver.encryptionRetryAfter = time.Now().Add(time.Hour)
		repairing := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{EnforceBitLockerEncryption: true}}
		require.NoError(t, f.receiver.Run(repairing))

		require.Equal(t, []pinOutcome{setOutcome, setOutcome}, f.server.PINResults)
		require.Nil(t, f.receiver.heldPINOutcome)
	})

	t.Run("an outcome the server no longer wants is dropped", func(t *testing.T) {
		f := newFixture(t, nil)
		f.server.SetPINResultImpl = func(pinOutcome) error { return notFound }
		require.NoError(t, f.receiver.Run(pinPending))
		require.Nil(t, f.receiver.heldPINOutcome)

		require.NoError(t, f.receiver.Run(nothingPending))
		require.Len(t, f.server.PINResults, 1)
	})
}
