package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReenrollTestClient(t *testing.T, serverURL, nodeKeyPath string) *OrbitClient {
	t.Helper()
	bc, err := NewBaseClient(serverURL, true, "", "", nil, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	return &OrbitClient{
		BaseClient:      bc,
		nodeKeyFilePath: nodeKeyPath,
		enrollSecret:    "secret",
		hostInfo:        fleet.OrbitHostInfo{HardwareUUID: "uuid-1", Platform: "linux"},
	}
}

// newNodeKeyFile creates a temp dir containing a node key file with the given contents and returns
// both. The dir is returned so tests can assert no stray temp files are left behind.
func newNodeKeyFile(t *testing.T, contents string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "secret-orbit-node-key.txt")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return dir, path
}

func requireNodeKey(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

// writeEnrollResponse writes a successful enroll response handing back nodeKey.
func writeEnrollResponse(t *testing.T, w http.ResponseWriter, nodeKey string) {
	t.Helper()
	assert.NoError(t, json.NewEncoder(w).Encode(fleet.EnrollOrbitResponse{OrbitNodeKey: nodeKey}))
}

// setReenrollGracePeriod temporarily overrides the package-level grace period for the duration of
// the test, restoring it via t.Cleanup. The grace period is package state, so tests using it must
// not run in parallel.
func setReenrollGracePeriod(t *testing.T, d time.Duration) {
	t.Helper()
	orig := unauthenticatedReenrollGracePeriod
	unauthenticatedReenrollGracePeriod = d
	t.Cleanup(func() { unauthenticatedReenrollGracePeriod = orig })
}

func TestGetNodeKeyOrEnrollEmptyFileEnrolls(t *testing.T) {
	// An empty/whitespace-only file must be treated like a missing one and trigger enrollment
	_, nodeKeyPath := newNodeKeyFile(t, "  \n")

	var enrollCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/orbit/enroll") {
			enrollCalls++
			writeEnrollResponse(t, w, "new-key")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
	key, err := oc.getNodeKeyOrEnroll()
	require.NoError(t, err)
	require.Equal(t, "new-key", key)
	require.Equal(t, 1, enrollCalls)
	requireNodeKey(t, nodeKeyPath, "new-key")
}

func TestGetNodeKeyOrEnrollValidFileReturnsKeyWithoutEnrolling(t *testing.T) {
	_, nodeKeyPath := newNodeKeyFile(t, "existing-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when a valid node key exists, got %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
	key, err := oc.getNodeKeyOrEnroll()
	require.NoError(t, err)
	require.Equal(t, "existing-key", key)
}

func TestEnrollAndWriteNodeKeyFileAtomicReplace(t *testing.T) {
	t.Run("success overwrites existing key and leaves no temp file", func(t *testing.T) {
		dir, nodeKeyPath := newNodeKeyFile(t, "old-key")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeEnrollResponse(t, w, "fresh-key")
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		key, err := oc.enrollAndWriteNodeKeyFile()
		require.NoError(t, err)
		require.Equal(t, "fresh-key", key)
		requireNodeKey(t, nodeKeyPath, "fresh-key")

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1, "temp file should have been renamed into place, not left behind")
	})

	t.Run("failed enroll preserves existing key and leaves no temp file", func(t *testing.T) {
		dir, nodeKeyPath := newNodeKeyFile(t, "old-key")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		_, err := oc.enrollAndWriteNodeKeyFile()
		require.Error(t, err)

		// acquire-then-replace: the existing key must survive a failed enroll.
		requireNodeKey(t, nodeKeyPath, "old-key")

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1, "temp file should have been cleaned up on failure")
	})
}

// TestNoteUnauthenticated covers the 401 debounce state machine directly.
func TestNoteUnauthenticated(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		setReenrollGracePeriod(t, 30*time.Second)
		oc := &OrbitClient{}

		// First 401 starts the streak with zero elapsed time.
		reenroll, waited := oc.noteUnauthenticated()
		require.False(t, reenroll)
		require.Zero(t, waited)
		require.False(t, oc.reenrollForced())

		// A success resets the streak (the transient 401 recovered).
		oc.clearReenrollState()
		require.True(t, oc.unauthenticatedSince.IsZero())

		// A fresh 401 after the reset starts a new streak from zero, not from the earlier one.
		reenroll, waited = oc.noteUnauthenticated()
		require.False(t, reenroll)
		require.Zero(t, waited)

		// Advance the fake clock past the grace period; the next 401 then arms re-enroll.
		time.Sleep(31 * time.Second)
		reenroll, waited = oc.noteUnauthenticated()
		require.True(t, reenroll, "a 401 streak older than the grace period should arm re-enroll")
		require.Equal(t, 31*time.Second, waited)
		require.True(t, oc.reenrollForced())
	})
}

func TestAuthenticatedRequest401Debounce(t *testing.T) {
	t.Run("single 401 within grace does not delete key or re-enroll", func(t *testing.T) {
		setReenrollGracePeriod(t, time.Hour)
		_, nodeKeyPath := newNodeKeyFile(t, "existing-key")

		var enrollCalls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/orbit/enroll") {
				enrollCalls++
				writeEnrollResponse(t, w, "new-key")
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		err := oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.ErrorIs(t, err, ErrUnauthenticated)
		require.False(t, oc.reenrollForced(), "a single 401 within the grace period should not arm re-enroll")
		require.Equal(t, 0, enrollCalls)
		requireNodeKey(t, nodeKeyPath, "existing-key") // not deleted on a transient 401
	})

	t.Run("sustained 401 arms re-enroll and next request replaces the key", func(t *testing.T) {
		setReenrollGracePeriod(t, 0) // any 401 immediately exceeds the grace period

		_, nodeKeyPath := newNodeKeyFile(t, "existing-key")

		var mu sync.Mutex
		rejectAuthed := true
		var enrollCalls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			if strings.HasSuffix(r.URL.Path, "/orbit/enroll") {
				enrollCalls++
				writeEnrollResponse(t, w, "new-key")
				return
			}
			if rejectAuthed {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(fleet.OrbitConfig{}))
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

		// First request: 401 -> re-enroll armed, but the existing key is NOT deleted yet.
		err := oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.ErrorIs(t, err, ErrUnauthenticated)
		require.True(t, oc.reenrollForced())
		requireNodeKey(t, nodeKeyPath, "existing-key")

		// Stop rejecting the authenticated endpoint so the post-enroll request succeeds.
		mu.Lock()
		rejectAuthed = false
		mu.Unlock()

		// Second request: getNodeKeyOrEnroll sees the armed re-enroll, enrolls, overwrites the key, and the request then succeeds.
		err = oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.NoError(t, err)
		mu.Lock()
		gotEnrollCalls := enrollCalls
		mu.Unlock()
		require.GreaterOrEqual(t, gotEnrollCalls, 1)
		require.False(t, oc.reenrollForced(), "re-enroll state should be cleared after success")
		requireNodeKey(t, nodeKeyPath, "new-key")
	})

	t.Run("host identity cert is removed only after the grace period, not on the first 401", func(t *testing.T) {
		setReenrollGracePeriod(t, time.Hour)
		dir, nodeKeyPath := newNodeKeyFile(t, "existing-key")
		certPath := filepath.Join(dir, "host-identity.crt")
		require.NoError(t, os.WriteFile(certPath, []byte("cert"), 0o600))

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		oc.hostIdentityCertPath = certPath
		var restartCalls int
		oc.receiverUpdateCancelFunc = func() { restartCalls++ }

		err := oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.ErrorIs(t, err, ErrUnauthenticated)
		// Within the grace period the cert must be preserved and no restart triggered.
		require.FileExists(t, certPath)
		require.Equal(t, 0, restartCalls)

		// Once 401s have persisted past the grace period, the cert is removed and a restart fires.
		setReenrollGracePeriod(t, 0)
		err = oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.ErrorIs(t, err, ErrUnauthenticated)
		require.NoFileExists(t, certPath)
		require.Equal(t, 1, restartCalls)
	})
}
