package managedaccount

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEscrower struct {
	mu          sync.Mutex
	calls       int
	password    string
	clientError string
	err         error
}

func (m *mockEscrower) SendManagedLocalAccountPassword(password, clientError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.password = password
	m.clientError = clientError
	return m.err
}

func (m *mockEscrower) snapshot() (calls int, password, clientError string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls, m.password, m.clientError
}

// newTestReceiver returns a receiver whose provisioning is stubbed. Tests wait on the channel
// attempt() returns, which closes only after the single-flight lock is released.
func newTestReceiver(escrower Escrower, provision provisionFunc) *Receiver {
	return &Receiver{escrower: escrower, provision: provision}
}

// awaitAttempt starts an attempt and waits for it to finish, failing rather than hanging if the
// attempt was dropped or never completes.
func awaitAttempt(t *testing.T, r *Receiver) {
	t.Helper()
	done := r.attempt()
	require.NotNil(t, done, "attempt was dropped")
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("provisioning attempt did not finish")
	}
}

func notification(enabled bool) *fleet.OrbitConfig {
	return &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{CreateWindowsManagedLocalAccount: enabled},
	}
}

func TestReceiverRun(t *testing.T) {
	// Run must gate on the notification, and must start provisioning when it is set. Everything below
	// exercises attempt() directly so it can wait on completion.
	t.Run("Run gates on the notification", func(t *testing.T) {
		esc := &mockEscrower{}
		provisioned := make(chan struct{})
		r := newTestReceiver(esc, func(string, string) error {
			close(provisioned)
			return nil
		})

		// A receiver loop that hands over no config at all must not take the process down with it.
		require.NoError(t, r.Run(nil))
		require.NoError(t, r.Run(notification(false)))
		select {
		case <-provisioned:
			t.Fatal("provisioning ran without the notification")
		case <-time.After(100 * time.Millisecond):
		}

		require.NoError(t, r.Run(notification(true)))
		select {
		case <-provisioned:
		case <-time.After(10 * time.Second):
			t.Fatal("the notification did not start provisioning")
		}
	})

	t.Run("provisions and escrows the password it generated", func(t *testing.T) {
		esc := &mockEscrower{}
		var gotUser, gotPassword string
		r := newTestReceiver(esc, func(username, password string) error {
			gotUser, gotPassword = username, password
			return nil
		})

		awaitAttempt(t, r)

		assert.Equal(t, fleet.ManagedLocalAccountUsername, gotUser)
		calls, escrowed, clientError := esc.snapshot()
		assert.Equal(t, 1, calls)
		assert.Empty(t, clientError)
		assert.NotEmpty(t, gotPassword)
		assert.Equal(t, gotPassword, escrowed)
	})

	// A creation failure is reported rather than swallowed, so it surfaces on the host and the server
	// keeps asking. No password is escrowed, because none was successfully set.
	t.Run("reports a provisioning failure as a client error", func(t *testing.T) {
		esc := &mockEscrower{}
		r := newTestReceiver(esc, func(string, string) error {
			return errors.New("NetUserAdd failed: access denied")
		})

		awaitAttempt(t, r)

		calls, password, clientError := esc.snapshot()
		assert.Equal(t, 1, calls)
		assert.Empty(t, password)
		assert.Contains(t, clientError, "NetUserAdd failed: access denied")
	})

	// The account exists with a password Fleet does not know. Nothing local records success, so the flow
	// re-runs and resets the password once the retry window has passed. A failed escrow arms the same
	// throttle as a failed creation, since neither got a password safely to the server.
	t.Run("a failed escrow leaves nothing that would block a retry", func(t *testing.T) {
		esc := &mockEscrower{err: errors.New("server unavailable")}
		var provisions int
		provision := func(string, string) error {
			provisions++
			return nil
		}

		r := newTestReceiver(esc, provision)
		// No throttle, so pacing cannot mask the retry this is looking for.
		r.retryFrequency = 0
		awaitAttempt(t, r)
		awaitAttempt(t, r)

		assert.Equal(t, 2, provisions, "the second notification must re-run provisioning")
	})

	t.Run("a panic while provisioning does not escape the goroutine", func(t *testing.T) {
		esc := &mockEscrower{}
		r := newTestReceiver(esc, func(string, string) error {
			panic("simulated syscall failure")
		})

		awaitAttempt(t, r)

		calls, _, _ := esc.snapshot()
		assert.Zero(t, calls, "nothing should be escrowed when provisioning panics")

		// The lock was released, so a later notification is still acted on.
		var provisioned bool
		r.provision = func(string, string) error {
			provisioned = true
			return nil
		}
		awaitAttempt(t, r)
		assert.True(t, provisioned, "a panic must not wedge the single-flight lock")
	})

	// The server re-sends the notification every config fetch, so a host that cannot provision would
	// otherwise redo the syscalls and re-post its error every 30 seconds forever.
	t.Run("a failed attempt is not retried until the retry frequency elapses", func(t *testing.T) {
		esc := &mockEscrower{}
		var provisions int
		r := newTestReceiver(esc, func(string, string) error {
			provisions++
			return errors.New("policy rejected the password")
		})
		r.retryFrequency = time.Hour

		awaitAttempt(t, r)
		assert.Equal(t, 1, provisions)

		// A notification arriving right after the failure is dropped rather than redoing the work.
		assert.Nil(t, r.attempt(), "a retry inside the frequency window must be dropped")
		assert.Equal(t, 1, provisions)

		// Once the window has passed, the host tries again.
		r.lastFailure = time.Now().Add(-2 * time.Hour)
		awaitAttempt(t, r)
		assert.Equal(t, 2, provisions)
	})

	t.Run("a success clears the retry throttle", func(t *testing.T) {
		r := newTestReceiver(&mockEscrower{}, func(string, string) error { return nil })
		r.retryFrequency = time.Hour

		awaitAttempt(t, r)
		// Make sure the 2nd back-to-back attempt is not dropped after a success.
		done := r.attempt()
		require.NotNil(t, done, "a success must not arm the retry throttle")
		<-done
	})

	t.Run("only one attempt runs at a time", func(t *testing.T) {
		esc := &mockEscrower{}
		started := make(chan struct{})
		release := make(chan struct{})

		r := newTestReceiver(esc, func(string, string) error {
			close(started)
			<-release
			return nil
		})

		done := r.attempt()
		require.NotNil(t, done)
		<-started

		// A second attempt while the first is in flight is dropped, not queued.
		assert.Nil(t, r.attempt(), "a concurrent attempt must be dropped")

		close(release)
		<-done

		calls, _, _ := esc.snapshot()
		assert.Equal(t, 1, calls, "the dropped attempt must not escrow a second password")
	})
}
