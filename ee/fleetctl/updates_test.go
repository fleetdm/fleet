package eefleetctl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestUpdatesIntegration(t *testing.T) {
	// Not t.Parallel() due to modifications to environment.
	tmpDir := t.TempDir()

	setPassphrases(t)

	require.NoError(t, runUpdatesCommand("init", "--path", tmpDir))

	// Capture stdout while running the updates roots command
	func() {
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
		require.Len(t, keys, 1)
		assert.Greater(t, len(keys[0].IDs()), 0)
		assert.Equal(t, "ed25519", keys[0].Type)
	}()

	testPath := filepath.Join(tmpDir, "test")
	require.NoError(t, ioutil.WriteFile(testPath, []byte("test"), os.ModePerm))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "linux", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "macos", "--name", "test", "--version", "1.3.3.7"))
	require.NoError(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.3.7"))

	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "linux", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "macos", "1.3.3.7", "test"))
	assertFileExists(t, filepath.Join(tmpDir, "repository", "targets", "test", "windows", "1.3.3.7", "test"))

	require.NoError(t, runUpdatesCommand("timestamp", "--path", tmpDir))

	// Should not be able to add with invalid passphrase
	require.NoError(t, os.Setenv("FLEET_SNAPSHOT_PASSPHRASE", "invalid"))
	// Reset the cache that already has correct passwords stored
	passHandler = newPassphraseHandler()
	require.Error(t, runUpdatesCommand("add", "--path", tmpDir, "--target", testPath, "--platform", "windows", "--name", "test", "--version", "1.3.4.7"))
}
