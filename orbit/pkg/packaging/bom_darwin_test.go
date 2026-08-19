//go:build darwin

package packaging

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteBomMatchesMkbom builds a BOM two ways for the same tree -- via the
// native mkbom pipeline used by xarBom (mkbom -> lsbom -> 0/80 transform ->
// mkbom -i) and via the pure-Go writeBom -- then asserts lsbom reports an
// identical manifest for both. This is the functional-equivalence bar for the
// mkbom replacement.
func TestWriteBomMatchesMkbom(t *testing.T) {
	for _, tool := range []string{"mkbom", "lsbom"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}

	root := t.TempDir()
	// A representative tree: nested dirs, an empty file, a binary-ish file, a
	// name with a space, and varied permissions.
	writeFile(t, filepath.Join(root, "opt", "orbit", "secret.txt"), []byte("SUPERSECRET"), 0o600)
	writeFile(t, filepath.Join(root, "opt", "orbit", "osquery.flags"), []byte{}, 0o600)
	writeFile(t, filepath.Join(root, "opt", "orbit", "bin", "orbit"), []byte("\x7fELF binary-ish payload"), 0o755)
	writeFile(t, filepath.Join(root, "Library", "LaunchDaemons", "com.fleetdm.orbit.plist"), []byte("<plist/>\n"), 0o644)
	writeFile(t, filepath.Join(root, "opt", "orbit", "bin", "desktop", "Fleet Desktop.app", "Contents", "Info.plist"), []byte("<x/>"), 0o644)

	// Reference BOM via the native pipeline (mirrors xarBom's darwin branch).
	refBom := filepath.Join(root, "..", "ref.bom")
	inBom := filepath.Join(t.TempDir(), "inBom")
	require.NoError(t, exec.Command("mkbom", root, inBom).Run()) //nolint:gosec
	lsOut, err := exec.Command("lsbom", inBom).Output()          //nolint:gosec
	require.NoError(t, err)
	// Rewrite ownership to root/admin (0/80), as the old darwin pipeline did.
	transformed := regexp.MustCompile(`(.+)\t([0-9]+/[0-9]+)`).ReplaceAll(lsOut, []byte("$1\t0/80"))
	require.NoError(t, os.WriteFile(inBom, transformed, 0o644))
	cmd := exec.Command("mkbom", "-i", inBom, refBom) //nolint:gosec
	require.NoError(t, cmd.Run())

	// Pure-Go BOM.
	myBom := filepath.Join(t.TempDir(), "my.bom")
	require.NoError(t, writeBom(root, myBom))

	require.Equal(t, sortedLsbom(t, refBom), sortedLsbom(t, myBom),
		"lsbom manifest of writeBom output must match the native mkbom pipeline")
}

// TestWriteBomRejectsSymlink verifies writeBom fails loudly on a symlink rather
// than emitting a malformed BOM entry. Symlinks cannot legitimately appear in a
// fleetd payload (extractTarGz rejects them; the orbit "current" symlink is
// created by postinstall at install time), so this is a defensive guard.
func TestWriteBomRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "real.txt"), []byte("hi"), 0o644)
	require.NoError(t, os.Symlink("real.txt", filepath.Join(root, "link.txt")))

	err := writeBom(root, filepath.Join(t.TempDir(), "out.bom"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported file type")
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, mode))
}

func sortedLsbom(t *testing.T, bom string) string {
	t.Helper()
	out, err := exec.Command("lsbom", bom).Output() //nolint:gosec
	require.NoErrorf(t, err, "lsbom failed to read %s", bom)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
