package cabundle

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBundle(t *testing.T) {
	require.True(t, bytes.HasPrefix(Bundle, []byte("##\n## Bundle of CA Root Certificates")))
}

func TestRefresh(t *testing.T) {
	cases := []struct {
		name        string
		installed   []byte // nil means no file
		wantUpdated bool
	}{
		{"missing file", nil, false},
		{"current bundle", Bundle, false},
		{"old bundle", []byte("##\n## Bundle of CA Root Certificates\n## old\n"), true},
		{"bundle with roots stripped by a script", bytes.TrimSuffix(Bundle, []byte("-----END CERTIFICATE-----\n")), true},
		{"empty file", []byte{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			if tc.installed != nil {
				require.NoError(t, os.WriteFile(path, tc.installed, 0o600))
			}

			updated, err := Refresh(dir)
			require.NoError(t, err)
			require.Equal(t, tc.wantUpdated, updated)

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			if tc.installed == nil {
				require.Empty(t, entries)
				return
			}
			require.Len(t, entries, 1, "temp file left behind")

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, Bundle, content)

			if tc.wantUpdated && runtime.GOOS != "windows" {
				info, err := os.Stat(path)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
			}

			updated, err = Refresh(dir)
			require.NoError(t, err)
			require.False(t, updated)
		})
	}
}
