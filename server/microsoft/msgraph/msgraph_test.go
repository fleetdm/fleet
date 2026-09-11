package msgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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
			// The tenant must appear in the token URL: client credentials is tenant-scoped
			assert.Contains(t, r.URL.Path, testTenantID)
			// Bound the body before parsing.
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
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

func writeDevices(t *testing.T, w http.ResponseWriter, nextLink string, devices ...WindowsAutopilotDevice) {
	t.Helper()
	body := map[string]any{"value": devices}
	if nextLink != "" {
		body["@odata.nextLink"] = nextLink
	}
	w.Header().Set("Content-Type", "application/json")
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}

// newSingleHostClient points both the token endpoint and Graph at one server, for tests that need to control the
// token response itself rather than just the Graph response.
func newSingleHostClient(t *testing.T, handler http.HandlerFunc) Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := newClientWithHosts(&fleet.MicrosoftGraphCredential{
		TenantID: testTenantID, ClientID: testClientID, ClientSecret: testSecret,
	}, srv.URL, srv.URL)
	require.NoError(t, err)
	return c
}

// newPagedGraphServer serves the given pages in order, linking each to the next. Page N is requested with
// ?$skiptoken=pageN, so the sequence is driven by the request rather than by call-order state.
func newPagedGraphServer(t *testing.T, pages ...[]WindowsAutopilotDevice) *graphServer {
	t.Helper()
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		i := 0
		if tok := r.URL.Query().Get("$skiptoken"); tok != "" {
			_, _ = fmt.Sscanf(tok, "page%d", &i)
		}
		// assert, not require: a failed require here would call t.FailNow off the test goroutine.
		if !assert.Less(t, i, len(pages), "client requested a page past the end of the fixture") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var next string
		if i+1 < len(pages) {
			next = fmt.Sprintf("%s%s?$skiptoken=page%d", gs.URL, autopilotDevicesPath, i+1)
		}
		writeDevices(t, w, next, pages[i]...)
	})
	return gs
}

func device(id, serial, tag string) WindowsAutopilotDevice {
	return WindowsAutopilotDevice{ID: id, SerialNumber: serial, GroupTag: tag, EntraDeviceID: "aad-" + id}
}

func TestNewClientRequiresFullCredential(t *testing.T) {
	t.Parallel()
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
			c, err := NewClient(tc.cred)
			require.Error(t, err)
			assert.Nil(t, c, "a rejected credential must not yield a usable client")
			assert.Contains(t, err.Error(), "not fully configured")
		})
	}
}

func TestListPagination(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		pages   [][]WindowsAutopilotDevice
		wantIDs []string
	}{
		{
			name:    "single page",
			pages:   [][]WindowsAutopilotDevice{{device("id-1", "SERIAL-1", "Engineering"), device("id-2", "SERIAL-2", "")}},
			wantIDs: []string{"id-1", "id-2"},
		},
		{
			// A tenant with no Autopilot registrations is a valid configuration, not a loop or an error.
			name:    "empty tenant",
			pages:   [][]WindowsAutopilotDevice{{}},
			wantIDs: nil,
		},
		{
			name: "multiple pages, order preserved",
			pages: [][]WindowsAutopilotDevice{
				{device("id-1", "SERIAL-1", "A")},
				{device("id-2", "SERIAL-2", "B")},
				{device("id-3", "SERIAL-3", "C")},
			},
			wantIDs: []string{"id-1", "id-2", "id-3"},
		},
		{
			// Graph's cursor is inclusive, so the last device of a page reappears as the first of the next. Verified
			// live: at $top=2 a five-device tenant returned seven rows.
			name: "inclusive cursor repeats the boundary device",
			pages: [][]WindowsAutopilotDevice{
				{device("id-1", "SERIAL-1", "A"), device("id-2", "SERIAL-2", "B")},
				{device("id-2", "SERIAL-2", "B"), device("id-3", "SERIAL-3", "C")},
			},
			wantIDs: []string{"id-1", "id-2", "id-3"},
		},
		{
			name: "devices without an id are skipped",
			pages: [][]WindowsAutopilotDevice{
				{device("", "SERIAL-1", "A"), device("id-2", "SERIAL-2", "B")},
			},
			wantIDs: []string{"id-2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := newPagedGraphServer(t, tc.pages...)

			devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(devices))
			for _, d := range devices {
				gotIDs = append(gotIDs, d.ID)
			}
			assert.Equal(t, tc.wantIDs, nilIfEmpty(gotIDs), "each device exactly once, in page order")
		})
	}
}

func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// Field mapping is separate from pagination: it pins the Graph wire names onto our struct, including the two values
// most likely to be mangled (an empty group tag, and one at Intune's 2048-character maximum).
func TestListParsesDeviceFields(t *testing.T) {
	t.Parallel()
	maxTag := strings.Repeat("a", 2048)
	gs := newPagedGraphServer(t, []WindowsAutopilotDevice{
		device("id-1", "SERIAL-1", "Engineering"),
		device("id-2", "VMware-56 4d 51 82", ""),
		device("id-3", "SERIAL-3", maxTag),
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 3)

	assert.Equal(t, "SERIAL-1", devices[0].SerialNumber)
	assert.Equal(t, "Engineering", devices[0].GroupTag)
	assert.Equal(t, "aad-id-1", devices[0].EntraDeviceID)
	// Empty is the common real-world case and must survive as empty rather than being dropped.
	assert.Empty(t, devices[1].GroupTag)
	// Serials can carry spaces; they must not be trimmed or split.
	assert.Equal(t, "VMware-56 4d 51 82", devices[1].SerialNumber)
	assert.Len(t, devices[2].GroupTag, 2048, "Intune's maximum group tag must survive intact")
	assert.Equal(t, maxTag, devices[2].GroupTag)
}

// The walk must refuse to continue in several shapes, each of which was either observed live or is a token-safety hazard.
func TestListRefusesToContinue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		handler func(gs *graphServer) http.HandlerFunc
		wantErr string
	}{
		{
			// Verified live at $top=1: the service echoes back a nextLink byte-identical to the URL just requested,
			// so "follow until absent" never terminates.
			name: "nextLink identical to the request URL",
			handler: func(gs *graphServer) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					writeDevices(t, w, gs.URL+r.URL.String(),
						device(fmt.Sprintf("id-%d", gs.graphRequests.Load()), "SERIAL-1", "A"))
				}
			},
			wantErr: "identical to the request URL",
		},
		{
			// The cursor is keyed on serial, and serials are not unique, so a run sharing one serial stalls it even
			// though the URL keeps changing.
			name: "same devices under an advancing cursor",
			handler: func(gs *graphServer) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					next := fmt.Sprintf("%s%s?$skiptoken=%s-more", gs.URL, autopilotDevicesPath, r.URL.Query().Get("$skiptoken"))
					writeDevices(t, w, next, device("id-1", "Default string", "A"), device("id-2", "Default string", "B"))
				}
			},
			wantErr: "stopped advancing",
		},
		{
			// Empty pages with a changing cursor are the one shape the identical-URL guard cannot see.
			name: "empty pages under an advancing cursor",
			handler: func(gs *graphServer) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					next := fmt.Sprintf("%s%s?$skiptoken=%s-more", gs.URL, autopilotDevicesPath, r.URL.Query().Get("$skiptoken"))
					writeDevices(t, w, next)
				}
			},
			wantErr: "stopped advancing",
		},
		{
			// A relative nextLink has no origin to validate, so it must be refused rather than resolved.
			name: "relative nextLink",
			handler: func(gs *graphServer) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					writeDevices(t, w, autopilotDevicesPath+"?$skiptoken=x", device("id-1", "SERIAL-1", "A"))
				}
			},
			wantErr: "unexpected origin",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gs *graphServer
			gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) { tc.handler(gs)(w, r) })

			devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Nil(t, devices, "a partial list must not be returned; the sync would treat it as authoritative")
			assert.Less(t, gs.graphRequests.Load(), int32(5), "must give up immediately, not grind to the page cap")
		})
	}
}

// The oauth2 transport attaches the bearer token to whatever we request, so a nextLink off the Graph origin is an
// exfiltration vector, not just a correctness bug.
func TestListRejectsNextLinkOnAnotherOrigin(t *testing.T) {
	t.Parallel()
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "client followed a next link to a foreign origin", "sent header %q", r.Header.Get("Authorization"))
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

func TestListClassifiesErrors(t *testing.T) {
	t.Parallel()
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
			// ...while directory endpoints answer the same cause with a different code.
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

			graphErr, ok := errors.AsType[*Error](err)
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
	t.Parallel()
	t.Run("succeeds on a good page", func(t *testing.T) {
		var gotTop string
		gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotTop = r.URL.Query().Get("$top")
			writeDevices(t, w, "", device("id-1", "SERIAL-1", "A"))
		})
		require.NoError(t, gs.client(t).VerifyCredential(t.Context()))
		// Verification must stay cheap: one request, and one device rather than a full page.
		assert.Equal(t, int32(1), gs.graphRequests.Load())
		assert.Equal(t, strconv.Itoa(verifyPageSize), gotTop)
		assert.Equal(t, 1, verifyPageSize)
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
		graphErr, ok := errors.AsType[*Error](err)
		require.True(t, ok)
		assert.True(t, graphErr.IsPermissionError())
	})
}

// A wrong or expired client secret fails at Entra's token endpoint, before Graph is reached, so it never becomes a
// Graph response. It still has to classify as an auth failure, or the admin is told it's a connection problem.
func TestTokenEndpointInvalidClientClassifiesAsAuthError(t *testing.T) {
	t.Parallel()
	c := newSingleHostClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret provided."}`))
	})

	_, err := c.ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)

	graphErr, ok := errors.AsType[*Error](err)
	require.True(t, ok, "a token-endpoint failure must still be classifiable")
	assert.True(t, graphErr.IsAuthError())
	assert.False(t, graphErr.IsPermissionError())
	assert.False(t, graphErr.IsTransient())
	assert.Equal(t, "invalid_client", graphErr.Code)
	assert.Contains(t, graphErr.Message, "AADSTS7000215")
}

// Token acquisition must inherit the caller's context.
func TestTokenAcquisitionHonorsCallerContext(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	// defer, not t.Cleanup: cleanups run LIFO, so the server's Close would run first and block forever waiting on a
	// handler that is itself blocked on this channel. A deferred close runs before any cleanup and breaks that
	// deadlock, which matters precisely when the test fails and you want a readable failure instead of a hang.
	defer close(release)

	c := newSingleHostClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Hang the token endpoint until the test returns.
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3599}`))
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := c.ListWindowsAutopilotDevices(ctx); done <- err }()

	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "a cancelled caller must abort the pending token fetch")
	case <-time.After(5 * time.Second):
		t.Fatal("token acquisition ignored the cancelled caller context and kept waiting")
	}
}

// Every page of one listing shares a single access token; a token per page would multiply calls against Entra.
func TestListRequestShape(t *testing.T) {
	t.Parallel()
	var tops []string
	gs := newPagedGraphServer(t,
		[]WindowsAutopilotDevice{device("id-1", "S1", "A")},
		[]WindowsAutopilotDevice{device("id-2", "S2", "B")},
		[]WindowsAutopilotDevice{device("id-3", "S3", "C")},
	)
	inner := gs.Config.Handler
	gs.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if top := r.URL.Query().Get("$top"); top != "" {
			tops = append(tops, top)
		}
		inner.ServeHTTP(w, r)
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)
	require.Len(t, devices, 3)

	require.NotEmpty(t, tops, "the first request must pin $top")
	assert.Equal(t, strconv.Itoa(pageSize), tops[0])
	assert.GreaterOrEqual(t, pageSize, 1000,
		"lowering the page size multiplies round trips and inclusive-cursor duplicates for large tenants")
	assert.Equal(t, int32(1), gs.tokenRequests.Load(), "all pages must share one token")
}

// The error path must bound what it reads, not read everything and then truncate: an edge proxy can answer a 5xx with a
// very large body, and this message ends up in logs and in the sync error shown to the admin.
func TestErrorBodyIsBoundedBeforeReading(t *testing.T) {
	t.Parallel()
	const huge = 5 << 20 // 5MB
	var served atomic.Int64
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		chunk := bytes.Repeat([]byte("x"), 64<<10)
		for written := 0; written < huge; written += len(chunk) {
			n, err := w.Write(chunk)
			served.Add(int64(n))
			if err != nil {
				return
			}
		}
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)

	graphErr, ok := errors.AsType[*Error](err)
	require.True(t, ok, "a 502 must surface as a *msgraph.Error")
	assert.LessOrEqual(t, len(graphErr.Message), maxErrorBodyBytes+len("... (truncated)"),
		"the retained message must stay bounded")
	assert.Contains(t, graphErr.Message, "truncated")

	// The retained message is bounded either way, because truncateBody trims it after the fact. What distinguishes a
	// bounded read is that the client stops pulling, so the server never gets to write the whole body.
	assert.Less(t, served.Load(), int64(huge),
		"the client must stop reading rather than allocate the entire error body")
}
