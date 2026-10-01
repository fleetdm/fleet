package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeGzFile writes payload gzipped to dir and returns the path.
func writeGzFile(t *testing.T, dir string, payload []byte) string {
	t.Helper()

	path := filepath.Join(dir, "nvdcve-1.1-2026.json.gz")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(payload)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))

	return path
}

func TestGunzipFileToDisk(t *testing.T) {
	const limit int64 = 1024

	for _, tc := range []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{name: "under limit", size: limit - 1},
		{name: "at limit", size: limit},
		{name: "over limit", size: limit + 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			payload := bytes.Repeat([]byte("a"), int(tc.size))
			gzPath := writeGzFile(t, dir, payload)
			outPath := filepath.Join(dir, "nvdcve-1.1-2026.json")

			err := gunzipFileToDisk(gzPath, dir, limit)
			if tc.wantErr {
				require.Error(t, err)
				// A truncated feed left on disk parses as valid JSON to a later
				// stage, so it must not survive the failure.
				require.NoFileExists(t, outPath)
				return
			}

			require.NoError(t, err)
			got, err := os.ReadFile(outPath)
			require.NoError(t, err)
			require.Equal(t, payload, got)
		})
	}
}
