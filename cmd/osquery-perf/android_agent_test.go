package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
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

// fakeFleet stands in for Fleet's PubSub push endpoint, capturing and decoding the
// AMAPI device payloads the agent sends.
type fakeFleet struct {
	mu       sync.Mutex
	messages []capturedMessage
}

type capturedMessage struct {
	notificationType string
	// raw is the decoded payload before unmarshalling, so a test can assert on
	// what is actually on the wire and not just on what unmarshals back.
	raw    []byte
	device androidmanagement.Device
}

func (f *fakeFleet) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/fleet/android_enterprise/pubsub", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message android.PubSubMessage `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		raw, err := base64.StdEncoding.DecodeString(body.Message.Data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var device androidmanagement.Device
		if err := json.Unmarshal(raw, &device); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.messages = append(f.messages, capturedMessage{
			notificationType: body.Message.Attributes["notificationType"],
			raw:              raw,
			device:           device,
		})
		f.mu.Unlock()
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func (f *fakeFleet) captured() []capturedMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]capturedMessage(nil), f.messages...)
}

// newVitalsTestAgent builds a fully generated agent pointed at fleetAddress, using
// the real constructor so the generated vitals are the ones a load test would send.
func newVitalsTestAgent(fleetAddress string) *androidAgent {
	return newAndroidAgent(1, fleetAddress, "secret", "token", "http://proxy.invalid", "LC01", time.Minute, 5, 0, defaultVitalsChurnProb, nil)
}

// newFullVitalsTestAgent is newVitalsTestAgent pinned to the case where every
// section is collected and nothing is re-rolled, so a test can assert on an exact
// payload instead of on whichever sections this device happened to draw.
func newFullVitalsTestAgent(fleetAddress string) *androidAgent {
	agent := newVitalsTestAgent(fleetAddress)
	agent.reportsDeviceSettings = true
	agent.reportsSecurityPosture = true
	agent.vitalsChurnProb = 0
	return agent
}

// The AMAPI enum members a simulated device is allowed to report. These are
// deliberately not the full documented sets: the *_UNSPECIFIED sentinels (and
// UPDATE_STATUS_UNKNOWN) are omitted because Fleet's reportedEnum blanks them
// out, so a device reporting one would be reporting nothing. Do not add them
// back — the point is that a value outside these sets is either garbage Fleet
// would store verbatim or a value it would silently drop.
//
// The two SIM sets keep their *_UNSPECIFIED members: a physical SIM really does
// report them, and Fleet dropping them is the behavior being exercised.
var (
	validEncryptionStatuses = []string{"UNSUPPORTED", "INACTIVE", "ACTIVATING", "ACTIVE", "ACTIVE_DEFAULT_KEY", "ACTIVE_PER_USER"}
	validUpdateStatuses     = []string{"UP_TO_DATE", "UNKNOWN_UPDATE_AVAILABLE", "SECURITY_UPDATE_AVAILABLE", "OS_UPDATE_AVAILABLE"}
	validPostures           = []string{"SECURE", "AT_RISK", "POTENTIALLY_COMPROMISED"}
	validSecurityRisks      = []string{"UNKNOWN_OS", "COMPROMISED_OS", "HARDWARE_BACKED_EVALUATION_FAILED"}
	validActivationStates   = []string{"ACTIVATION_STATE_UNSPECIFIED", "ACTIVATED", "NOT_ACTIVATED"}
	validConfigModes        = []string{"CONFIG_MODE_UNSPECIFIED", "ADMIN_CONFIGURED", "USER_CONFIGURED"}
)

// assertVitalsReported checks that a device payload carries every section Fleet
// reads host vitals from. A missing section is not an error Fleet can report — it
// silently stores NULL — so the load test has to assert it here.
func assertVitalsReported(t *testing.T, agent *androidAgent, device androidmanagement.Device) {
	t.Helper()

	assert.Equal(t, agent.apiLevel, device.ApiLevel, "api_level")
	assert.Positive(t, device.ApiLevel, "an unreported api level stores as NULL")

	require.NotNil(t, device.HardwareInfo, "manufacturer lives on hardwareInfo")
	assert.Equal(t, agent.manufacturer, device.HardwareInfo.Manufacturer)
	assert.NotEmpty(t, device.HardwareInfo.Manufacturer)

	require.NotNil(t, device.SoftwareInfo, "kernel, bootloader and patch level live on softwareInfo")
	assert.Equal(t, agent.securityPatchLevel, device.SoftwareInfo.SecurityPatchLevel)
	assert.Equal(t, agent.deviceKernelVersion, device.SoftwareInfo.DeviceKernelVersion)
	assert.Equal(t, agent.bootloaderVersion, device.SoftwareInfo.BootloaderVersion)
	assert.NotEmpty(t, device.SoftwareInfo.SecurityPatchLevel)
	assert.NotEmpty(t, device.SoftwareInfo.DeviceKernelVersion)
	assert.NotEmpty(t, device.SoftwareInfo.BootloaderVersion)
	require.NotNil(t, device.SoftwareInfo.SystemUpdateInfo, "system_update_status")
	assert.Contains(t, validUpdateStatuses, device.SoftwareInfo.SystemUpdateInfo.UpdateStatus)

	require.NotNil(t, device.DeviceSettings, "adb, passcode, Play Protect and encryption")
	assert.Equal(t, agent.adbEnabled, device.DeviceSettings.AdbEnabled)
	assert.Equal(t, agent.deviceSecure, device.DeviceSettings.IsDeviceSecure)
	assert.Equal(t, agent.verifyAppsEnabled, device.DeviceSettings.VerifyAppsEnabled)
	assert.Contains(t, validEncryptionStatuses, device.DeviceSettings.EncryptionStatus)

	require.NotNil(t, device.SecurityPosture, "security_posture")
	assert.Contains(t, validPostures, device.SecurityPosture.DevicePosture)

	require.NotNil(t, device.NetworkInfo, "telephony_infos")
	require.NotEmpty(t, device.NetworkInfo.TelephonyInfos)
	assert.Equal(t, "COMPANY_OWNED", device.Ownership, "Fleet drops telephony info for a personally owned device")
}

// TestStatusReportIncludesHostVitals is the load-test counterpart of Fleet's
// androidDeviceVitals extraction: a status report that omits any of these sections
// leaves the host vitals columns NULL, so a load test would never exercise them.
func TestStatusReportIncludesHostVitals(t *testing.T) {
	fleetPubSub := &fakeFleet{}
	srv := fleetPubSub.server(t)
	agent := newFullVitalsTestAgent(srv.URL)

	state := testState()
	require.NoError(t, agent.sendStatusReport(&state))

	messages := fleetPubSub.captured()
	require.Len(t, messages, 1)
	assert.Equal(t, string(android.PubSubStatusReport), messages[0].notificationType)

	device := messages[0].device
	assertVitalsReported(t, agent, device)

	// The status-report-only fields must survive the shared payload refactor.
	assert.Equal(t, "ACTIVE", device.AppliedState)
	assert.Equal(t, state.PolicyVersion, device.AppliedPolicyVersion)
	assert.Equal(t, state.PolicyName, device.AppliedPolicyName)
	assert.NotEmpty(t, device.LastStatusReportTime)
	assert.NotEmpty(t, device.LastPolicySyncTime)
	assert.Len(t, device.ApplicationReports, 5)
	assert.Contains(t, device.EnrollmentTokenData, "secret")
}

// TestEnrollmentIncludesHostVitals covers the other write path: Fleet also stores
// vitals when it first creates the host from an ENROLLMENT message, and the two
// payloads must agree or a host's vitals would change on its first status report.
func TestEnrollmentIncludesHostVitals(t *testing.T) {
	fleetPubSub := &fakeFleet{}
	srv := fleetPubSub.server(t)
	agent := newFullVitalsTestAgent(srv.URL)

	require.NoError(t, agent.sendEnrollment())
	state := testState()
	require.NoError(t, agent.sendStatusReport(&state))

	messages := fleetPubSub.captured()
	require.Len(t, messages, 2)
	assert.Equal(t, string(android.PubSubEnrollment), messages[0].notificationType)

	enrolled, reported := messages[0].device, messages[1].device
	assertVitalsReported(t, agent, enrolled)
	assert.Contains(t, enrolled.EnrollmentTokenData, "secret", "the enroll secret is how Fleet finds the team")

	// The agent is pinned to no churn, so even the volatile vitals must match
	// here: what a message reports is the device's state at that moment, and
	// nothing changed between these two.
	assert.Equal(t, enrolled.ApiLevel, reported.ApiLevel)
	assert.Equal(t, enrolled.HardwareInfo, reported.HardwareInfo)
	assert.Equal(t, enrolled.SoftwareInfo, reported.SoftwareInfo)
	assert.Equal(t, enrolled.DeviceSettings, reported.DeviceSettings)
	assert.Equal(t, enrolled.SecurityPosture, reported.SecurityPosture)
	assert.Equal(t, enrolled.NetworkInfo, reported.NetworkInfo)
}

// TestStatusReportsChurnVolatileVitals is the point of splitting the vitals in
// two. Fleet overwrites every vitals column on every status report, but MySQL
// changes no row and advances no updated_at when the values are identical, so a
// device whose vitals never move makes the load test measure none of the write
// cost of this table. The device's identity must still hold still.
func TestStatusReportsChurnVolatileVitals(t *testing.T) {
	fleetPubSub := &fakeFleet{}
	srv := fleetPubSub.server(t)
	agent := newVitalsTestAgent(srv.URL)
	agent.vitalsChurnProb = 1 // re-roll on every report

	state := testState()
	// Enough reports that a value failing to move is a real failure and not a run
	// of luck: the least likely of the assertions below (security posture never
	// leaving SECURE) has a false-failure probability around 1e-10 at this count,
	// against roughly 1 in 7,000 at 60 reports.
	const reports = 150
	for range reports {
		require.NoError(t, agent.sendStatusReport(&state))
	}

	messages := fleetPubSub.captured()
	require.Len(t, messages, reports)

	var (
		settings       = map[string]int{}
		postures       = map[string]int{}
		updates        = map[string]int{}
		simConfigModes = map[string]int{}
		hasESIM        bool
		sectionsGone   int
	)
	first := messages[0].device
	for _, msg := range messages {
		device := msg.device

		// Device identity must not move.
		assert.Equal(t, first.ApiLevel, device.ApiLevel)
		require.NotNil(t, device.HardwareInfo)
		assert.Equal(t, first.HardwareInfo.Manufacturer, device.HardwareInfo.Manufacturer)
		assert.Equal(t, first.HardwareInfo.SerialNumber, device.HardwareInfo.SerialNumber)
		require.NotNil(t, device.SoftwareInfo)
		assert.Equal(t, first.SoftwareInfo.SecurityPatchLevel, device.SoftwareInfo.SecurityPatchLevel)
		assert.Equal(t, first.SoftwareInfo.DeviceKernelVersion, device.SoftwareInfo.DeviceKernelVersion)
		assert.Equal(t, first.SoftwareInfo.BootloaderVersion, device.SoftwareInfo.BootloaderVersion)

		// The SIM cards are hardware; only their eSIM activation moves.
		require.NotNil(t, device.NetworkInfo)
		require.Len(t, device.NetworkInfo.TelephonyInfos, len(first.NetworkInfo.TelephonyInfos))
		for i, sim := range device.NetworkInfo.TelephonyInfos {
			assert.Equal(t, first.NetworkInfo.TelephonyInfos[i].IccId, sim.IccId)
			assert.Equal(t, first.NetworkInfo.TelephonyInfos[i].PhoneNumber, sim.PhoneNumber)
			assert.Equal(t, first.NetworkInfo.TelephonyInfos[i].CarrierName, sim.CarrierName)
			assert.Contains(t, validActivationStates, sim.ActivationState)
			assert.Contains(t, validConfigModes, sim.ConfigMode)
			// A physical SIM reports neither eSIM enum, on this report or any
			// other, so only an eSIM's can churn.
			if sim.ActivationState != activationStateUnspecified {
				hasESIM = true
				simConfigModes[sim.ConfigMode]++
			} else {
				assert.Equal(t, configModeUnspecified, sim.ConfigMode, "a physical SIM must stay physical")
			}
		}

		require.NotNil(t, device.SoftwareInfo.SystemUpdateInfo)
		assert.Contains(t, validUpdateStatuses, device.SoftwareInfo.SystemUpdateInfo.UpdateStatus)
		updates[device.SoftwareInfo.SystemUpdateInfo.UpdateStatus]++

		if device.DeviceSettings == nil || device.SecurityPosture == nil {
			sectionsGone++
		}
		if ds := device.DeviceSettings; ds != nil {
			assert.Contains(t, validEncryptionStatuses, ds.EncryptionStatus)
			settings[fmt.Sprintf("%t/%t/%t/%s", ds.AdbEnabled, ds.IsDeviceSecure, ds.VerifyAppsEnabled, ds.EncryptionStatus)]++
		}
		if sp := device.SecurityPosture; sp != nil {
			assert.Contains(t, validPostures, sp.DevicePosture)
			// A device that became secure again must drop the details it used to
			// report, or Fleet would keep a stale risk on a secure host.
			if sp.DevicePosture == "SECURE" {
				assert.Empty(t, sp.PostureDetails)
			} else {
				assert.NotEmpty(t, sp.PostureDetails)
			}
			postures[sp.DevicePosture]++
		}
	}

	// Over 60 re-rolls each of these must have taken more than one value, or the
	// vitals row would be written once and then never change again.
	assert.Greater(t, len(settings), 1, "device settings never changed across %d reports", reports)
	assert.Greater(t, len(postures), 1, "security posture never changed across %d reports", reports)
	assert.Greater(t, len(updates), 1, "system update status never changed across %d reports", reports)
	assert.Positive(t, sectionsGone, "no report ever omitted a section, so Fleet never nulls a column")
	if hasESIM {
		assert.Greater(t, len(simConfigModes), 1, "eSIM config mode never changed across %d reports", reports)
	}
}

// TestUncollectedSectionsAreOmitted covers the other side of the vitals write: a
// policy that does not collect a section makes AMAPI omit it, and Fleet nulls the
// matching columns. A payload that always carries every section would never
// exercise that.
func TestUncollectedSectionsAreOmitted(t *testing.T) {
	fleetPubSub := &fakeFleet{}
	srv := fleetPubSub.server(t)
	agent := newFullVitalsTestAgent(srv.URL)
	agent.reportsDeviceSettings = false
	agent.reportsSecurityPosture = false

	require.NoError(t, agent.sendEnrollment())
	state := testState()
	require.NoError(t, agent.sendStatusReport(&state))

	messages := fleetPubSub.captured()
	require.Len(t, messages, 2)
	for _, msg := range messages {
		assert.Nil(t, msg.device.DeviceSettings, "an uncollected section must be absent, not empty")
		assert.Nil(t, msg.device.SecurityPosture, "an uncollected section must be absent, not empty")
		assert.NotContains(t, string(msg.raw), "deviceSettings")
		assert.NotContains(t, string(msg.raw), "securityPosture")

		// The sections the policy still collects must be unaffected.
		require.NotNil(t, msg.device.HardwareInfo)
		assert.NotEmpty(t, msg.device.HardwareInfo.Manufacturer)
		require.NotNil(t, msg.device.NetworkInfo)
		assert.NotEmpty(t, msg.device.NetworkInfo.TelephonyInfos)
	}
}

// TestDeviceSettingsBooleansAlwaysOnTheWire guards the ForceSendFields on
// DeviceSettings. Without them Go's omitempty drops every false boolean, so a
// device with ADB off, no passcode and Play Protect off would send
// "deviceSettings":{} — which unmarshals the same way for Fleet, but no longer
// resembles what AMAPI actually pushes.
func TestDeviceSettingsBooleansAlwaysOnTheWire(t *testing.T) {
	fleetPubSub := &fakeFleet{}
	srv := fleetPubSub.server(t)
	agent := newFullVitalsTestAgent(srv.URL)
	agent.adbEnabled = false
	agent.deviceSecure = false
	agent.verifyAppsEnabled = false

	require.NoError(t, agent.sendEnrollment())

	messages := fleetPubSub.captured()
	require.Len(t, messages, 1)
	payload := string(messages[0].raw)
	assert.Contains(t, payload, `"adbEnabled":false`)
	assert.Contains(t, payload, `"isDeviceSecure":false`)
	assert.Contains(t, payload, `"verifyAppsEnabled":false`)

	// And Fleet still reads them back as the false values they are.
	require.NotNil(t, messages[0].device.DeviceSettings)
	assert.False(t, messages[0].device.DeviceSettings.AdbEnabled)
	assert.False(t, messages[0].device.DeviceSettings.IsDeviceSecure)
	assert.False(t, messages[0].device.DeviceSettings.VerifyAppsEnabled)
}

// TestGenerateDeviceVitalsAcrossFleet checks the generated values across many
// devices: each one has to be internally consistent, and the fleet as a whole has
// to cover the branches Fleet's extraction treats differently — a secure device
// vs. one with posture details, and an eSIM vs. a physical SIM whose
// *_UNSPECIFIED enums Fleet blanks out.
func TestGenerateDeviceVitalsAcrossFleet(t *testing.T) {
	const deviceCount = 400

	var (
		postures       = map[string]int{}
		updateStatuses = map[string]int{}
		activations    = map[string]int{}
		configModes    = map[string]int{}
		dualSIM        int
		adbEnabled     int
		insecure       int
		unencrypted    int
		noSettings     int
		noPosture      int
	)

	for i := 0; i < deviceCount; i++ {
		agent := newVitalsTestAgent("http://fleet.invalid")

		assert.Equal(t, androidManufacturers[agent.brand], agent.manufacturer)
		assert.NotEmpty(t, agent.manufacturer, "brand %q has no manufacturer mapping", agent.brand)
		assert.Equal(t, androidAPILevels[agent.androidVersion], agent.apiLevel)
		assert.Positive(t, agent.apiLevel, "Android %q has no API level mapping", agent.androidVersion)

		assert.Contains(t, androidSecurityPatchLevels, agent.securityPatchLevel)
		assert.Contains(t, agent.deviceKernelVersion, "-android"+agent.androidVersion+"-",
			"the kernel must name the Android release it ships on")
		kernelBase := androidKernelBases[agent.androidVersion]
		require.NotEmpty(t, kernelBase, "Android %q has no kernel line mapping", agent.androidVersion)
		assert.True(t, strings.HasPrefix(agent.deviceKernelVersion, kernelBase+"."),
			"kernel %q does not start with the %q line", agent.deviceKernelVersion, kernelBase)
		assert.Contains(t, agent.bootloaderVersion, agent.hardware)

		assert.Contains(t, validEncryptionStatuses, agent.encryptionStatus)
		assert.Contains(t, validUpdateStatuses, agent.systemUpdateStatus)
		assert.Contains(t, validPostures, agent.securityPosture)

		// AMAPI only attaches posture details to a device that is not secure.
		if agent.securityPosture == "SECURE" {
			assert.Empty(t, agent.postureDetails, "a secure device reports no posture details")
		} else {
			require.NotEmpty(t, agent.postureDetails, "posture %q must explain itself", agent.securityPosture)
			for _, detail := range agent.postureDetails {
				require.NotNil(t, detail)
				assert.Contains(t, validSecurityRisks, detail.SecurityRisk)
				require.NotEmpty(t, detail.Advice)
				for _, advice := range detail.Advice {
					require.NotNil(t, advice)
					assert.NotEmpty(t, advice.DefaultMessage, "Fleet only keeps the default message")
				}
			}
		}

		require.NotEmpty(t, agent.telephonyInfos)
		require.LessOrEqual(t, len(agent.telephonyInfos), 2, "no simulated device has more than two SIMs")
		for _, sim := range agent.telephonyInfos {
			require.NotNil(t, sim)
			assert.Regexp(t, `^89[0-9]{17}$`, sim.IccId, "an ICCID is 19 digits starting with 89")
			carrierIdx := slices.IndexFunc(androidCarriers, func(c androidCarrier) bool { return c.name == sim.CarrierName })
			require.GreaterOrEqual(t, carrierIdx, 0, "unknown carrier %q", sim.CarrierName)
			carrier := androidCarriers[carrierIdx]
			assert.Regexp(t, fmt.Sprintf(`^\%s[0-9]{%d}$`, carrier.dialCode, carrier.nationalLen), sim.PhoneNumber,
				"the number must match the country %s operates in", carrier.name)
			assert.Contains(t, validActivationStates, sim.ActivationState)
			assert.Contains(t, validConfigModes, sim.ConfigMode)
			// A physical SIM reports neither eSIM enum, and an eSIM reports both.
			if sim.ActivationState == "ACTIVATION_STATE_UNSPECIFIED" {
				assert.Equal(t, "CONFIG_MODE_UNSPECIFIED", sim.ConfigMode, "a physical SIM has no config mode")
			} else {
				assert.NotEqual(t, "CONFIG_MODE_UNSPECIFIED", sim.ConfigMode, "an eSIM reports a config mode")
			}
			// Every value is stored in a varchar(255) column.
			assert.LessOrEqual(t, len(sim.PhoneNumber), fleet.MDMAndroidDeviceVitalMaxLength)
			assert.LessOrEqual(t, len(sim.CarrierName), fleet.MDMAndroidDeviceVitalMaxLength)
			assert.LessOrEqual(t, len(sim.IccId), fleet.MDMAndroidDeviceVitalMaxLength)

			activations[sim.ActivationState]++
			configModes[sim.ConfigMode]++
		}

		// Nothing generated may exceed the vitals columns' width.
		for name, value := range map[string]string{
			"manufacturer":     agent.manufacturer,
			"patch level":      agent.securityPatchLevel,
			"kernel version":   agent.deviceKernelVersion,
			"bootloader":       agent.bootloaderVersion,
			"encryption":       agent.encryptionStatus,
			"update status":    agent.systemUpdateStatus,
			"security posture": agent.securityPosture,
		} {
			assert.LessOrEqual(t, len(value), fleet.MDMAndroidDeviceVitalMaxLength, "%s is too long for its column", name)
		}

		postures[agent.securityPosture]++
		updateStatuses[agent.systemUpdateStatus]++
		if len(agent.telephonyInfos) == 2 {
			dualSIM++
		}
		if agent.adbEnabled {
			adbEnabled++
		}
		if !agent.deviceSecure {
			insecure++
		}
		// The states that make Fleet store an encryption_type meaning "off".
		if agent.encryptionStatus == "INACTIVE" || agent.encryptionStatus == "UNSUPPORTED" {
			unencrypted++
		}
		if !agent.reportsDeviceSettings {
			noSettings++
		}
		if !agent.reportsSecurityPosture {
			noPosture++
		}
	}

	// The point of the weighting is a fleet that is mostly healthy but not
	// uniformly so; a fleet with only one value in any of these gives the vitals
	// UI and API nothing to distinguish. Each band is the weight's implied
	// binomial mean over deviceCount devices, ±5 standard deviations: loose
	// enough that a false failure is roughly one run in a million, tight enough
	// that changing a weight is caught rather than shrugged at.
	for _, band := range []struct {
		name      string
		got       int
		low, high int
	}{
		{"SECURE devices", postures["SECURE"], 304, 376},
		{"AT_RISK devices", postures["AT_RISK"], 10, 70},
		{"POTENTIALLY_COMPROMISED devices", postures["POTENTIALLY_COMPROMISED"], 4, 45},
		{"UP_TO_DATE devices", updateStatuses["UP_TO_DATE"], 234, 326},
		{"SECURITY_UPDATE_AVAILABLE devices", updateStatuses["SECURITY_UPDATE_AVAILABLE"], 24, 96},
		{"OS_UPDATE_AVAILABLE devices", updateStatuses["OS_UPDATE_AVAILABLE"], 10, 70},
		{"UNKNOWN_UPDATE_AVAILABLE devices", updateStatuses["UNKNOWN_UPDATE_AVAILABLE"], 4, 45},
		{"dual-SIM devices", dualSIM, 74, 166},
		{"devices with ADB on", adbEnabled, 4, 45},
		{"devices without a passcode", insecure, 4, 45},
		{"unencrypted devices", unencrypted, 5, 48},
		{"devices not collecting deviceSettings", noSettings, 10, 70},
		{"devices not collecting securityPosture", noPosture, 10, 70},
	} {
		assert.GreaterOrEqual(t, band.got, band.low, "%s: %d is below the expected band", band.name, band.got)
		assert.LessOrEqual(t, band.got, band.high, "%s: %d is above the expected band", band.name, band.got)
	}

	// Both SIM kinds must appear, since Fleet stores their enums differently.
	assert.Positive(t, activations["ACTIVATION_STATE_UNSPECIFIED"], "physical SIMs")
	assert.Positive(t, activations["ACTIVATED"], "activated eSIMs")
	assert.Positive(t, configModes["ADMIN_CONFIGURED"])
	assert.Positive(t, configModes["USER_CONFIGURED"])
}

// TestVitalsMapsAgree guards the three per-release / per-brand maps against
// drifting apart: newAndroidAgent draws its brands and Android versions from two
// of them, and generateDeviceVitals then indexes the third. A release present in
// one and missing from another would silently report an empty manufacturer or a
// kernel version like ".47-android16-3-gdeadbeef".
func TestVitalsMapsAgree(t *testing.T) {
	require.NotEmpty(t, androidAPILevels)
	require.NotEmpty(t, androidManufacturers)

	for version := range androidAPILevels {
		assert.Positive(t, androidAPILevels[version], "Android %q has no API level", version)
		assert.NotEmpty(t, androidKernelBases[version], "Android %q has no kernel line", version)
	}
	for version := range androidKernelBases {
		assert.Contains(t, androidAPILevels, version, "kernel line for unsimulated Android %q", version)
	}
	for brand, manufacturer := range androidManufacturers {
		assert.NotEmpty(t, manufacturer, "brand %q has no manufacturer", brand)
	}
}

// TestRecentSecurityPatchLevels checks the patch levels a device may report are
// recent bulletin dates, not the hardcoded set that would go stale.
func TestRecentSecurityPatchLevels(t *testing.T) {
	got := recentSecurityPatchLevels(time.Date(2026, 3, 17, 4, 5, 6, 0, time.UTC))
	assert.Equal(t, []string{"2025-12-01", "2026-01-01", "2026-02-01", "2026-03-01"}, got,
		"four bulletin dates ending at the current month, newest last")

	// The package-level set the agent actually draws from must be usable too.
	require.NotEmpty(t, androidSecurityPatchLevels)
	for _, level := range androidSecurityPatchLevels {
		assert.Regexp(t, `^[0-9]{4}-[0-9]{2}-01$`, level)
	}
}

// TestWeightedChoice covers the picker the vitals distributions rely on.
func TestWeightedChoice(t *testing.T) {
	t.Run("a certain value is always picked", func(t *testing.T) {
		for range 20 {
			assert.Equal(t, "only", weightedChoice([]weightedValue{{"only", 1}}))
		}
	})

	t.Run("a zero-weight value is never picked", func(t *testing.T) {
		for range 100 {
			assert.Equal(t, "always", weightedChoice([]weightedValue{{"never", 0}, {"always", 1}}))
		}
	})

	// The rounding fallback must not resurrect a zero-weight value just because
	// it happens to be last.
	t.Run("a zero-weight value is never picked in last position", func(t *testing.T) {
		for range 100 {
			assert.Equal(t, "always", weightedChoice([]weightedValue{{"always", 0.9}, {"never", 0}}))
		}
	})

	// rand.Float64 can return a value the cumulative weights don't reach when they
	// sum to slightly under 1; that must still yield a real value, not "".
	t.Run("a short weight sum falls back to the last value", func(t *testing.T) {
		for range 100 {
			got := weightedChoice([]weightedValue{{"first", 0.1}, {"last", 0.1}})
			assert.Contains(t, []string{"first", "last"}, got)
		}
	})

	t.Run("every value is reachable", func(t *testing.T) {
		seen := map[string]bool{}
		for range 1000 {
			seen[weightedChoice([]weightedValue{{"a", 0.4}, {"b", 0.4}, {"c", 0.2}})] = true
		}
		assert.Equal(t, map[string]bool{"a": true, "b": true, "c": true}, seen)
	})
}
