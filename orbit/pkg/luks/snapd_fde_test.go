//go:build linux

package luks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the exact /v2/system-volumes request/response contract the
// socket client codes against (confirmed from canonical/snapd master). If snapd
// changes the contract, update both this test and snapd_fde.go together.

func decodeAction(t *testing.T, r *http.Request) recoveryKeyActionRequest {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	var req recoveryKeyActionRequest
	assert.NoError(t, json.Unmarshal(body, &req))
	return req
}

// writeSystemInfo returns a canned system-info response with the given snapd
// version. Test handlers should route GET /v2/system-info to this so the
// version preflight in ensureFleetRecoveryKey sees a version that permits the
// rest of the flow to run.
func writeSystemInfo(w http.ResponseWriter, version string) {
	_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"version":"` + version + `"}}`))
}

func TestEnsureFleetRecoveryKeyViaSocket(t *testing.T) {
	const wantKey = "55055-39320-64491-48436-47667-15525-36879-32875"
	var sawGenerate, sawAdd, sawCheck bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/v2/changes/9" {
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"ready":true,"status":"Done"}}`))
			return
		}

		if r.URL.Path == "/v2/system-info" {
			writeSystemInfo(w, "2.75.0")
			return
		}

		assert.Equal(t, "/v2/system-volumes", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		req := decodeAction(t, r)

		switch req.Action {
		case actionGenerateRecoveryKey:
			sawGenerate = true
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"recovery-key":"` + wantKey + `","key-id":"kid-1"}}`))
		case actionAddRecoveryKey:
			sawAdd = true
			assert.Equal(t, "kid-1", req.KeyID)
			if assert.Len(t, req.Keyslots, 1) {
				assert.Equal(t, FleetRecoveryKeyName, req.Keyslots[0].Name)
				assert.Empty(t, req.Keyslots[0].ContainerRole, "name-only keyslot targets all system containers")
			}
			_, _ = w.Write([]byte(`{"type":"async","status-code":202,"change":"9"}`))
		case actionCheckRecoveryKey:
			sawCheck = true
			assert.Equal(t, wantKey, req.RecoveryKey)
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":null}`))
		default:
			t.Errorf("unexpected action %q", req.Action)
		}
	}))
	defer srv.Close()

	fde := &snapdSocketFDE{client: newTestSnapdClient(srv)}
	key, err := fde.ensureFleetRecoveryKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, wantKey, key)
	assert.True(t, sawGenerate, "generate-recovery-key was called")
	assert.True(t, sawAdd, "add-recovery-key was called")
	assert.True(t, sawCheck, "check-recovery-key was called")
}

func TestEnsureFleetRecoveryKeyViaSocketFallsBackToReplace(t *testing.T) {
	var sawReplace bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/changes/9" {
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"ready":true,"status":"Done"}}`))
			return
		}
		if r.URL.Path == "/v2/system-info" {
			writeSystemInfo(w, "2.75.0")
			return
		}
		req := decodeAction(t, r)
		switch req.Action {
		case actionGenerateRecoveryKey:
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"recovery-key":"11111-22222","key-id":"kid-2"}}`))
		case actionAddRecoveryKey:
			// Simulate a slot that already exists.
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"type":"error","status-code":409,"result":{"message":"keyslot already exists"}}`))
		case actionReplaceRecoveryKey:
			sawReplace = true
			assert.Equal(t, "kid-2", req.KeyID)
			_, _ = w.Write([]byte(`{"type":"async","status-code":202,"change":"9"}`))
		case actionCheckRecoveryKey:
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":null}`))
		default:
			t.Errorf("unexpected action %q", req.Action)
		}
	}))
	defer srv.Close()

	fde := &snapdSocketFDE{client: newTestSnapdClient(srv)}
	key, err := fde.ensureFleetRecoveryKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "11111-22222", key)
	assert.True(t, sawReplace, "fell back to replace-recovery-key when add reported a conflict")
}

func TestEnsureFleetRecoveryKeyReportsUnsupportedFDEState(t *testing.T) {
	// Regression: on some hosts snapd is new enough to expose the endpoint
	// but reports "this action is not supported on this system" because the
	// FDE state is indeterminate (`ubuntu-fde` LUKS2 tokens exist but snapd
	// is not the authoritative manager). The error message must call that
	// out and point at snap-tpmctl so the admin has somewhere to look.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/system-info" {
			writeSystemInfo(w, "2.75.0")
			return
		}
		req := decodeAction(t, r)
		if req.Action == actionGenerateRecoveryKey {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"error","status-code":400,"result":{"message":"this action is not supported on this system"}}`))
			return
		}
		t.Errorf("did not expect action %q after generate-recovery-key failed", req.Action)
	}))
	defer srv.Close()

	fde := &snapdSocketFDE{client: newTestSnapdClient(srv)}
	_, err := fde.ensureFleetRecoveryKey(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "snap-tpmctl status", "error points the admin at the diagnostic command")
	assert.Contains(t, err.Error(), "indeterminate", "error names the state operators will see")
	assert.Contains(t, err.Error(), "not supported", "underlying snapd message is preserved for debugging")
}

func TestEnsureFleetRecoveryKeyRejectsOldSnapd(t *testing.T) {
	// Regression: snapd < 2.74 has the /v2/system-volumes endpoint but rejects
	// generate-recovery-key with 400 "this action is not supported on this
	// system". Preflight the version so the operator sees a clear
	// upgrade-required message instead of the raw snapd 400.
	cases := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "2.73 is too old", version: "2.73.4", wantErr: true},
		{name: "2.68 with ubuntu suffix is too old", version: "2.68.5+24.04", wantErr: true},
		{name: "2.74.0 exactly is accepted", version: "2.74.0", wantErr: false},
		{name: "2.75.1 is accepted", version: "2.75.1", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sawGenerate bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v2/system-info" {
					writeSystemInfo(w, tc.version)
					return
				}
				if r.URL.Path == "/v2/changes/9" {
					_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"ready":true,"status":"Done"}}`))
					return
				}
				req := decodeAction(t, r)
				switch req.Action {
				case actionGenerateRecoveryKey:
					sawGenerate = true
					_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"recovery-key":"55055-39320","key-id":"kid-3"}}`))
				case actionAddRecoveryKey:
					_, _ = w.Write([]byte(`{"type":"async","status-code":202,"change":"9"}`))
				case actionCheckRecoveryKey:
					_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":null}`))
				default:
					t.Errorf("unexpected action %q", req.Action)
				}
			}))
			defer srv.Close()

			fde := &snapdSocketFDE{client: newTestSnapdClient(srv)}
			_, err := fde.ensureFleetRecoveryKey(context.Background())
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.version, "error names the too-old version")
				assert.Contains(t, err.Error(), snapdMinVersion, "error names the required minimum version")
				assert.False(t, sawGenerate, "preflight fails before generate-recovery-key is called")
			} else {
				require.NoError(t, err)
				assert.True(t, sawGenerate, "flow proceeds past preflight on supported versions")
			}
		})
	}
}
