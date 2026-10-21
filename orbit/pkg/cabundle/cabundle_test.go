package cabundle

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedBundleIsKnown(t *testing.T) {
	_, ok := knownBundles[Hash(Bundle)]
	require.True(t, ok, "add the embedded bundle's hash to known_bundles.txt")
	require.True(t, bytes.HasPrefix(Bundle, []byte("##\n## Bundle of CA Root Certificates")))
}

func TestHashNormalizesCRLF(t *testing.T) {
	lf := []byte("a\nb\n")
	crlf := []byte("a\r\nb\r\n")
	require.Equal(t, Hash(lf), Hash(crlf))
	require.NotEqual(t, Hash(lf), Hash([]byte("a\nb")))
}

func TestRefresh(t *testing.T) {
	oldBundle := []byte("##\n## Bundle of CA Root Certificates\n## older bundle used in tests\n")
	customBundle := []byte("-----BEGIN CERTIFICATE-----\ncustom internal CA\n-----END CERTIFICATE-----\n")

	// Register the fake old bundle as one Fleet shipped.
	knownBundles[Hash(oldBundle)] = struct{}{}
	t.Cleanup(func() { delete(knownBundles, Hash(oldBundle)) })

	cases := []struct {
		name        string
		installed   []byte // nil means no file
		wantOutcome Outcome
		wantContent []byte // nil means no file afterwards
	}{
		{"missing file is left missing", nil, Missing, nil},
		{"current bundle is unchanged", Bundle, Unchanged, Bundle},
		{"current bundle with CRLF is unchanged", bytes.ReplaceAll(Bundle, []byte("\n"), []byte("\r\n")), Unchanged, bytes.ReplaceAll(Bundle, []byte("\n"), []byte("\r\n"))},
		{"old Fleet bundle is replaced", oldBundle, Updated, Bundle},
		{"old Fleet bundle with CRLF is replaced", bytes.ReplaceAll(oldBundle, []byte("\n"), []byte("\r\n")), Updated, Bundle},
		{"custom bundle is left alone", customBundle, Custom, customBundle},
		{"empty file is left alone", []byte{}, Custom, []byte{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			if tc.installed != nil {
				require.NoError(t, os.WriteFile(path, tc.installed, 0o644))
			}

			got, err := Refresh(dir)
			require.NoError(t, err)
			require.Equal(t, tc.wantOutcome, got)

			if tc.wantContent == nil {
				_, err := os.Stat(path)
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, tc.wantContent, content)

			if runtime.GOOS != "windows" {
				info, err := os.Stat(path)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
			}

			// No temp files left behind.
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1)
		})
	}
}

func TestRefreshIsIdempotent(t *testing.T) {
	oldBundle := []byte("## older bundle\n")
	knownBundles[Hash(oldBundle)] = struct{}{}
	t.Cleanup(func() { delete(knownBundles, Hash(oldBundle)) })

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), oldBundle, 0o644))

	got, err := Refresh(dir)
	require.NoError(t, err)
	require.Equal(t, Updated, got)

	got, err = Refresh(dir)
	require.NoError(t, err)
	require.Equal(t, Unchanged, got)
}
