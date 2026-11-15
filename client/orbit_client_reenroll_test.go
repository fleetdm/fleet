package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
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

func TestGetNodeKeyOrEnrollFailedEnrollKeepsKey(t *testing.T) {
	setReenrollGracePeriod(t, 0)
	t.Setenv("FLEETD_ENROLL_RETRY_INTERVAL", "1ms")
	_, nodeKeyPath := newNodeKeyFile(t, "old-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
	oc.noteUnauthenticated()
	_, err := oc.getNodeKeyOrEnroll()
	require.ErrorContains(t, err, "orbit node key enroll failed")
	requireNodeKey(t, nodeKeyPath, "old-key")
	key, err := oc.GetNodeKey()
	require.NoError(t, err)
	require.Equal(t, "old-key", key)
}

func TestWriteNodeKeyFileAtomicReplace(t *testing.T) {
	t.Run("success overwrites existing key and leaves no temp file", func(t *testing.T) {
		dir, nodeKeyPath := newNodeKeyFile(t, "old-key")

		oc := &OrbitClient{nodeKeyFilePath: nodeKeyPath}
		require.NoError(t, oc.writeNodeKeyFile("fresh-key"))
		requireNodeKey(t, nodeKeyPath, "fresh-key")

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1, "temp file should have been renamed into place, not left behind")
	})

	t.Run("failed rename leaves no temp file", func(t *testing.T) {
		// A non-empty directory at the destination makes the rename fail after the temp file was written.
		dir := t.TempDir()
		nodeKeyPath := filepath.Join(dir, "secret-orbit-node-key.txt")
		require.NoError(t, os.Mkdir(nodeKeyPath, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(nodeKeyPath, "keep"), nil, 0o600))

		oc := &OrbitClient{nodeKeyFilePath: nodeKeyPath}
		err := oc.writeNodeKeyFile("fresh-key")
		require.ErrorContains(t, err, "replace orbit node key file")

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1, "temp file should have been cleaned up on failure")
		require.Equal(t, "secret-orbit-node-key.txt", entries[0].Name())
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
		// A refresher that finds only the secret orbit already holds must not restart it.
		oc.SetEnrollSecretRefresher(func() string { return "secret" })
		var restarts int
		oc.receiverUpdateCancelFunc = func() { restarts++ }

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
		require.Zero(t, restarts)
	})

	t.Run("re-enroll rejected with a spent secret picks up a resent one and restarts", func(t *testing.T) {
		setReenrollGracePeriod(t, 0)
		t.Setenv("FLEETD_ENROLL_RETRY_INTERVAL", "1ms")
		_, nodeKeyPath := newNodeKeyFile(t, "existing-key")

		// The enroll endpoint accepts only the secret an administrator resent, as the server does for a host whose one-time
		// secret was used. The authenticated endpoint rejects the node key until orbit has re-enrolled.
		var mu sync.Mutex
		var presented []string
		reenrolled := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			if strings.HasSuffix(r.URL.Path, "/orbit/enroll") {
				var req fleet.EnrollOrbitRequest
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				presented = append(presented, req.EnrollSecret)
				if req.EnrollSecret != "resent" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				reenrolled = true
				writeEnrollResponse(t, w, "new-key")
				return
			}
			if !reenrolled {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(fleet.OrbitConfig{}))
		}))
		defer srv.Close()

		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		var refreshes, rejections, restarts int
		oc.SetEnrollSecretRefresher(func() string {
			refreshes++
			if refreshes == 1 {
				return "" // the resend lands between the first rejected attempt and the retry
			}
			return "resent"
		})
		oc.SetOnEnrollRejected(func() { rejections++ })
		oc.receiverUpdateCancelFunc = func() { restarts++ }

		err := oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.ErrorIs(t, err, ErrUnauthenticated)

		err = oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
		require.NoError(t, err)
		requireNodeKey(t, nodeKeyPath, "new-key")
		mu.Lock()
		require.Equal(t, []string{"secret", "resent"}, presented)
		mu.Unlock()
		require.Equal(t, 1, rejections)
		require.Equal(t, 1, restarts, "osquery holds the old secret until orbit restarts")
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

type keyCheckingServer struct {
	*httptest.Server
	mu      sync.Mutex
	enrolls int
	current string
	seen    []string
}

func newKeyCheckingServer(t *testing.T) *keyCheckingServer {
	t.Helper()
	s := &keyCheckingServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/orbit/enroll") {
			s.enrolls++
			s.current = fmt.Sprintf("key-%d", s.enrolls)
			writeEnrollResponse(t, w, s.current)
			return
		}
		var req fleet.OrbitGetConfigRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		s.seen = append(s.seen, req.OrbitNodeKey)
		if req.OrbitNodeKey == "" || req.OrbitNodeKey != s.current {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.NoError(t, json.NewEncoder(w).Encode(fleet.OrbitConfig{}))
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *keyCheckingServer) enrollCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enrolls
}

func (s *keyCheckingServer) rotate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = "rotated-elsewhere"
}

func getConfig(oc *OrbitClient) error {
	return oc.authenticatedRequest("POST", "/api/fleet/orbit/config", &fleet.OrbitGetConfigRequest{}, &fleet.OrbitConfig{})
}

func TestGetNodeKeyOrEnrollUnwritableNodeKey(t *testing.T) {
	t.Setenv("FLEETD_ENROLL_RETRY_INTERVAL", "1ms")
	missingKeyPath := func(t *testing.T) (dir, path string) {
		dir = filepath.Join(t.TempDir(), "orbit")
		return dir, filepath.Join(dir, constant.OrbitNodeKeyFileName)
	}

	t.Run("enrolls once and requests use the issued key", func(t *testing.T) {
		srv := newKeyCheckingServer(t)
		_, nodeKeyPath := missingKeyPath(t)
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

		err := getConfig(oc)
		require.Equal(t, 1, srv.enrollCount(), "enroll requests")
		require.NoError(t, err)
		require.NoError(t, getConfig(oc))
		require.Equal(t, 1, srv.enrollCount(), "enroll requests")
		key, err := oc.GetNodeKey()
		require.NoError(t, err)
		require.Equal(t, "key-1", key)
		require.NoFileExists(t, nodeKeyPath)
	})

	t.Run("concurrent callers share one enroll", func(t *testing.T) {
		srv := newKeyCheckingServer(t)
		_, nodeKeyPath := missingKeyPath(t)
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

		var wg sync.WaitGroup
		keys := make([]string, 2)
		errs := make([]error, 2)
		for i := range keys {
			wg.Go(func() { keys[i], errs[i] = oc.getNodeKeyOrEnroll() })
		}
		wg.Wait()
		require.Equal(t, 1, srv.enrollCount(), "enroll requests")
		require.NoError(t, errors.Join(errs...))
		require.Equal(t, []string{"key-1", "key-1"}, keys)
	})

	t.Run("a rejected unsaved key re-enrolls once", func(t *testing.T) {
		setReenrollGracePeriod(t, 0)
		srv := newKeyCheckingServer(t)
		_, nodeKeyPath := missingKeyPath(t)
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		require.NoError(t, getConfig(oc))

		srv.rotate()
		require.ErrorIs(t, getConfig(oc), ErrUnauthenticated)
		require.NoError(t, getConfig(oc))
		require.NoError(t, getConfig(oc))
		require.Equal(t, 2, srv.enrollCount())
		srv.mu.Lock()
		require.Equal(t, []string{"key-1", "key-1", "key-2", "key-2"}, srv.seen)
		srv.mu.Unlock()
	})

	t.Run("saves the key once the directory is back", func(t *testing.T) {
		srv := newKeyCheckingServer(t)
		dir, nodeKeyPath := missingKeyPath(t)
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		require.NoError(t, getConfig(oc))

		require.NoError(t, os.Mkdir(dir, 0o700))
		require.NoError(t, getConfig(oc))
		requireNodeKey(t, nodeKeyPath, "key-1")
		require.Equal(t, 1, srv.enrollCount())
	})

	t.Run("requests use the unsaved key over a stale key file", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("needs a directory that chmod makes unwritable")
		}
		setReenrollGracePeriod(t, 0)
		srv := newKeyCheckingServer(t)
		_, nodeKeyPath := newNodeKeyFile(t, "stale-key")
		dir := filepath.Dir(nodeKeyPath)
		require.NoError(t, os.Chmod(dir, 0o500))       // nolint:gosec // G302
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // nolint:gosec // G302
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

		require.ErrorIs(t, getConfig(oc), ErrUnauthenticated)
		for range 3 {
			require.NoError(t, getConfig(oc))
		}
		require.Equal(t, 1, srv.enrollCount())
		key, err := oc.GetNodeKey()
		require.NoError(t, err)
		require.Equal(t, "key-1", key)
		requireNodeKey(t, nodeKeyPath, "stale-key")
	})

	t.Run("deleting a saved key file enrolls once", func(t *testing.T) {
		srv := newKeyCheckingServer(t)
		_, nodeKeyPath := newNodeKeyFile(t, "")
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		require.NoError(t, getConfig(oc))
		requireNodeKey(t, nodeKeyPath, "key-1")

		require.NoError(t, os.Remove(nodeKeyPath))
		require.NoError(t, getConfig(oc))
		require.NoError(t, getConfig(oc))
		require.Equal(t, 2, srv.enrollCount())
		requireNodeKey(t, nodeKeyPath, "key-2")
	})

	t.Run("restart for a refreshed secret waits until the key is saved", func(t *testing.T) {
		srv := newKeyCheckingServer(t)
		dir, nodeKeyPath := missingKeyPath(t)
		oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)
		oc.SetEnrollSecretRefresher(func() string { return "resent" })
		var restarts int
		oc.receiverUpdateCancelFunc = func() { restarts++ }

		require.NoError(t, getConfig(oc))
		require.Zero(t, restarts, "a restart before the key is saved would enroll again")

		require.NoError(t, os.Mkdir(dir, 0o700))
		require.NoError(t, getConfig(oc))
		require.Equal(t, 1, restarts)
		requireNodeKey(t, nodeKeyPath, "key-1")
		require.Equal(t, 1, srv.enrollCount())
	})
}
