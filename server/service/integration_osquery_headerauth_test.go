package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationsOsqueryHeaderAuth covers the end-to-end behavior of
// osquery host authentication through the real HTTP handler stack
// (kithttp.ServerBefore hooks, decoders, authzcheck, error encoding) under
// both osquery.allow_body_auth_fallback modes:
//
//   - Default mode (flag=true): the Authorization: NodeKey header is ignored
//     entirely. Legacy body-based auth is the sole authenticator. The
//     pre-auth middleware is NOT installed in this mode.
//   - Strict mode (flag=false): the Authorization: NodeKey header is
//     required. Body-based auth is not consulted. Pre-auth rejects
//     absent/invalid headers before the body is read.
//
// See unit tests in osquery_header_auth_test.go for isolated middleware
// coverage.
func (s *integrationTestSuite) TestIntegrationsOsqueryHeaderAuth() {
	t := s.T()
	ctx := context.Background()

	// Create a host with a known node_key.
	nodeKey := t.Name() + "-nodekey"
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(nodeKey),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + ".local",
		Platform:        "linux",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Helper: build a minimal valid submit-logs body with the given node_key.
	makeLogBody := func(bodyNodeKey string) []byte {
		body, err := json.Marshal(submitLogsRequest{
			NodeKey: bodyNodeKey,
			LogType: "status",
			Data:    []json.RawMessage{json.RawMessage(`{}`)},
		})
		require.NoError(t, err)
		return body
	}

	// Helper: assert the 401 response body contains node_invalid:true.
	assertNodeInvalid := func(resp *http.Response) {
		t.Helper()
		defer resp.Body.Close()
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, true, body["node_invalid"], "response should have node_invalid:true; got: %v", body)
	}

	// --------------------------------------------------------------------
	// Default mode: allow_body_auth_fallback=true (the suite's default).
	// The pre-auth middleware is NOT installed. The Authorization header
	// is ignored entirely; body-based auth is the sole authenticator.
	// --------------------------------------------------------------------

	t.Run("default: body node_key alone authenticates", func(t *testing.T) {
		resp := s.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(nodeKey),
			http.StatusOK, map[string]string{})
		defer resp.Body.Close()
	})

	t.Run("default: header is ignored — invalid header + valid body still 200", func(t *testing.T) {
		resp := s.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(nodeKey),
			http.StatusOK, map[string]string{"Authorization": "NodeKey bogus-token"})
		defer resp.Body.Close()
	})

	t.Run("default: header is ignored — valid header + invalid body still 401", func(t *testing.T) {
		resp := s.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody("bogus-body-token"),
			http.StatusUnauthorized, map[string]string{"Authorization": "NodeKey " + nodeKey})
		assertNodeInvalid(resp)
	})

	t.Run("default: invalid body node_key rejects via legacy auth", func(t *testing.T) {
		resp := s.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody("bogus-body-token"),
			http.StatusUnauthorized, map[string]string{})
		assertNodeInvalid(resp)
	})

	t.Run("default: enroll endpoint unaffected", func(t *testing.T) {
		// /api/osquery/enroll uses enroll_secret, not node_key. The header
		// pre-auth wiring never touches this route under any flag value.
		body := fmt.Sprintf(`{"enroll_secret":"nosuchsecret","host_identifier":"%s"}`, t.Name())
		resp := s.DoRawWithHeaders("POST", "/api/osquery/enroll", []byte(body),
			http.StatusUnauthorized, map[string]string{"Authorization": "NodeKey bogus"})
		defer resp.Body.Close()
	})

	t.Run("default: yara endpoint uses body auth", func(t *testing.T) {
		body := fmt.Appendf(nil, `{"node_key":%q}`, nodeKey)
		// 404 = body auth succeeded, rule not found in datastore.
		resp := s.DoRawWithHeaders("POST", "/api/osquery/yara/no-such-rule", body,
			http.StatusNotFound, map[string]string{})
		resp.Body.Close()

		resp2 := s.DoRawWithHeaders("POST", "/api/osquery/yara/no-such-rule",
			[]byte(`{"node_key":"bogus"}`),
			http.StatusUnauthorized, map[string]string{})
		assertNodeInvalid(resp2)
	})

	// --------------------------------------------------------------------
	// Strict mode: allow_body_auth_fallback=false. Spin up a second server
	// on the same DB so we exercise the pre-auth wiring through the real
	// kithttp stack.
	// --------------------------------------------------------------------

	t.Run("strict mode (allow_body_auth_fallback=false)", func(t *testing.T) {
		cfg := config.TestConfig()
		cfg.Osquery.AllowBodyAuthFallback = false

		_, customServer := RunServerForTestsWithDS(t, s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		t.Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		t.Run("valid NodeKey header → 200", func(t *testing.T) {
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(""),
				http.StatusOK, map[string]string{"Authorization": "NodeKey " + nodeKey})
			defer resp.Body.Close()
		})

		t.Run("case-insensitive scheme accepted", func(t *testing.T) {
			for _, scheme := range []string{"nodekey", "NODEKEY", "NoDeKeY"} {
				resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(""),
					http.StatusOK, map[string]string{"Authorization": scheme + " " + nodeKey})
				resp.Body.Close()
			}
		})

		t.Run("invalid NodeKey header → 401", func(t *testing.T) {
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(""),
				http.StatusUnauthorized, map[string]string{"Authorization": "NodeKey bogus-token"})
			assertNodeInvalid(resp)
		})

		t.Run("absent header → 401 (no body fallback)", func(t *testing.T) {
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(nodeKey),
				http.StatusUnauthorized, map[string]string{})
			assertNodeInvalid(resp)
		})

		t.Run("wrong scheme → 401", func(t *testing.T) {
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", makeLogBody(nodeKey),
				http.StatusUnauthorized, map[string]string{"Authorization": "Bearer " + nodeKey})
			assertNodeInvalid(resp)
		})

		t.Run("invalid header rejects before body is read", func(t *testing.T) {
			// Send a body well above the global request size limit. If pre-auth
			// rejects before reading the body, we get a clean 401. If the body
			// were read, we'd get a 413 PayloadTooLarge.
			padSize := int(platform_http.MaxRequestBodySize) * 2
			var buf bytes.Buffer
			buf.WriteByte('{')
			buf.WriteString(`"node_key":"`)
			buf.WriteString(nodeKey)
			buf.WriteString(`","log_type":"status","data":["`)
			buf.WriteString(strings.Repeat("A", padSize))
			buf.WriteString(`"]}`)

			resp := ts.DoRawWithHeaders("POST", "/api/osquery/log", buf.Bytes(),
				http.StatusUnauthorized, map[string]string{"Authorization": "NodeKey bogus"})
			assertNodeInvalid(resp)
		})

		t.Run("/distributed/write also strict-gated", func(t *testing.T) {
			body, err := json.Marshal(map[string]any{
				"node_key": "",
				"queries":  map[string]any{},
				"statuses": map[string]any{},
			})
			require.NoError(t, err)

			resp := ts.DoRawWithHeaders("POST", "/api/osquery/distributed/write", body,
				http.StatusOK, map[string]string{"Authorization": "NodeKey " + nodeKey})
			resp.Body.Close()

			resp2 := ts.DoRawWithHeaders("POST", "/api/osquery/distributed/write", body,
				http.StatusUnauthorized, map[string]string{"Authorization": "NodeKey bogus"})
			assertNodeInvalid(resp2)
		})

		t.Run("/carve/block strict-gated", func(t *testing.T) {
			// In strict mode the pre-auth wrapper on /carve/block requires
			// a valid header before the streaming parser runs. Absent
			// header → 401 even though session_id+request_id auth is the
			// streaming parser's mechanism.
			body := `{"block_id":0,"session_id":"does-not-exist","request_id":"x","data":"aGk="}`
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/carve/block", []byte(body),
				http.StatusUnauthorized, map[string]string{})
			assertNodeInvalid(resp)
		})

		t.Run("/yara/{name} exempt from strict mode (uses body auth)", func(t *testing.T) {
			// 404 means body auth succeeded and the request reached the
			// service layer, where the rule lookup fails. If pre-auth had
			// applied to /yara, an absent header would have produced 401
			// with "missing or malformed Authorization header".
			body := fmt.Appendf(nil, `{"node_key":%q}`, nodeKey)
			resp := ts.DoRawWithHeaders("POST", "/api/osquery/yara/no-such-rule", body,
				http.StatusNotFound, map[string]string{})
			respBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err)
			assert.NotContains(t, string(respBody), "missing or malformed Authorization header")

			resp2 := ts.DoRawWithHeaders("POST", "/api/osquery/yara/no-such-rule",
				[]byte(`{"node_key":"bogus"}`),
				http.StatusUnauthorized, map[string]string{})
			assertNodeInvalid(resp2)
		})
	})
}
