package netskope

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFileInfo struct{ dir bool }

func (f fakeFileInfo) Name() string       { return "" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

// statter returns a stat func that reports only the given paths as existing.
func statter(paths map[string]fakeFileInfo) func(string) (os.FileInfo, error) {
	return func(path string) (os.FileInfo, error) {
		if info, ok := paths[path]; ok {
			return info, nil
		}
		return nil, os.ErrNotExist
	}
}

func TestFindInstallPath(t *testing.T) {
	t.Parallel()

	// Build paths with filepath.Join and nsdiagBinaryName so the expectations
	// match what findInstallPath computes on the host running the test
	// (separator, and "nsdiag" vs "nsdiag.exe").
	first := filepath.Join("Netskope", "STAgent")
	second := filepath.Join("Netskope (x86)", "STAgent")
	candidates := []string{first, second}

	for _, tc := range []struct {
		name  string
		paths map[string]fakeFileInfo
		want  string
	}{
		{
			name: "prefers the candidate holding nsdiag over an earlier bare directory",
			paths: map[string]fakeFileInfo{
				first:  {dir: true},
				second: {dir: true},
				filepath.Join(second, nsdiagBinaryName()): {dir: false},
			},
			want: second,
		},
		{
			name: "first candidate with nsdiag wins",
			paths: map[string]fakeFileInfo{
				first:                                    {dir: true},
				filepath.Join(first, nsdiagBinaryName()): {dir: false},
				second:                                   {dir: true},
				filepath.Join(second, nsdiagBinaryName()): {dir: false},
			},
			want: first,
		},
		{
			// A present-but-incomplete install is still reported, so Generate can
			// explain why it has no state instead of claiming nothing is installed.
			name:  "falls back to an existing directory without nsdiag",
			paths: map[string]fakeFileInfo{first: {dir: true}},
			want:  first,
		},
		{
			name:  "nothing installed",
			paths: map[string]fakeFileInfo{},
			want:  "",
		},
		{
			// Guards against a directory named "nsdiag" being treated as the binary.
			name: "a directory named nsdiag is not the binary",
			paths: map[string]fakeFileInfo{
				filepath.Join(first, nsdiagBinaryName()): {dir: true},
			},
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, findInstallPath(candidates, statter(tc.paths)))
		})
	}
}

func TestFindInstallPathNoCandidates(t *testing.T) {
	t.Parallel()

	assert.Empty(t, findInstallPath(nil, statter(nil)))
}

func TestNsdiagBinaryName(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		assert.Equal(t, "nsdiag.exe", nsdiagBinaryName())
		return
	}
	assert.Equal(t, "nsdiag", nsdiagBinaryName())
}

// TestInstallPathCandidatesAreAbsolute matters for security: nsdiag is executed
// as root, so the path must never be relative or resolved through $PATH.
func TestInstallPathCandidatesAreAbsolute(t *testing.T) {
	t.Parallel()

	candidates := installPathCandidates()
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		require.NotEmpty(t, candidates, "expected install path candidates on %s", runtime.GOOS)
	default:
		assert.Empty(t, candidates, "unsupported platforms must report no candidates")
		return
	}

	for _, c := range candidates {
		// filepath.IsAbs understands drive-letter paths when GOOS is windows.
		assert.True(t, filepath.IsAbs(c), "candidate %q must be an absolute path", c)
	}
}
