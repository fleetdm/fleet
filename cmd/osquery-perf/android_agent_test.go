package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProxy is a stand-in for the android-amapi-mock coordination API.
type fakeProxy struct {
	mu sync.Mutex
	// registered is false until a device registers, mimicking a proxy that lost its
	// in-memory state.
	registered bool
	// state is served on a successful poll.
	state proxyDeviceState
	// pollStatus, when non-zero, overrides the status code of a poll for a registered device.
	pollStatus int
	// registerStatus, when non-zero, is the status code returned to a registration.
	registerStatus int

	registrations []registeredDevice
	pollCount     int
}

type registeredDevice struct {
	EnterpriseSpecificID string `json:"enterprise_specific_id"`
	DeviceName           string `json:"device_name"`
	EnterpriseID         string `json:"enterprise_id"`
	PolicyName           string `json:"policy_name"`
	PolicyVersion        int64  `json:"policy_version"`
}

func (p *fakeProxy) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /mock/devices/register", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var d registeredDevice
		if err := json.Unmarshal(body, &d); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		p.registrations = append(p.registrations, d)
		status := p.registerStatus
		if status == 0 {
			p.registered = true
			status = http.StatusOK
		}
		p.mu.Unlock()
		w.WriteHeader(status)
	})
	mux.HandleFunc("GET /mock/devices/{esid}/state", func(w http.ResponseWriter, _ *http.Request) {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.pollCount++
		if !p.registered {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		if p.pollStatus != 0 {
			http.Error(w, "simulated failure", p.pollStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.state)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func (p *fakeProxy) registrationsMade() []registeredDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]registeredDevice(nil), p.registrations...)
}

func (p *fakeProxy) set(mutate func(*fakeProxy)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	mutate(p)
}

func newTestAndroidAgent(proxyAddress string) *androidAgent {
	return &androidAgent{
		agentIndex:           1,
		proxyAddress:         proxyAddress,
		enterpriseSpecificID: "35B8F9A0-4C2E-4B1D-9F3A-7E6D5C4B3A21",
		enterpriseID:         "LC01",
		deviceName:           "enterprises/LC01/devices/fakedevice",
	}
}

func testState() proxyDeviceState {
	return proxyDeviceState{
		PolicyVersion: 12,
		PolicyName:    "enterprises/LC01/policies/host-uuid",
	}
}

// TestCurrentStateRecoversLostRegistration covers the load-testing bug where a proxy that
// lost its in-memory device state left the agent polling a 404 forever. The agent stopped
// sending status reports, so Fleet's hourly reconciler marked the host unenrolled and MDM
// status went to Off.
func TestCurrentStateRecoversLostRegistration(t *testing.T) {
	proxy := &fakeProxy{state: testState()}
	srv := proxy.server(t)
	agent := newTestAndroidAgent(srv.URL)

	// The proxy has no record of this device, so the first poll 404s.
	state, stale, err := agent.currentState()
	require.NoError(t, err, "the agent must recover by registering again")
	require.NotNil(t, state)
	assert.False(t, stale)
	assert.Equal(t, int64(12), state.PolicyVersion)
	assert.Equal(t, "enterprises/LC01/policies/host-uuid", state.PolicyName)

	registrations := proxy.registrationsMade()
	require.Len(t, registrations, 1, "exactly one re-registration")
	assert.Equal(t, agent.enterpriseSpecificID, registrations[0].EnterpriseSpecificID)
	assert.Equal(t, agent.deviceName, registrations[0].DeviceName)
	assert.Equal(t, agent.enterpriseID, registrations[0].EnterpriseID)

	// A later poll succeeds directly, without registering again.
	state, stale, err = agent.currentState()
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.False(t, stale)
	assert.Len(t, proxy.registrationsMade(), 1, "a healthy poll must not re-register")
}

// TestCurrentStateReRegistersWithLastKnownPolicy checks that recovery reports the policy the
// agent last observed, so the proxy doesn't tell Fleet the applied policy regressed.
func TestCurrentStateReRegistersWithLastKnownPolicy(t *testing.T) {
	proxy := &fakeProxy{
		registered: true,
		state:      proxyDeviceState{PolicyVersion: 57, PolicyName: "enterprises/LC01/policies/host-uuid"},
	}
	srv := proxy.server(t)
	agent := newTestAndroidAgent(srv.URL)

	// Establish a known state.
	_, _, err := agent.currentState()
	require.NoError(t, err)

	// The proxy restarts and forgets the device.
	proxy.set(func(p *fakeProxy) { p.registered = false })

	state, _, err := agent.currentState()
	require.NoError(t, err)
	require.NotNil(t, state)

	registrations := proxy.registrationsMade()
	require.Len(t, registrations, 1)
	assert.Equal(t, "enterprises/LC01/policies/host-uuid", registrations[0].PolicyName)
	assert.Equal(t, int64(57), registrations[0].PolicyVersion)
}

// TestCurrentStateStopsWhenDeviceDeleted covers the other half of the recovery: a device
// Fleet deleted through AMAPI (a real unenrollment) must NOT be registered again, or it would
// resurrect itself and re-enroll the host.
func TestCurrentStateStopsWhenDeviceDeleted(t *testing.T) {
	t.Run("a deleted device is terminal on poll", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, pollStatus: http.StatusGone, state: testState()}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		state, stale, err := agent.currentState()
		require.ErrorIs(t, err, errProxyDeviceDeleted)
		assert.Nil(t, state)
		assert.False(t, stale)
		assert.Empty(t, proxy.registrationsMade(), "a deleted device must not register again")
	})

	// Even with a usable cached state, a delete must not fall back to reporting.
	t.Run("a deleted device does not fall back to stale state", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, state: testState()}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		_, _, err := agent.currentState()
		require.NoError(t, err)
		require.NotNil(t, agent.lastState)

		proxy.set(func(p *fakeProxy) { p.pollStatus = http.StatusGone })

		state, _, err := agent.currentState()
		require.ErrorIs(t, err, errProxyDeviceDeleted)
		assert.Nil(t, state)
	})

	// If the proxy forgot the device AND refuses the registration as deleted, that is still
	// terminal — not a transient failure to paper over with stale state.
	t.Run("a registration refused as deleted is terminal", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, state: testState()}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		_, _, err := agent.currentState()
		require.NoError(t, err)

		proxy.set(func(p *fakeProxy) {
			p.registered = false
			p.registerStatus = http.StatusGone
		})

		state, _, err := agent.currentState()
		require.ErrorIs(t, err, errProxyDeviceDeleted)
		assert.Nil(t, state)
	})
}

func TestCurrentStateFallsBackToLastKnownState(t *testing.T) {
	t.Run("a transient failure reuses the last known state", func(t *testing.T) {
		state := testState()
		state.PendingCertificates = []uint{7}
		state.PendingCommands = []string{"enterprises/LC01/devices/fakedevice/operations/op1"}
		proxy := &fakeProxy{registered: true, state: state}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		_, _, err := agent.currentState()
		require.NoError(t, err)

		proxy.set(func(p *fakeProxy) { p.pollStatus = http.StatusInternalServerError })

		got, stale, err := agent.currentState()
		require.NoError(t, err, "the agent must keep reporting so it isn't reconciled as unenrolled")
		require.NotNil(t, got)
		assert.True(t, stale, "the caller must be able to count this as an error")
		assert.Equal(t, int64(12), got.PolicyVersion)
		assert.Empty(t, got.PendingCommands, "already-acked commands must not be acked again")
		assert.Empty(t, got.PendingCertificates, "already-handled certificates must not be re-driven")
	})

	t.Run("no state yet returns the error", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, pollStatus: http.StatusInternalServerError}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		state, stale, err := agent.currentState()
		require.Error(t, err)
		assert.Nil(t, state)
		assert.False(t, stale)
	})

	// The proxy forgot the device and registering fails for a transient reason: fall back
	// rather than going silent, and confirm the re-registration was actually attempted.
	t.Run("a failed re-registration falls back to stale state", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, state: testState()}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		_, _, err := agent.currentState()
		require.NoError(t, err)

		proxy.set(func(p *fakeProxy) {
			p.registered = false
			p.registerStatus = http.StatusInternalServerError
		})

		state, stale, err := agent.currentState()
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, stale)
		assert.Equal(t, int64(12), state.PolicyVersion)
		require.Len(t, proxy.registrationsMade(), 1, "the re-registration must have been attempted")
	})

	// Reusing stale state forever would keep a dead agent looking healthy.
	t.Run("stale state is not reused forever", func(t *testing.T) {
		proxy := &fakeProxy{registered: true, state: testState()}
		srv := proxy.server(t)
		agent := newTestAndroidAgent(srv.URL)

		_, _, err := agent.currentState()
		require.NoError(t, err)

		proxy.set(func(p *fakeProxy) { p.pollStatus = http.StatusInternalServerError })

		for i := range maxStaleStateReports {
			state, stale, err := agent.currentState()
			require.NoError(t, err, "fallback %d must still be allowed", i+1)
			require.NotNil(t, state)
			require.True(t, stale)
		}

		state, _, err := agent.currentState()
		require.Error(t, err, "the fallback must be bounded")
		assert.Nil(t, state)

		// A healthy poll clears the budget.
		proxy.set(func(p *fakeProxy) { p.pollStatus = 0 })
		_, stale, err := agent.currentState()
		require.NoError(t, err)
		require.False(t, stale)
		assert.Zero(t, agent.staleStateReports)
	})
}

// TestCurrentStateDoesNotMutateCachedState guards against the fallback clearing pending
// commands on the cached state itself, which would drop commands on a later healthy poll.
func TestCurrentStateDoesNotMutateCachedState(t *testing.T) {
	state := testState()
	state.PendingCommands = []string{"enterprises/LC01/devices/fakedevice/operations/op1"}
	state.PendingCertificates = []uint{7}
	proxy := &fakeProxy{registered: true, state: state}
	srv := proxy.server(t)
	agent := newTestAndroidAgent(srv.URL)

	got, _, err := agent.currentState()
	require.NoError(t, err)
	require.Len(t, got.PendingCommands, 1)

	proxy.set(func(p *fakeProxy) { p.pollStatus = http.StatusInternalServerError })

	_, _, err = agent.currentState()
	require.NoError(t, err)

	require.NotNil(t, agent.lastState)
	assert.Len(t, agent.lastState.PendingCommands, 1, "the cached state must be left intact")
	assert.Len(t, agent.lastState.PendingCertificates, 1, "the cached state must be left intact")
}

func TestPollProxyStateClassifiesFailures(t *testing.T) {
	proxy := &fakeProxy{}
	srv := proxy.server(t)
	agent := newTestAndroidAgent(srv.URL)

	// Unregistered: recoverable by registering again.
	_, err := agent.pollProxyState()
	require.ErrorIs(t, err, errProxyDeviceUnknown)

	// Deleted: terminal.
	proxy.set(func(p *fakeProxy) {
		p.registered = true
		p.pollStatus = http.StatusGone
	})
	_, err = agent.pollProxyState()
	require.ErrorIs(t, err, errProxyDeviceDeleted)

	// Anything else is just an error, and must not be mistaken for either.
	proxy.set(func(p *fakeProxy) { p.pollStatus = http.StatusTooManyRequests })
	_, err = agent.pollProxyState()
	require.Error(t, err)
	require.NotErrorIs(t, err, errProxyDeviceUnknown, "only a 404 means the registration was lost")
	require.NotErrorIs(t, err, errProxyDeviceDeleted, "only a 410 means the device was deleted")
	assert.Contains(t, err.Error(), "429", "error should carry the status code")
}
