package managedaccount

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

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

// newTestReceiver returns a receiver whose provisioning is stubbed and whose attempts can be waited
// on, so the background goroutine does not make the tests racy.
func newTestReceiver(t *testing.T, escrower Escrower, provision provisionFunc) (*Receiver, chan struct{}) {
	t.Helper()
	done := make(chan struct{})
	return &Receiver{escrower: escrower, provision: provision, done: done}, done
}

func notification(enabled bool) *fleet.OrbitConfig {
	return &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{CreateWindowsManagedLocalAccount: enabled},
	}
}

func TestReceiverRun(t *testing.T) {
	t.Run("does nothing without the notification", func(t *testing.T) {
		esc := &mockEscrower{}
		var provisioned bool
		r, done := newTestReceiver(t, esc, func(string, string) error {
			provisioned = true
			return nil
		})

		require.NoError(t, r.Run(notification(false)))

		select {
		case <-done:
			t.Fatal("provisioning ran without the notification")
		case <-time.After(100 * time.Millisecond):
		}
		assert.False(t, provisioned)
		calls, _, _ := esc.snapshot()
		assert.Zero(t, calls)
	})

	t.Run("provisions and escrows the password it generated", func(t *testing.T) {
		esc := &mockEscrower{}
		var gotUser, gotPassword string
		r, done := newTestReceiver(t, esc, func(username, password string) error {
			gotUser, gotPassword = username, password
			return nil
		})

		require.NoError(t, r.Run(notification(true)))
		<-done

		assert.Equal(t, fleet.ManagedLocalAccountUsername, gotUser)
		calls, escrowed, clientError := esc.snapshot()
		assert.Equal(t, 1, calls)
		assert.Empty(t, clientError)
		// The escrowed password must be the one actually set on the account, or the admin is handed a
		// password that does not work.
		assert.Equal(t, gotPassword, escrowed)
	})

	// A creation failure is reported rather than swallowed, so it surfaces on the host and the server
	// keeps asking. No password is escrowed, because none was successfully set.
	t.Run("reports a provisioning failure as a client error", func(t *testing.T) {
		esc := &mockEscrower{}
		r, done := newTestReceiver(t, esc, func(string, string) error {
			return errors.New("NetUserAdd failed: access denied")
		})

		require.NoError(t, r.Run(notification(true)))
		<-done

		calls, password, clientError := esc.snapshot()
		assert.Equal(t, 1, calls)
		assert.Empty(t, password)
		assert.Contains(t, clientError, "NetUserAdd failed: access denied")
	})

	// The account exists with a password Fleet does not know. Nothing local records success, so the
	// next notification re-runs the flow, resets the password, and escrows the new one.
	t.Run("a failed escrow leaves nothing that would block a retry", func(t *testing.T) {
		esc := &mockEscrower{err: errors.New("server unavailable")}
		var provisions int
		provision := func(string, string) error {
			provisions++
			return nil
		}

		r, done := newTestReceiver(t, esc, provision)
		require.NoError(t, r.Run(notification(true)))
		<-done

		r.done = make(chan struct{})
		require.NoError(t, r.Run(notification(true)))
		<-r.done

		assert.Equal(t, 2, provisions, "the second notification must re-run provisioning")
	})

	t.Run("only one attempt runs at a time", func(t *testing.T) {
		esc := &mockEscrower{}
		release := make(chan struct{})
		var concurrent, maxConcurrent int
		var mu sync.Mutex

		r, done := newTestReceiver(t, esc, func(string, string) error {
			mu.Lock()
			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()
			<-release
			mu.Lock()
			concurrent--
			mu.Unlock()
			return nil
		})

		require.NoError(t, r.Run(notification(true)))
		// Second notification while the first attempt is still in flight; it must be dropped.
		require.NoError(t, r.Run(notification(true)))
		close(release)
		<-done

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, maxConcurrent)
		calls, _, _ := esc.snapshot()
		assert.Equal(t, 1, calls, "the dropped notification must not escrow a second password")
	})
}

func TestGeneratePassword(t *testing.T) {
	seen := make(map[string]struct{})
	for range 100 {
		pw, err := generatePassword()
		require.NoError(t, err)

		assert.Equal(t, passwordLength, utf8.RuneCountInString(pw))
		assert.True(t, strings.ContainsAny(pw, lowerChars), "no lowercase in %q", pw)
		assert.True(t, strings.ContainsAny(pw, upperChars), "no uppercase in %q", pw)
		assert.True(t, strings.ContainsAny(pw, digitChars), "no digit in %q", pw)
		assert.True(t, strings.ContainsAny(pw, symbolChars), "no symbol in %q", pw)

		_, duplicate := seen[pw]
		assert.False(t, duplicate, "generated a duplicate password")
		seen[pw] = struct{}{}
	}
}
