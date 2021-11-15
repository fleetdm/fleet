//go:build darwin || linux
// +build darwin linux

package eefleetctl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/urfave/cli/v2"
)

func TestPassphraseHandlerEnvironment(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	testCases := []struct {
		role       string
		passphrase string
	}{
		{role: "root", passphrase: "rootpassphrase"},
		{role: "timestamp", passphrase: "timestamp5#$#@"},
		{role: "snapshot", passphrase: "snapshot$#@"},
		{role: "targets", passphrase: "$#^#$@targets"},
	}

	for _, tt := range testCases {
		t.Run(tt.role, func(t *testing.T) {
			tt := tt
			t.Parallel()

			handler := newPassphraseHandler()
			envKey := fmt.Sprintf("FLEET_%s_PASSPHRASE", strings.ToUpper(tt.role))
			require.NoError(t, os.Setenv(envKey, tt.passphrase))

			passphrase, err := handler.getPassphrase(tt.role, false)
			require.NoError(t, err)
			assert.Equal(t, tt.passphrase, string(passphrase))

			// Should work second time with cache
			passphrase, err = handler.getPassphrase(tt.role, false)
			require.NoError(t, err)
			assert.Equal(t, tt.passphrase, string(passphrase))
		})
	}
}

func TestPassphraseHandlerEmpty(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	handler := newPassphraseHandler()
	require.NoError(t, os.Setenv("FLEET_ROOT_PASSPHRASE", ""))
	_, err := handler.getPassphrase("root", false)
	require.Error(t, err)
}

func setPassphrases(t *testing.T) {
	t.Helper()
	require.NoError(t, os.Setenv("FLEET_ROOT_PASSPHRASE", "root"))
	require.NoError(t, os.Setenv("FLEET_TIMESTAMP_PASSPHRASE", "timestamp"))
	require.NoError(t, os.Setenv("FLEET_TARGETS_PASSPHRASE", "targets"))
	require.NoError(t, os.Setenv("FLEET_SNAPSHOT_PASSPHRASE", "snapshot"))
}

func runUpdatesCommand(args ...string) error {
	app := cli.NewApp()
	app.Commands = []*cli.Command{UpdatesCommand()}
	return app.Run(append([]string{os.Args[0], "updates"}, args...))
}

func TestUpdatesInit(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	// Should fail with already initialized
	require.Error(t, runUpdatesCommand("init", "--path", tmpDir))
}

func TestUpdatesErrorInvalidPassphrase(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	// Should not be able to add with invalid passphrase
	require.NoError(t, os.Setenv("FLEET_SNAPSHOT_PASSPHRASE", "invalid"))
	// Reset the cache that already has correct passwords stored
	passHandler = newPassphraseHandler()
	require.Error(t, runUpdatesCommand("add", "--path", tmpDir, "--target", "anything", "--platform", "windows", "--name", "test", "--version", "1.3.4.7"))
}

func TestUpdatesInitKeysInitializedError(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	// Create an empty "keys" directory
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "keys"), os.ModePerm|os.ModeDir))
	// Should fail with already initialized
	require.Error(t, runUpdatesCommand("init", "--path", tmpDir))
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	st, err := os.Stat(path)
	require.NoError(t, err, "stat should succeed")
	assert.True(t, st.Mode().IsRegular(), "should be regular file: %s", path)
}

func assertVersion(t *testing.T, expected int, versionFunc func() (int, error)) {
	t.Helper()
	actual, err := versionFunc()
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

// Capture stdout while running the updates roots command
func getRoots(t *testing.T, tmpDir string) string {
	t.Helper()

	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	require.NoError(t, runUpdatesCommand("roots", "--path", tmpDir))
	require.NoError(t, w.Close())

	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	// Check output
	var keys []data.Key
	require.NoError(t, json.Unmarshal(out, &keys))
	assert.Greater(t, len(keys[0].IDs()), 0)
	assert.Equal(t, "ed25519", keys[0].Type)

	return string(out)
}

func TestUpdatesIntegration(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	// Run an HTTP server to serve the update metadata
	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(tmpDir, "repository"))))
	defer server.Close()

	roots := getRoots(t, tmpDir)

	// Initialize an update client
	localStore, err := filestore.New(filepath.Join(tmpDir, "tuf-metadata.json"))
	require.NoError(t, err)
	client, err := update.New(update.Options{RootDirectory: tmpDir, ServerURL: server.URL, RootKeys: roots, LocalStore: localStore})
	require.NoError(t, err)
	require.NoError(t, client.UpdateMetadata())
	_, err = client.Lookup("any", "target")
	require.Error(t, err, "lookup should fail before targets added")

	repo, err := openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 1, repo.TargetsVersion)
	assertVersion(t, 1, repo.SnapshotVersion)
	assertVersion(t, 1, repo.TimestampVersion)

	// Add some targets
	// Use the current binary for this test so that it is a binary that is valid for execution on
	// the current system.
	testPath, err := os.Executable()
	require.NoError(t, err)
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "linux", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "macos", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.3.7"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "linux", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "macos", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "windows", "1.3.3.7", "test"))

	// Verify the client can look up and download the updates
	require.NoError(t, client.UpdateMetadata())
	targets, err := client.Targets()
	require.NoError(t, err)
	assert.Len(t, targets, 3)
	_, err = client.Get("test", "1.3.3.7")
	require.NoError(t, err)

	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 4, repo.TargetsVersion)
	assertVersion(t, 4, repo.SnapshotVersion)
	assertVersion(t, 4, repo.TimestampVersion)

	require.NoError(t, runUpdatesCommand("timestamp", "--path", tmpDir))

	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 4, repo.TargetsVersion)
	assertVersion(t, 4, repo.SnapshotVersion)
	assertVersion(t, 5, repo.TimestampVersion)

	// Rotate root
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "root"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 2, repo.RootVersion)
	assertVersion(t, 5, repo.TargetsVersion)
	assertVersion(t, 5, repo.SnapshotVersion)
	assertVersion(t, 6, repo.TimestampVersion)

	// Rotate targets
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "targets"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 3, repo.RootVersion)
	assertVersion(t, 6, repo.TargetsVersion)
	assertVersion(t, 6, repo.SnapshotVersion)
	assertVersion(t, 7, repo.TimestampVersion)

	// Rotate snapshot
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "snapshot"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 4, repo.RootVersion)
	assertVersion(t, 7, repo.TargetsVersion)
	assertVersion(t, 7, repo.SnapshotVersion)
	assertVersion(t, 8, repo.TimestampVersion)

	// Rotate timestamp
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "timestamp"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 5, repo.RootVersion)
	assertVersion(t, 8, repo.TargetsVersion)
	assertVersion(t, 8, repo.SnapshotVersion)
	assertVersion(t, 9, repo.TimestampVersion)

	// Should still be able to add after rotations
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.3.7"))

	// Root key should have changed
	newRoots := getRoots(t, tmpDir)
	assert.NotEqual(t, roots, newRoots)

	// Should still be able to retrieve an update after rotations
	require.NoError(t, client.UpdateMetadata())
	targets, err = client.Targets()
	require.NoError(t, err)
	assert.Len(t, targets, 3)
	// Remove the old copy first
	require.NoError(t, os.RemoveAll(client.LocalPath("test", "1.3.3.7")))
	_, err = client.Get("test", "1.3.3.7")
	require.NoError(t, err)

	// Update client should be able to initialize with new root
	tmpDir = t.TempDir()
	localStore, err = filestore.New(filepath.Join(tmpDir, "tuf-metadata.json"))
	require.NoError(t, err)
	client, err = update.New(update.Options{RootDirectory: tmpDir, ServerURL: server.URL, RootKeys: roots, LocalStore: localStore})
	require.NoError(t, err)
	require.NoError(t, client.UpdateMetadata())
}

func TestCommit(t *testing.T) {
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	initialEntries, err := os.ReadDir(filepath.Join(tmpDir, "repository"))
	require.NoError(t, err)

	initialRoots := getRoots(t, tmpDir)

	commit, _, err := startRotatePseudoTx(tmpDir)
	require.NoError(t, err)

	repo, err := openRepo(tmpDir)
	require.NoError(t, err)

	// Make rotations that change repo
	require.NoError(t, updatesGenKey(repo, "root"))
	require.NoError(t, repo.Sign("root.json"))
	require.NoError(t, repo.SnapshotWithExpires(time.Now().Add(snapshotExpirationDuration)))
	require.NoError(t, repo.TimestampWithExpires(time.Now().Add(timestampExpirationDuration)))
	require.NoError(t, repo.Commit())

	// Assert directory has changed after commit.
	require.NoError(t, commit())
	entries, err := os.ReadDir(filepath.Join(tmpDir, "repository"))
	require.NoError(t, err)
	assert.NotEqual(t, initialEntries, entries)

	_, err = os.Stat(filepath.Join(tmpDir, "repository", ".backup"))
	assert.True(t, os.IsNotExist(err))

	// Roots should have changed.
	roots := getRoots(t, tmpDir)
	assert.NotEqual(t, initialRoots, roots)
}

func TestRollback(t *testing.T) {
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	initialEntries, err := os.ReadDir(filepath.Join(tmpDir, "repository"))
	require.NoError(t, err)

	initialRoots := getRoots(t, tmpDir)

	_, rollback, err := startRotatePseudoTx(tmpDir)
	require.NoError(t, err)

	repo, err := openRepo(tmpDir)
	require.NoError(t, err)

	// Make rotations that change repo
	require.NoError(t, updatesGenKey(repo, "root"))
	require.NoError(t, repo.Sign("root.json"))
	require.NoError(t, repo.SnapshotWithExpires(time.Now().Add(snapshotExpirationDuration)))
	require.NoError(t, repo.TimestampWithExpires(time.Now().Add(timestampExpirationDuration)))
	require.NoError(t, repo.Commit())

	// Assert directory has NOT changed after rollback.
	require.NoError(t, rollback())
	entries, err := os.ReadDir(filepath.Join(tmpDir, "repository"))
	require.NoError(t, err)
	assert.Equal(t, initialEntries, entries)

	_, err = os.Stat(filepath.Join(tmpDir, "repository", ".backup"))
	assert.True(t, os.IsNotExist(err))

	// Roots should NOT have changed.
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	roots := getRoots(t, tmpDir)
	assert.Equal(t, initialRoots, roots)
}
