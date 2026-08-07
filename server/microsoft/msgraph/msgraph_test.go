package msgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantID = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
	testClientID = "7f6b1665-51f5-48de-a9b6-ac17539583fb"
	testSecret   = "test-client-secret"
)

// graphServer is a stand-in for both the Entra token endpoint and Microsoft Graph. handler serves only the Autopilot
// collection; token requests are answered generically.
type graphServer struct {
	*httptest.Server
	tokenRequests atomic.Int32
	graphRequests atomic.Int32
}

func newGraphServer(t *testing.T, handler http.HandlerFunc) *graphServer {
	t.Helper()
	gs := &graphServer{}
	gs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token") {
			gs.tokenRequests.Add(1)
			// The tenant must appear in the token URL: client credentials is tenant-scoped, and Microsoft rejects
			// /common and /organizations for this flow.
			assert.Contains(t, r.URL.Path, testTenantID)
			// Bound the body before parsing (gosec G120): unbounded ParseForm can be a memory-exhaustion vector.
			// The token form is tiny, so 1 MiB is generous.
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			// assert, not require: a failed require inside an HTTP handler would call t.FailNow off the test
			// goroutine, which testify cannot do safely.
			assert.NoError(t, r.ParseForm())
			assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
			assert.Equal(t, testClientID, r.Form.Get("client_id"))
			assert.Equal(t, testSecret, r.Form.Get("client_secret"))
			assert.Equal(t, graphScope, r.Form.Get("scope"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3599}`))
			return
		}
		gs.graphRequests.Add(1)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		handler(w, r)
	}))
	t.Cleanup(gs.Close)
	return gs
}

func (gs *graphServer) client(t *testing.T) Client {
	t.Helper()
	c, err := newClientWithHosts(&fleet.MicrosoftGraphCredential{
		TenantID: testTenantID, ClientID: testClientID, ClientSecret: testSecret,
	}, gs.URL, gs.URL)
	require.NoError(t, err)
	return c
}

func writeDevices(t *testing.T, w http.ResponseWriter, nextLink string, devices ...fleet.WindowsAutopilotDevice) {
	t.Helper()
	body := map[string]any{"value": devices}
	if nextLink != "" {
		body["@odata.nextLink"] = nextLink
	}
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func device(id, serial, tag string) fleet.WindowsAutopilotDevice {
	return fleet.WindowsAutopilotDevice{ID: id, SerialNumber: serial, GroupTag: tag, AzureADDeviceID: "aad-" + id}
}

func TestNewClientRequiresFullCredential(t *testing.T) {
	for _, tc := range []struct {
		name string
		cred *fleet.MicrosoftGraphCredential
	}{
		{"nil", nil},
		{"missing secret", &fleet.MicrosoftGraphCredential{TenantID: "t", ClientID: "c"}},
		{"missing client", &fleet.MicrosoftGraphCredential{TenantID: "t", ClientSecret: "s"}},
		{"missing tenant", &fleet.MicrosoftGraphCredential{ClientID: "c", ClientSecret: "s"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClient(tc.cred)
			require.Error(t, err)
		})
	}
}

func TestListSinglePage(t *testing.T) {
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeDevices(t, w, "",
			device("id-1", "SERIAL-1", "Engineering"),
			device("id-2", "SERIAL-2", ""),
		)
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 2)

	assert.Equal(t, "id-1", devices[0].ID)
	assert.Equal(t, "SERIAL-1", devices[0].SerialNumber)
	assert.Equal(t, "Engineering", devices[0].GroupTag)
	assert.Equal(t, "aad-id-1", devices[0].AzureADDeviceID)
	// An empty group tag is the common real-world case and must survive as empty rather than being dropped.
	assert.Empty(t, devices[1].GroupTag)

	assert.Equal(t, int32(1), gs.tokenRequests.Load(), "token should be minted once")
}

func TestListFollowsNextLink(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("$skiptoken") {
		case "":
			writeDevices(t, w, gs.URL+autopilotDevicesPath+"?$skiptoken=page2", device("id-1", "SERIAL-1", "A"))
		case "page2":
			writeDevices(t, w, gs.URL+autopilotDevicesPath+"?$skiptoken=page3", device("id-2", "SERIAL-2", "B"))
		default:
			writeDevices(t, w, "", device("id-3", "SERIAL-3", "C"))
		}
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 3)
	assert.Equal(t, []string{"id-1", "id-2", "id-3"}, []string{devices[0].ID, devices[1].ID, devices[2].ID})
}

// Graph's cursor is inclusive, so the last device of a page reappears as the first device of the next. Verified live:
// at $top=2 a five-device tenant returned seven rows. Each device must still be emitted exactly once.
func TestListDeduplicatesBoundaryDevice(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("$skiptoken") == "" {
			writeDevices(t, w, gs.URL+autopilotDevicesPath+"?$skiptoken=LastSerialNumber='SERIAL-2'",
				device("id-1", "SERIAL-1", "A"), device("id-2", "SERIAL-2", "B"))
			return
		}
		// The boundary device repeats here, exactly as the live service does.
		writeDevices(t, w, "", device("id-2", "SERIAL-2", "B"), device("id-3", "SERIAL-3", "C"))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 3, "the boundary device must be returned once, not twice")

	ids := map[string]int{}
	for _, d := range devices {
		ids[d.ID]++
	}
	assert.Equal(t, map[string]int{"id-1": 1, "id-2": 1, "id-3": 1}, ids)
}

// The hang this guard exists for: at $top=1 the live service returns an @odata.nextLink byte-identical to the URL just
// requested, so "follow nextLink until absent" never terminates. The walk must stop instead of spinning.
func TestListTerminatesOnSelfReferentialNextLink(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		selfLink := gs.URL + autopilotDevicesPath + "?$skiptoken=LastSerialNumber='SERIAL-1'"
		if r.URL.Query().Get("$skiptoken") == "" {
			writeDevices(t, w, selfLink, device("id-1", "SERIAL-1", "A"))
			return
		}
		// Echo back the very same link forever.
		writeDevices(t, w, selfLink, device("id-1", "SERIAL-1", "A"))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	// Either guard is an acceptable stop; what matters is that it stops, and that it does not hand back a partial list
	// the sync would treat as authoritative.
	assert.Contains(t, err.Error(), "aborting")
	// It must give up almost immediately rather than grinding to the page cap and hammering Graph.
	assert.Less(t, gs.graphRequests.Load(), int32(5))
}

// Because the cursor is keyed on serial number and serials are not unique, a run of devices sharing one serial stops
// the cursor advancing even though the URL keeps changing. The walk must fail rather than loop, and must not hand back
// the devices gathered so far: a partial list would make the sync delete every pending host it did not see.
func TestListFailsWhenPageYieldsNothingNew(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Every page hands back the same devices under a different cursor value, so the URL keeps changing but no
		// progress is made.
		token := r.URL.Query().Get("$skiptoken")
		next := fmt.Sprintf("%s%s?$skiptoken=%s-more", gs.URL, autopilotDevicesPath, token)
		writeDevices(t, w, next,
			device("id-1", "Default string", "A"),
			device("id-2", "Default string", "B"))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped advancing")
	assert.Nil(t, devices, "a partial list must not be returned; the sync would treat it as authoritative")
	assert.Less(t, gs.graphRequests.Load(), int32(5))
}

func TestListSkipsDevicesWithoutID(t *testing.T) {
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeDevices(t, w, "", device("", "SERIAL-1", "A"), device("id-2", "SERIAL-2", "B"))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "id-2", devices[0].ID)
}

func TestListMaxLengthGroupTag(t *testing.T) {
	maxTag := strings.Repeat("a", 2048)
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeDevices(t, w, "", device("id-1", "SERIAL-1", maxTag))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Len(t, devices[0].GroupTag, 2048, "Intune's maximum group tag must survive intact")
	assert.Equal(t, maxTag, devices[0].GroupTag)
}

func TestListClassifiesErrors(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		body         string
		retryAfter   string
		wantAuth     bool
		wantPerm     bool
		wantTransien bool
		wantCode     string
	}{
		{
			name: "unauthorized", status: http.StatusUnauthorized,
			body:     `{"error":{"code":"InvalidAuthenticationToken","message":"Access token is empty."}}`,
			wantAuth: true, wantCode: "InvalidAuthenticationToken",
		},
		{
			// Intune endpoints answer a missing permission with "Forbidden"...
			name: "forbidden intune", status: http.StatusForbidden,
			body:     `{"error":{"code":"Forbidden","message":"Application is not authorized."}}`,
			wantPerm: true, wantCode: "Forbidden",
		},
		{
			// ...while directory endpoints answer the same cause with a different code. Classification keys on the
			// status precisely so both land in the same bucket.
			name: "forbidden directory", status: http.StatusForbidden,
			body:     `{"error":{"code":"Authorization_RequestDenied","message":"Insufficient privileges."}}`,
			wantPerm: true, wantCode: "Authorization_RequestDenied",
		},
		{
			name: "throttled", status: http.StatusTooManyRequests,
			body: `{"error":{"code":"TooManyRequests","message":"slow down"}}`, retryAfter: "42",
			wantTransien: true, wantCode: "TooManyRequests",
		},
		{
			name: "server error", status: http.StatusBadGateway, body: `bad gateway`,
			wantTransien: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.retryAfter != "" {
					w.Header().Set("Retry-After", tc.retryAfter)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
			require.Error(t, err)

			graphErr, ok := AsError(err)
			require.True(t, ok, "error must remain classifiable through the wrap chain")
			assert.Equal(t, tc.status, graphErr.StatusCode)
			assert.Equal(t, tc.wantAuth, graphErr.IsAuthError())
			assert.Equal(t, tc.wantPerm, graphErr.IsPermissionError())
			assert.Equal(t, tc.wantTransien, graphErr.IsTransient())
			if tc.wantCode != "" {
				assert.Equal(t, tc.wantCode, graphErr.Code)
			}
			if tc.retryAfter != "" {
				assert.Equal(t, 42, int(graphErr.RetryAfter.Seconds()))
			}
		})
	}
}

func TestVerifyCredential(t *testing.T) {
	t.Run("succeeds on a good page", func(t *testing.T) {
		gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
			writeDevices(t, w, "", device("id-1", "SERIAL-1", "A"))
		})
		require.NoError(t, gs.client(t).VerifyCredential(t.Context()))
		// Verification must stay cheap: one page, regardless of how many the tenant has.
		assert.Equal(t, int32(1), gs.graphRequests.Load())
	})

	t.Run("succeeds on an empty tenant", func(t *testing.T) {
		// A tenant with no Autopilot registrations is a valid configuration, not a bad credential.
		gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
			writeDevices(t, w, "")
		})
		require.NoError(t, gs.client(t).VerifyCredential(t.Context()))
	})

	t.Run("fails when the app lacks permission", func(t *testing.T) {
		gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"Forbidden","message":"no consent"}}`))
		})
		err := gs.client(t).VerifyCredential(t.Context())
		require.Error(t, err)
		graphErr, ok := AsError(err)
		require.True(t, ok)
		assert.True(t, graphErr.IsPermissionError())
	})
}

func TestTokenAcquisitionFailureSurfaces(t *testing.T) {
	// A wrong or expired client secret fails at the token endpoint, before Graph is ever reached. The error has to
	// reach the caller rather than being swallowed as an empty device list, which would look like "tenant has no
	// devices" and silently delete pending hosts downstream.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret provided."}`))
	}))
	t.Cleanup(srv.Close)

	c, err := newClientWithHosts(&fleet.MicrosoftGraphCredential{
		TenantID: testTenantID, ClientID: testClientID, ClientSecret: "wrong",
	}, srv.URL, srv.URL)
	require.NoError(t, err)

	_, err = c.ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AADSTS7000215")
}

// A next link is a URL chosen by the remote service, and the oauth2 transport attaches the access token to whatever we
// request. A link pointing off the Graph origin must be refused rather than followed, or Fleet would hand its token to
// another host.
func TestListRejectsNextLinkOnAnotherOrigin(t *testing.T) {
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("client followed a next link to a foreign origin and sent header %q", r.Header.Get("Authorization"))
		writeDevices(t, w, "")
	}))
	t.Cleanup(evil.Close)

	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeDevices(t, w, evil.URL+autopilotDevicesPath, device("id-1", "SERIAL-1", "A"))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected origin")
	assert.Nil(t, devices)
}

func TestListRejectsRelativeNextLink(t *testing.T) {
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeDevices(t, w, "/v1.0/deviceManagement/windowsAutopilotDeviceIdentities?$skiptoken=x",
			device("id-1", "SERIAL-1", "A"))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected origin")
}

// A wrong or expired client secret fails at Entra's token endpoint, before Graph is reached, so it never becomes a
// Graph response. It still has to classify as an auth failure, or the admin is told it's a connection problem.
func TestTokenEndpointInvalidClientClassifiesAsAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret provided."}`))
	}))
	t.Cleanup(srv.Close)

	c, err := newClientWithHosts(&fleet.MicrosoftGraphCredential{
		TenantID: testTenantID, ClientID: testClientID, ClientSecret: "wrong",
	}, srv.URL, srv.URL)
	require.NoError(t, err)

	_, err = c.ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)

	graphErr, ok := AsError(err)
	require.True(t, ok, "a token-endpoint failure must still be classifiable")
	assert.True(t, graphErr.IsAuthError())
	assert.False(t, graphErr.IsPermissionError())
	assert.False(t, graphErr.IsTransient())
	assert.Equal(t, "invalid_client", graphErr.Code)
	assert.Contains(t, graphErr.Message, "AADSTS7000215")
}

// Token acquisition must inherit the caller's context. clientcredentials.Config.Client bakes the context into its
// token source, so a client built once at construction would fetch tokens under a background context and keep waiting
// after the caller had cancelled.
func TestTokenAcquisitionHonorsCallerContext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang the token endpoint until the test releases it.
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3599}`))
	}))
	t.Cleanup(func() { close(release); srv.Close() })

	c, err := newClientWithHosts(&fleet.MicrosoftGraphCredential{
		TenantID: testTenantID, ClientID: testClientID, ClientSecret: testSecret,
	}, srv.URL, srv.URL)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := c.ListWindowsAutopilotDevices(ctx); done <- err }()

	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "a cancelled caller must abort the pending token fetch")
	case <-time.After(10 * time.Second):
		t.Fatal("token acquisition ignored the cancelled caller context and kept waiting")
	}
}

// Every page of one listing shares a single access token; a token per page would multiply calls against Entra.
func TestListMintsOneTokenPerOperation(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("$skiptoken") {
		case "":
			writeDevices(t, w, gs.URL+autopilotDevicesPath+"?$skiptoken=p2", device("id-1", "S1", "A"))
		case "p2":
			writeDevices(t, w, gs.URL+autopilotDevicesPath+"?$skiptoken=p3", device("id-2", "S2", "B"))
		default:
			writeDevices(t, w, "", device("id-3", "S3", "C"))
		}
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 3)
	assert.Equal(t, int32(1), gs.tokenRequests.Load(), "all pages must share one token")
}
