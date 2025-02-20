//go:build darwin || linux
// +build darwin linux

package eefleetctl

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/race"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

			handler := newPassphraseHandler()
			envKey := fmt.Sprintf("FLEET_%s_PASSPHRASE", strings.ToUpper(tt.role))
			t.Setenv(envKey, tt.passphrase)

			passphrase, err := handler.getPassphrase(tt.role, false, false)
			require.NoError(t, err)
			assert.Equal(t, tt.passphrase, string(passphrase))

			// Should work second time with cache
			passphrase, err = handler.getPassphrase(tt.role, false, false)
			require.NoError(t, err)
			assert.Equal(t, tt.passphrase, string(passphrase))
		})
	}
}

func TestPassphraseHandlerEmpty(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	handler := newPassphraseHandler()
	t.Setenv("FLEET_ROOT_PASSPHRASE", "")
	_, err := handler.getPassphrase("root", false, false)
	require.Error(t, err)
}

func setPassphrases(t *testing.T) {
	t.Helper()
	t.Setenv("FLEET_ROOT_PASSPHRASE", "root")
	t.Setenv("FLEET_TIMESTAMP_PASSPHRASE", "timestamp")
	t.Setenv("FLEET_TARGETS_PASSPHRASE", "targets")
	t.Setenv("FLEET_SNAPSHOT_PASSPHRASE", "snapshot")
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
	t.Setenv("FLEET_SNAPSHOT_PASSPHRASE", "invalid")
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

func assertVersion(t *testing.T, expected int64, versionFunc func() (int64, error)) {
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

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	// Check output contains the root.json
	var keys map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &keys))
	signed_ := keys["signed"]
	require.NotNil(t, signed_)
	signed, ok := signed_.(map[string]interface{})
	require.True(t, ok)
	keys_ := signed["keys"]
	require.NotNil(t, keys_)
	require.NotEmpty(t, signed["keys"])
	keys, ok = keys_.(map[string]interface{})
	require.True(t, ok)
	require.NotEmpty(t, keys)
	// Get first key (map key is the identifier of the key).
	var key map[string]interface{}
	for _, key_ := range keys {
		key, ok = key_.(map[string]interface{})
		require.True(t, ok)
		break
	}
	require.NotEmpty(t, key)
	require.Equal(t, "ed25519", key["scheme"])
	require.Equal(t, "ed25519", key["keytype"])
	keyval_, ok := key["keyval"].(map[string]interface{})
	require.True(t, ok)
	require.NotEmpty(t, keyval_["public"]) // keyval_["public"] contains the public key.

	return string(out)
}

func compressSingleFile(t *testing.T, filePath, outFilePath string) {
	outf, err := secure.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY, constant.DefaultFileMode)
	require.NoError(t, err)
	defer outf.Close()
	gw := gzip.NewWriter(outf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	inf, err := os.Open(filePath)
	require.NoError(t, err)
	infi, err := inf.Stat()
	require.NoError(t, err)
	err = tw.WriteHeader(
		&tar.Header{
			Name: filepath.Base(filePath),
			Size: infi.Size(),
			Mode: 0o777,
		},
	)
	require.NoError(t, err)
	_, err = io.Copy(tw, inf)
	require.NoError(t, err)
	err = tw.Close()
	require.NoError(t, err)
	err = gw.Close()
	require.NoError(t, err)
	err = outf.Close()
	require.NoError(t, err)
}

func TestIntegrationsUpdates(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	// Run an HTTP server to serve the update metadata
	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(tmpDir, "repository"))))
	t.Cleanup(server.Close)

	roots := getRoots(t, tmpDir)

	// Use the current binary as target for this test so that it is a binary that
	// is valid for execution on the current system.
	testPath, err := os.Executable()
	require.NoError(t, err)

	tarGzFilePath := filepath.Join(filepath.Dir(testPath), "other.app.tar.gz")
	compressSingleFile(t, testPath, tarGzFilePath)

	// Initialize an update client
	localStore, err := filestore.New(filepath.Join(tmpDir, "tuf-metadata.json"))
	require.NoError(t, err)
	updater, err := update.NewUpdater(update.Options{
		RootDirectory: tmpDir,
		ServerURL:     server.URL,
		RootKeys:      roots,
		LocalStore:    localStore,
		Targets: update.Targets{
			"test": update.TargetInfo{
				Platform:   "macos",
				Channel:    "1.3.3.7",
				TargetFile: "test",
			},
			"other": update.TargetInfo{
				Platform:             "macos-app",
				Channel:              "1.3.3.8",
				TargetFile:           "other.app.tar.gz",
				ExtractedExecSubPath: []string{filepath.Base(testPath)},
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, updater.UpdateMetadata())
	_, err = updater.Lookup("any")
	require.Error(t, err, "lookup should fail before targets added")

	repo, err := openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 1, repo.TargetsVersion)
	assertVersion(t, 1, repo.SnapshotVersion)
	assertVersion(t, 1, repo.TimestampVersion)

	// Add some targets
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "linux", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "macos", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", tarGzFilePath, "--platform", "macos-app", "--name", "other", "--version", "1.3.3.8"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "linux", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "macos", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "windows", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "other", "macos-app", "1.3.3.8", "other.app.tar.gz"))

	// Verify the client can look up and download the updates
	require.NoError(t, updater.UpdateMetadata())
	targets, err := updater.Targets()
	require.NoError(t, err)
	assert.Len(t, targets, 4)
	_, err = updater.Get("test")
	require.NoError(t, err)
	other, err := updater.Get("other")
	require.NoError(t, err)
	require.Equal(t, filepath.Base(other.ExecPath), filepath.Base(testPath))

	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 5, repo.TargetsVersion)
	assertVersion(t, 5, repo.SnapshotVersion)
	assertVersion(t, 5, repo.TimestampVersion)

	require.NoError(t, runUpdatesCommand("timestamp", "--path", tmpDir))

	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 1, repo.RootVersion)
	assertVersion(t, 5, repo.TargetsVersion)
	assertVersion(t, 5, repo.SnapshotVersion)
	assertVersion(t, 6, repo.TimestampVersion)

	// Rotate root
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "root"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 2, repo.RootVersion)
	assertVersion(t, 6, repo.TargetsVersion)
	assertVersion(t, 6, repo.SnapshotVersion)
	assertVersion(t, 7, repo.TimestampVersion)

	// Rotate targets
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "targets"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 3, repo.RootVersion)
	assertVersion(t, 7, repo.TargetsVersion)
	assertVersion(t, 7, repo.SnapshotVersion)
	assertVersion(t, 8, repo.TimestampVersion)

	// Rotate snapshot
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "snapshot"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 4, repo.RootVersion)
	assertVersion(t, 8, repo.TargetsVersion)
	assertVersion(t, 8, repo.SnapshotVersion)
	assertVersion(t, 9, repo.TimestampVersion)

	// Rotate timestamp
	require.NoError(t, runUpdatesCommand("rotate", "--path", tmpDir, "timestamp"))
	repo, err = openRepo(tmpDir)
	require.NoError(t, err)
	assertVersion(t, 5, repo.RootVersion)
	assertVersion(t, 9, repo.TargetsVersion)
	assertVersion(t, 9, repo.SnapshotVersion)
	assertVersion(t, 10, repo.TimestampVersion)

	// Should still be able to add after rotations
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.3.7"))

	// Root key should have changed
	newRoots := getRoots(t, tmpDir)
	assert.NotEqual(t, roots, newRoots)

	// Should still be able to retrieve an update after rotations
	require.NoError(t, updater.UpdateMetadata())
	targets, err = updater.Targets()
	require.NoError(t, err)
	assert.Len(t, targets, 4)
	// Remove the old test copy first
	p, err := updater.ExecutableLocalPath("test")
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(p))
	_, err = updater.Get("test")
	require.NoError(t, err)
	// Remove the old other copy first
	o, err := updater.Get("other")
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(filepath.Join(filepath.Dir(o.ExecPath), "other.app.tar.gz")))
	require.NoError(t, os.RemoveAll(filepath.Join(filepath.Dir(o.ExecPath), filepath.Base(testPath))))
	o2, err := updater.Get("other")
	require.NoError(t, err)
	require.Equal(t, o, o2)
	_, err = os.Stat(o2.ExecPath)
	require.NoError(t, err)

	// Update client should be able to initialize with new root
	tmpDir = t.TempDir()
	localStore, err = filestore.New(filepath.Join(tmpDir, "tuf-metadata.json"))
	require.NoError(t, err)
	updater, err = update.NewUpdater(update.Options{RootDirectory: tmpDir, ServerURL: server.URL, RootKeys: roots, LocalStore: localStore})
	require.NoError(t, err)
	require.NoError(t, updater.UpdateMetadata())
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
	require.NoError(t, repo.SnapshotWithExpires(time.Now().Add(mustParseDuration(snapshotExpirationDuration))))
	require.NoError(t, repo.TimestampWithExpires(time.Now().Add(mustParseDuration(timestampExpirationDuration))))
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
	require.NoError(t, repo.SnapshotWithExpires(time.Now().Add(mustParseDuration(snapshotExpirationDuration))))
	require.NoError(t, repo.TimestampWithExpires(time.Now().Add(mustParseDuration(timestampExpirationDuration))))
	require.NoError(t, repo.Commit())

	// Assert directory has NOT changed after rollback.
	require.NoError(t, rollback())
	entries, err := os.ReadDir(filepath.Join(tmpDir, "repository"))
	require.NoError(t, err)
	assert.Equal(t, initialEntries, entries)

	_, err = os.Stat(filepath.Join(tmpDir, "repository", ".backup"))
	assert.True(t, os.IsNotExist(err))

	// Roots should NOT have changed.
	_, err = openRepo(tmpDir)
	require.NoError(t, err)
	roots := getRoots(t, tmpDir)
	assert.Equal(t, initialRoots, roots)
}

// TestIntegrationsUpdatesExpiredSignatures is used to test the expected behavior
// of go-tuf@v0.5.2 methods client.Update and client.Target when signatures are expired (their
// behavior depends on which of the roles has the expired signature).
func TestIntegrationsUpdatesExpiredSignatures(t *testing.T) {
	// Not t.Parallel() due to modifications to environment and global variables.
	if race.Enabled {
		// Because execution is slower and thus sleep time would need to be higher.
		t.Skip("Skipping test when race is enabled")
	}

	setPassphrases(t)
	const timeToExpire = 5 * time.Second

	for _, tc := range []struct {
		name                      string
		overrideKeyExpirationFunc func(t *testing.T)
		updateMetadataFails       bool
		lookupFails               bool
	}{
		{
			name: "root expired",
			overrideKeyExpirationFunc: func(t *testing.T) {
				oldKeyExpirationDuration := keyExpirationDuration_
				t.Cleanup(func() {
					keyExpirationDuration_ = oldKeyExpirationDuration
				})
				keyExpirationDuration_ = timeToExpire
			},
			// When the root signature is expired both client.Update fails and client.Target fail.
			updateMetadataFails: true,
			lookupFails:         true,
		},
		{
			name: "snapshot expired",
			overrideKeyExpirationFunc: func(t *testing.T) {
				oldKeyExpirationDuration := snapshotExpirationDuration_
				t.Cleanup(func() {
					snapshotExpirationDuration_ = oldKeyExpirationDuration
				})
				snapshotExpirationDuration_ = timeToExpire
			},
			// When the snapshot signature is expired client.Update does not fail and client.Target does fail.
			updateMetadataFails: false,
			lookupFails:         true,
		},
		{
			name: "targets expired",
			overrideKeyExpirationFunc: func(t *testing.T) {
				oldKeyExpirationDuration := targetsExpirationDuration_
				t.Cleanup(func() {
					targetsExpirationDuration_ = oldKeyExpirationDuration
				})
				targetsExpirationDuration_ = timeToExpire
			},
			// When the targets signature is expired client.Update does not fail and client.Target does fail.
			updateMetadataFails: false,
			lookupFails:         true,
		},
		{
			name: "timestamp expired",
			overrideKeyExpirationFunc: func(t *testing.T) {
				oldKeyExpirationDuration := timestampExpirationDuration_
				t.Cleanup(func() {
					timestampExpirationDuration_ = oldKeyExpirationDuration
				})
				timestampExpirationDuration_ = timeToExpire
			},
			// When the timestamp signature is expired client.Update fails and client.Target does not fail.
			updateMetadataFails: true,
			lookupFails:         false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.overrideKeyExpirationFunc(t)

			tmpDir := t.TempDir()
			err := runUpdatesCommand("init", "--path", tmpDir)
			require.NoError(t, err)

			roots := getRoots(t, tmpDir)

			// Use the current binary as target for this test so that it is a binary that
			// is valid for execution on the current system.
			testPath, err := os.Executable()
			require.NoError(t, err)

			// Add a dummy target
			require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "macos", "--name", "test", "--version", "1.3.3.7"))
			assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "macos", "1.3.3.7", "test"))

			// Run an HTTP server to serve the update metadata
			server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(tmpDir, "repository"))))
			t.Cleanup(server.Close)

			// Initialize an update client
			localStore, err := filestore.New(filepath.Join(tmpDir, "tuf-metadata.json"))
			require.NoError(t, err)
			updater, err := update.NewUpdater(update.Options{
				RootDirectory: tmpDir,
				ServerURL:     server.URL,
				RootKeys:      roots,
				LocalStore:    localStore,
				Targets: update.Targets{
					"test": update.TargetInfo{
						Platform:   "macos",
						Channel:    "1.3.3.7",
						TargetFile: "test",
					},
				},
			})
			require.NoError(t, err)
			err = updater.UpdateMetadata()
			require.NoError(t, err)

			time.Sleep(timeToExpire + 1*time.Second)

			// Expect UpdateMetadata (client.Update) to fail when the signature has expired.
			err = updater.UpdateMetadata()
			if tc.updateMetadataFails {
				require.Error(t, err)
				require.True(t, update.IsExpiredErr(err))
			} else {
				require.NoError(t, err)
			}

			// Expect Lookup (client.Target) to fail when the signature has expired.
			_, err = updater.Lookup("test")
			if tc.lookupFails {
				require.Error(t, err)
				require.True(t, update.IsExpiredErr(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
