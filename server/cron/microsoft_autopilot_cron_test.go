package cron

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGraphClient struct {
	devices []msgraph.WindowsAutopilotDevice
	listErr error
}

func (f *fakeGraphClient) VerifyCredential(context.Context) error { return nil }

func (f *fakeGraphClient) ListWindowsAutopilotDevices(context.Context) ([]msgraph.WindowsAutopilotDevice, error) {
	return f.devices, f.listErr
}

// autopilotSyncEnv wires a mock datastore whose Autopilot state lives in memory, so a test can assert what the sync
// wrote rather than how it phrased the SQL.
type autopilotSyncEnv struct {
	ds      *mock.Store
	stored  map[string]*fleet.HostAutopilotDevice // keyed by Autopilot device ID, like the real table
	nextID  uint
	removed []uint
	invalid map[string]bool
	// aggregateRefreshed counts recomputations of the app-config banner flag.
	aggregateRefreshed int
	syncResults        map[string]*string
}

func newAutopilotSyncEnv(t *testing.T, creds ...*fleet.MicrosoftGraphCredential) *autopilotSyncEnv {
	t.Helper()
	env := &autopilotSyncEnv{
		ds:          new(mock.Store),
		stored:      map[string]*fleet.HostAutopilotDevice{},
		nextID:      100,
		invalid:     map[string]bool{},
		syncResults: map[string]*string{},
	}

	// Reflect the stored flag back onto the credential the way the real datastore does. Returning a fixed fixture
	// would let a sync that never clears credential_invalid pass this suite.
	env.ds.ListMicrosoftGraphCredentialsFunc = func(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
		out := make([]*fleet.MicrosoftGraphCredential, 0, len(creds))
		for _, c := range creds {
			copied := *c
			copied.CredentialInvalid = env.invalid[c.TenantID]
			out = append(out, &copied)
		}
		return out, nil
	}
	env.ds.ListHostAutopilotDevicesFunc = func(ctx context.Context, tenantID string) ([]*fleet.HostAutopilotDevice, error) {
		out := []*fleet.HostAutopilotDevice{}
		for _, d := range env.stored {
			if d.TenantID == tenantID {
				copied := *d
				out = append(out, &copied)
			}
		}
		return out, nil
	}
	env.ds.IngestWindowsAutopilotDevicesFunc = func(ctx context.Context, devices []*fleet.HostAutopilotDevice) error {
		for _, d := range devices {
			copied := *d
			if existing, ok := env.stored[d.AutopilotDeviceID]; ok {
				copied.HostID = existing.HostID
			} else {
				env.nextID++
				copied.HostID = env.nextID
			}
			env.stored[d.AutopilotDeviceID] = &copied
		}
		return nil
	}
	env.ds.RemoveWindowsAutopilotHostsFunc = func(ctx context.Context, hostIDs []uint) error {
		env.removed = append(env.removed, hostIDs...)
		for _, id := range hostIDs {
			for deviceID, d := range env.stored {
				if d.HostID == id {
					delete(env.stored, deviceID)
				}
			}
		}
		return nil
	}
	env.ds.SetMicrosoftGraphCredentialInvalidFunc = func(ctx context.Context, tenantID string, invalid bool) (bool, error) {
		changed := env.invalid[tenantID] != invalid
		env.invalid[tenantID] = invalid
		return changed, nil
	}
	env.ds.RecordMicrosoftGraphSyncResultFunc = func(ctx context.Context, tenantID string, syncErr *string) error {
		env.syncResults[tenantID] = syncErr
		return nil
	}
	env.ds.UpdateMicrosoftGraphCredentialInvalidAggregateFunc = func(ctx context.Context) error {
		env.aggregateRefreshed++
		return nil
	}
	return env
}

func (e *autopilotSyncEnv) serials() []string {
	out := make([]string, 0, len(e.stored))
	for _, d := range e.stored {
		out = append(out, d.HardwareSerial)
	}
	return out
}

// deviceID returns the stored record for an Autopilot device ID, failing the test if the sync never wrote it.
func (e *autopilotSyncEnv) device(t *testing.T, autopilotDeviceID string) *fleet.HostAutopilotDevice {
	t.Helper()
	d, ok := e.stored[autopilotDeviceID]
	require.True(t, ok, "no stored record for autopilot device %s", autopilotDeviceID)
	return d
}

func device(id, serial, tag string) msgraph.WindowsAutopilotDevice {
	return msgraph.WindowsAutopilotDevice{ID: id, SerialNumber: serial, GroupTag: tag, EntraDeviceID: "aad-" + id}
}

func testCred(tenant string) *fleet.MicrosoftGraphCredential {
	return &fleet.MicrosoftGraphCredential{TenantID: tenant, ClientID: "client-" + tenant, ClientSecret: "secret"}
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func factoryFor(clients map[string]*fakeGraphClient) msgraph.ClientFactory {
	return func(cred *fleet.MicrosoftGraphCredential) (msgraph.Client, error) {
		if c, ok := clients[cred.TenantID]; ok {
			return c, nil
		}
		return &fakeGraphClient{}, nil
	}
}

func TestMicrosoftAutopilotSync(t *testing.T) {
	t.Parallel()
	const tenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"

	t.Run("no credential configured is a no-op", func(t *testing.T) {
		env := newAutopilotSyncEnv(t)
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(nil), discardLogger()))
		assert.False(t, env.ds.ListHostAutopilotDevicesFuncInvoked)
	})

	t.Run("creates a pending host per device and stores the group tag", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", "Engineering"),
			device("ap-2", "SERIAL-2", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.ElementsMatch(t, []string{"SERIAL-1", "SERIAL-2"}, env.serials())
		assert.Equal(t, "Engineering", env.device(t, "ap-1").GroupTag)
		require.Contains(t, env.syncResults, tenantA, "every pass records its outcome")
		assert.Nil(t, env.syncResults[tenantA], "a successful sync clears last_sync_error")
	})

	t.Run("an unchanged device is not rewritten", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", "Engineering"),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		env.ds.IngestWindowsAutopilotDevicesFuncInvoked = false
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		assert.False(t, env.ds.IngestWindowsAutopilotDevicesFuncInvoked,
			"re-syncing an identical registry must not ship every device over the wire again")
	})

	t.Run("a changed group tag updates in place", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", "Engineering"),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		originalID := env.device(t, "ap-1").HostID

		clients[tenantA].devices = []msgraph.WindowsAutopilotDevice{device("ap-1", "SERIAL-1", "Marketing")}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.Equal(t, "Marketing", env.device(t, "ap-1").GroupTag)
		assert.Equal(t, originalID, env.device(t, "ap-1").HostID, "the host is reused, not recreated")
	})

	t.Run("a device removed from autopilot removes its pending host", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", ""), device("ap-2", "SERIAL-2", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		goneID := env.device(t, "ap-2").HostID

		clients[tenantA].devices = []msgraph.WindowsAutopilotDevice{device("ap-1", "SERIAL-1", "")}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.Equal(t, []uint{goneID}, env.removed)
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
	})

	t.Run("an empty response never deletes existing pending hosts", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		clients[tenantA].devices = nil
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.Empty(t, env.removed, "zero devices is indistinguishable from a misconfiguration")
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
	})

	t.Run("two devices sharing a serial are two pending hosts", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "DUP-SERIAL", "Engineering"),
			device("ap-2", "DUP-SERIAL", "Marketing"),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.ElementsMatch(t, []string{"DUP-SERIAL", "DUP-SERIAL"}, env.serials(),
			"Windows serials are not unique, so both registrations are real devices")
		assert.NotEqual(t, env.device(t, "ap-1").HostID, env.device(t, "ap-2").HostID)

		// Retiring one of them must leave the other alone. Diffing on the serial would find the serial still present
		// and never remove anything, or remove both.
		clients[tenantA].devices = []msgraph.WindowsAutopilotDevice{device("ap-1", "DUP-SERIAL", "Engineering")}
		goneID := env.device(t, "ap-2").HostID
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.Equal(t, []uint{goneID}, env.removed)
		assert.ElementsMatch(t, []string{"DUP-SERIAL"}, env.serials())
	})

	t.Run("a device with no autopilot device id is skipped", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", ""),
			device("", "SERIAL-2", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials(),
			"the device id is the resolution key, so a device without one cannot be stored")
	})

	t.Run("a placeholder serial is skipped", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", ""),
			device("ap-2", "Default string", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials(),
			"a placeholder serial cannot identify a host before osquery runs")
	})

	t.Run("a pagination error aborts the tenant without deleting anything", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {devices: []msgraph.WindowsAutopilotDevice{
			device("ap-1", "SERIAL-1", ""),
		}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		clients[tenantA].listErr = errors.New("pagination stopped advancing")
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.Empty(t, env.removed)
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
		require.NotNil(t, env.syncResults[tenantA])
		assert.Contains(t, *env.syncResults[tenantA], "pagination stopped advancing")
	})
}

func TestMicrosoftAutopilotSyncCredentialFlag(t *testing.T) {
	t.Parallel()
	const tenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"

	// The banner is driven by a stored aggregate on the app config, so every flag change has to be followed by a
	// recomputation. Without it the credentials endpoint reports the failure correctly and the banner never appears,
	// with nothing else failing to indicate the problem.
	for _, tc := range []struct {
		name          string
		err           error
		wantInvalid   bool
		wantRefreshed bool
	}{
		{"graph 401 marks the credential invalid", &msgraph.Error{StatusCode: http.StatusUnauthorized}, true, true},
		{"graph 403 marks the credential invalid", &msgraph.Error{StatusCode: http.StatusForbidden}, true, true},
		{"graph 429 does not", &msgraph.Error{StatusCode: http.StatusTooManyRequests}, false, false},
		{"graph 500 does not", &msgraph.Error{StatusCode: http.StatusInternalServerError}, false, false},
		{"a non-graph error does not", errors.New("dial tcp: timeout"), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newAutopilotSyncEnv(t, testCred(tenantA))
			clients := map[string]*fakeGraphClient{tenantA: {listErr: tc.err}}
			require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

			assert.Equal(t, tc.wantInvalid, env.invalid[tenantA])
			if tc.wantRefreshed {
				assert.Equal(t, 1, env.aggregateRefreshed, "the banner aggregate must be recomputed when the flag changes")
			} else {
				assert.Zero(t, env.aggregateRefreshed, "a Microsoft outage must not raise a credential alarm")
			}
		})
	}

	t.Run("the flag and the banner clear on the next successful sync", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		clients := map[string]*fakeGraphClient{tenantA: {listErr: &msgraph.Error{StatusCode: http.StatusUnauthorized}}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))
		require.True(t, env.invalid[tenantA])

		clients[tenantA] = &fakeGraphClient{devices: []msgraph.WindowsAutopilotDevice{device("ap-1", "SERIAL-1", "")}}
		require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

		assert.False(t, env.invalid[tenantA])
		assert.Equal(t, 2, env.aggregateRefreshed, "clearing the flag must refresh the banner too")
	})
}

func TestMicrosoftAutopilotSyncIsolatesTenants(t *testing.T) {
	t.Parallel()
	const tenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
	const tenantB = "11111111-1111-1111-1111-111111111111"

	env := newAutopilotSyncEnv(t, testCred(tenantA), testCred(tenantB))
	clients := map[string]*fakeGraphClient{
		tenantA: {listErr: &msgraph.Error{StatusCode: http.StatusUnauthorized}},
		tenantB: {devices: []msgraph.WindowsAutopilotDevice{device("ap-b", "SERIAL-B", "Sales")}},
	}
	require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(clients), discardLogger()))

	assert.True(t, env.invalid[tenantA], "the failing tenant is flagged")
	assert.False(t, env.invalid[tenantB], "the healthy tenant is not")
	assert.ElementsMatch(t, []string{"SERIAL-B"}, env.serials(),
		"one tenant's failure must not stop another tenant's sync")
}
