package msgraph

import (
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
	"golang.org/x/oauth2"
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
	require.NoError(t, json.NewEncoder(w).Encode(body))
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

// The hang this guard exists for: at $top=1 the live service returns an @odata.nextLink byte-identical to the URL just
// requested, so "follow nextLink until absent" never terminates. The walk must stop instead of spinning.
func TestListTerminatesOnSelfReferentialNextLink(t *testing.T) {
	var gs *graphServer
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Echo back the exact URL just requested, to simulate infinite loop.
		writeDevices(t, w, gs.URL+r.URL.String(), device(fmt.Sprintf("id-%d", gs.graphRequests.Load()), "SERIAL-1", "A"))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	// Assert the specific guard.
	assert.Contains(t, err.Error(), "identical to the request URL")
	assert.Less(t, gs.graphRequests.Load(), int32(5), "must give up immediately, not grind to the page cap")
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

// The page size is sent explicitly rather than relying on Graph's default.
func TestListRequestsExplicitPageSize(t *testing.T) {
	var gotTop string
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotTop = r.URL.Query().Get("$top")
		writeDevices(t, w, "", device("id-1", "SERIAL-1", "A"))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.NoError(t, err)

	assert.Equal(t, strconv.Itoa(pageSize), gotTop, "the first request must pin $top")
	assert.GreaterOrEqual(t, pageSize, 1000,
		"lowering the page size multiplies round trips and inclusive-cursor duplicates for large tenants")
}

// Empty pages carrying a non-empty nextLink are the one loop shape the identical-URL guard cannot see, because the URL
// keeps changing. The no-progress guard has to catch it.
func TestListFailsFastOnEmptyPagesWithAdvancingCursor(t *testing.T) {
	var gs *graphServer
	pages := 0
	gs = newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		writeDevices(t, w, fmt.Sprintf("%s%s?$skiptoken=p%d", gs.URL, autopilotDevicesPath, pages))
	})

	devices, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped advancing")
	assert.Nil(t, devices, "a partial list must not be returned")
	assert.LessOrEqual(t, pages, 2, "must stop on the first unproductive page, not grind to maxPages")
}

func TestErrorUnwrapsUnderlyingCause(t *testing.T) {
	// A token-endpoint failure must keep the oauth2 error reachable.
	retrieveErr := &oauth2.RetrieveError{
		ErrorCode:        "invalid_client",
		ErrorDescription: "AADSTS7000215: Invalid client secret provided.",
	}
	wrapped := fmt.Errorf("outer: %w", newTokenError(retrieveErr))

	graphErr, ok := errors.AsType[*Error](wrapped)
	require.True(t, ok)
	assert.True(t, graphErr.IsAuthError())

	var gotRetrieve *oauth2.RetrieveError
	require.ErrorAs(t, wrapped, &gotRetrieve, "the oauth2 cause must survive wrapping")
	require.NotNil(t, gotRetrieve)
	assert.Equal(t, "invalid_client", gotRetrieve.ErrorCode)
}

func TestParseRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   bool // whether a positive duration is expected
	}{
		{"delta seconds", "42", true},
		{"zero", "0", false},
		{"negative is ignored", "-5", false},
		{"absent", "", false},
		{"garbage", "soon", false},
		// RFC 7231 permits an HTTP-date. Reading it as zero would mean no backoff at all.
		{"http date in the future", time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat), true},
		{"http date in the past", time.Now().Add(-90 * time.Second).UTC().Format(http.TimeFormat), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRetryAfter(tc.header)
			assert.Equal(t, tc.want, got > 0, "got %v", got)
		})
	}
	assert.Equal(t, 42*time.Second, parseRetryAfter("42"))
}

func TestGraphErrorBodyIsBounded(t *testing.T) {
	// An edge proxy can return a large HTML page on 5xx; this string lands in logs and in the admin-visible sync error.
	huge := strings.Repeat("x", 10_000)
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(huge))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	graphErr, ok := errors.AsType[*Error](err)
	require.True(t, ok)
	assert.LessOrEqual(t, len(graphErr.Message), maxErrorBodyBytes+len("... (truncated)"))
	assert.Contains(t, graphErr.Message, "truncated")
}
