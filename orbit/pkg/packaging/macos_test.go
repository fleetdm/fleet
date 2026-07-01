package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestWriteMacOSSecret(t *testing.T) {
	t.Run("skips writing when enroll secret is empty", func(t *testing.T) {
		// With --use-system-configuration the enroll secret is empty and is
		// resolved at runtime; writing an empty secret.txt produces confusing
		// keystore errors during ABM enrollment.
		orbitRoot := t.TempDir()
		require.NoError(t, writeMacOSSecret(Options{EnrollSecret: ""}, orbitRoot))
		require.NoFileExists(t, filepath.Join(orbitRoot, constant.OsqueryEnrollSecretFileName))
	})

	t.Run("writes the secret when present", func(t *testing.T) {
		orbitRoot := t.TempDir()
		require.NoError(t, writeMacOSSecret(Options{EnrollSecret: "mysecret"}, orbitRoot))
		path := filepath.Join(orbitRoot, constant.OsqueryEnrollSecretFileName)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "mysecret", string(contents))
	})
}
