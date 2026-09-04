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

const (
	tenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
	tenantB = "11111111-1111-1111-1111-111111111111"
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
	clients            map[string]*fakeGraphClient
}

func newAutopilotSyncEnv(t *testing.T, creds ...*fleet.MicrosoftGraphCredential) *autopilotSyncEnv {
	t.Helper()
	env := &autopilotSyncEnv{
		ds:          new(mock.Store),
		stored:      map[string]*fleet.HostAutopilotDevice{},
		nextID:      100,
		invalid:     map[string]bool{},
		syncResults: map[string]*string{},
		clients:     map[string]*fakeGraphClient{},
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
	env.ds.SetMicrosoftGraphCredentialInvalidFunc = func(ctx context.Context, tenantID string, invalid bool) error {
		env.invalid[tenantID] = invalid
		return nil
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

// graph sets what a tenant's Graph client returns on the next sync. Called again to change the registry between passes.
func (e *autopilotSyncEnv) graph(tenantID string, devices ...msgraph.WindowsAutopilotDevice) {
	e.clients[tenantID] = &fakeGraphClient{devices: devices}
}

// graphFails makes a tenant's Graph client return err instead of a device list.
func (e *autopilotSyncEnv) graphFails(tenantID string, err error) {
	e.clients[tenantID] = &fakeGraphClient{listErr: err}
}

// sync runs one full cron pass over every configured tenant.
func (e *autopilotSyncEnv) sync(t *testing.T) {
	t.Helper()
	require.NoError(t, cronMicrosoftAutopilotSync(t.Context(), e.ds, factoryFor(e.clients), discardLogger()))
}

func (e *autopilotSyncEnv) serials() []string {
	out := make([]string, 0, len(e.stored))
	for _, d := range e.stored {
		out = append(out, d.HardwareSerial)
	}
	return out
}

// device returns the stored record for an Autopilot device ID, failing the test if the sync never wrote it.
func (e *autopilotSyncEnv) device(t *testing.T, autopilotDeviceID string) *fleet.HostAutopilotDevice {
	t.Helper()
	d, ok := e.stored[autopilotDeviceID]
	require.True(t, ok, "no stored record for autopilot device %s", autopilotDeviceID)
	return d
}

// hostID is the host the sync resolved an Autopilot device onto.
func (e *autopilotSyncEnv) hostID(t *testing.T, autopilotDeviceID string) uint {
	t.Helper()
	return e.device(t, autopilotDeviceID).HostID
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

	t.Run("no credential configured is a no-op", func(t *testing.T) {
		env := newAutopilotSyncEnv(t)
		env.sync(t)
		assert.False(t, env.ds.ListHostAutopilotDevicesFuncInvoked)
	})

	t.Run("creates a pending host per device and stores the group tag", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", "Engineering"), device("ap-2", "SERIAL-2", ""))
		env.sync(t)

		assert.ElementsMatch(t, []string{"SERIAL-1", "SERIAL-2"}, env.serials())
		assert.Equal(t, "Engineering", env.device(t, "ap-1").GroupTag)
		require.Contains(t, env.syncResults, tenantA, "every pass records its outcome")
		assert.Nil(t, env.syncResults[tenantA], "a successful sync clears last_sync_error")
	})

	t.Run("an unchanged device is not rewritten", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", "Engineering"))
		env.sync(t)

		env.ds.IngestWindowsAutopilotDevicesFuncInvoked = false
		env.sync(t)
		assert.False(t, env.ds.IngestWindowsAutopilotDevicesFuncInvoked,
			"re-syncing an identical registry must not ship every device over the wire again")
	})

	t.Run("a changed group tag updates in place", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", "Engineering"))
		env.sync(t)

		env.graph(tenantA, device("ap-1", "SERIAL-1", "Marketing"))
		env.sync(t)

		assert.Equal(t, "Marketing", env.device(t, "ap-1").GroupTag,
			"the diff must notice a group tag change and ship the device again")
	})

	t.Run("a device that joins entra updates in place", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		unjoined := device("ap-1", "SERIAL-1", "")
		unjoined.EntraDeviceID = ""
		env.graph(tenantA, unjoined)
		env.sync(t)
		require.Empty(t, env.device(t, "ap-1").EntraDeviceID)

		// A device is issued its Entra device ID when it joins, which is after it was registered with Autopilot. A
		// diff that only watched the group tag would never store it.
		env.graph(tenantA, device("ap-1", "SERIAL-1", ""))
		env.sync(t)
		assert.Equal(t, "aad-ap-1", env.device(t, "ap-1").EntraDeviceID)
	})

	t.Run("a device removed from autopilot removes its pending host", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", ""), device("ap-2", "SERIAL-2", ""))
		env.sync(t)
		goneID := env.hostID(t, "ap-2")

		env.graph(tenantA, device("ap-1", "SERIAL-1", ""))
		env.sync(t)

		assert.Equal(t, []uint{goneID}, env.removed)
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
	})

	// An empty list is not a special case. Every way of being handed a wrong or truncated one already fails before the
	// diff, so a successful response listing nothing is a fact about the tenant rather than a symptom.
	t.Run("a tenant that empties out removes all of its pending hosts", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", ""))
		env.sync(t)
		goneID := env.hostID(t, "ap-1")

		env.graph(tenantA)
		env.sync(t)

		assert.Equal(t, []uint{goneID}, env.removed)
		assert.Empty(t, env.serials())
	})

	t.Run("two devices sharing a serial are two pending hosts", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "DUP-SERIAL", "Engineering"), device("ap-2", "DUP-SERIAL", "Marketing"))
		env.sync(t)

		assert.ElementsMatch(t, []string{"DUP-SERIAL", "DUP-SERIAL"}, env.serials(),
			"the sync must not collapse two registrations that share a serial")

		// Retiring one must leave the other alone. Diffing on the serial would find the serial still present and
		// remove nothing, or remove both.
		goneID := env.hostID(t, "ap-2")
		env.graph(tenantA, device("ap-1", "DUP-SERIAL", "Engineering"))
		env.sync(t)

		assert.Equal(t, []uint{goneID}, env.removed)
		assert.ElementsMatch(t, []string{"DUP-SERIAL"}, env.serials())
	})

	// Both fields are the identity a pending host is built from: the device ID resolves it, the serial is all it has
	// until the machine boots. A device missing either cannot become a usable pending host.
	for _, tc := range []struct {
		name   string
		device msgraph.WindowsAutopilotDevice
	}{
		{"a device with no autopilot device id is skipped", device("", "SERIAL-2", "")},
		{"a device with a placeholder serial is skipped", device("ap-2", "Default string", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newAutopilotSyncEnv(t, testCred(tenantA))
			env.graph(tenantA, device("ap-1", "SERIAL-1", ""), tc.device)
			env.sync(t)
			assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
		})
	}

	t.Run("a pagination error aborts the tenant without deleting anything", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graph(tenantA, device("ap-1", "SERIAL-1", ""))
		env.sync(t)

		env.graphFails(tenantA, errors.New("pagination stopped advancing"))
		env.sync(t)

		assert.Empty(t, env.removed)
		assert.ElementsMatch(t, []string{"SERIAL-1"}, env.serials())
		require.NotNil(t, env.syncResults[tenantA])
		// A failure Graph did not report has no admin-facing remedy, so the stored message stays generic.
		assert.Equal(t, "Couldn't sync Windows Autopilot devices from Microsoft Graph.", *env.syncResults[tenantA])
	})

	t.Run("a graph failure is stored as one admin-facing sentence, not the wrap chain", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graphFails(tenantA, &msgraph.Error{StatusCode: http.StatusUnauthorized, Code: "invalid_client", Message: "AADSTS7000215: bad secret"})
		env.sync(t)

		require.NotNil(t, env.syncResults[tenantA])
		stored := *env.syncResults[tenantA]
		assert.Equal(t, "Microsoft Graph rejected the credential. Check the tenant ID, client ID, and client secret.", stored)
		assert.NotContains(t, stored, "list windows autopilot devices", "the ctxerr wrap chain must never reach the UI")
	})
}

func TestMicrosoftAutopilotSyncCredentialFlag(t *testing.T) {
	t.Parallel()

	// Only an explicit credential rejection may raise the alarm. A Microsoft outage must never flag a credential, or
	// one bad hour at Microsoft would tell every Fleet admin to go re-enter their client secret.
	for _, tc := range []struct {
		name        string
		err         error
		wantInvalid bool
	}{
		{"graph 401 marks the credential invalid", &msgraph.Error{StatusCode: http.StatusUnauthorized}, true},
		{"graph 403 marks the credential invalid", &msgraph.Error{StatusCode: http.StatusForbidden}, true},
		{"graph 429 does not", &msgraph.Error{StatusCode: http.StatusTooManyRequests}, false},
		{"graph 500 does not", &msgraph.Error{StatusCode: http.StatusInternalServerError}, false},
		{"a non-graph error does not", errors.New("dial tcp: timeout"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newAutopilotSyncEnv(t, testCred(tenantA))
			env.graphFails(tenantA, tc.err)
			env.sync(t)

			assert.Equal(t, tc.wantInvalid, env.invalid[tenantA])
			// The banner is driven by a stored aggregate on the app config, recomputed once per pass.
			assert.Equal(t, 1, env.aggregateRefreshed, "the banner aggregate is recomputed on every pass")
		})
	}

	t.Run("the flag and the banner clear on the next successful sync", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		env.graphFails(tenantA, &msgraph.Error{StatusCode: http.StatusUnauthorized})
		env.sync(t)
		require.True(t, env.invalid[tenantA])

		env.graph(tenantA, device("ap-1", "SERIAL-1", ""))
		env.sync(t)

		assert.False(t, env.invalid[tenantA])
		assert.Equal(t, 2, env.aggregateRefreshed, "clearing the flag must refresh the banner too")
	})

	t.Run("a failed banner recomputation is retried on the next pass", func(t *testing.T) {
		env := newAutopilotSyncEnv(t, testCred(tenantA))
		aggregateErr := errors.New("could not save app config")
		env.ds.UpdateMicrosoftGraphCredentialInvalidAggregateFunc = func(ctx context.Context) error {
			env.aggregateRefreshed++
			if env.aggregateRefreshed == 1 {
				return aggregateErr
			}
			return nil
		}

		env.graphFails(tenantA, &msgraph.Error{StatusCode: http.StatusUnauthorized})
		require.ErrorIs(t, cronMicrosoftAutopilotSync(t.Context(), env.ds, factoryFor(env.clients), discardLogger()), aggregateErr)
		require.True(t, env.invalid[tenantA], "the per-tenant flag is stored even though the banner was not recomputed")

		// Nothing flips on this pass: the credential is still rejected and the flag is already set.
		env.sync(t)
		assert.Equal(t, 2, env.aggregateRefreshed, "the banner recomputation must be retried until it succeeds")
	})
}

func TestMicrosoftAutopilotSyncIsolatesTenants(t *testing.T) {
	t.Parallel()

	env := newAutopilotSyncEnv(t, testCred(tenantA), testCred(tenantB))
	env.graphFails(tenantA, &msgraph.Error{StatusCode: http.StatusUnauthorized})
	env.graph(tenantB, device("ap-b", "SERIAL-B", "Sales"))
	env.sync(t)

	assert.True(t, env.invalid[tenantA], "the failing tenant is flagged")
	assert.False(t, env.invalid[tenantB], "the healthy tenant is not")
	assert.ElementsMatch(t, []string{"SERIAL-B"}, env.serials(),
		"one tenant's failure must not stop another tenant's sync")
}
