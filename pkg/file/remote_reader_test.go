package file_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mock "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/stretchr/testify/require"
)

// TestExtractInstallerMetadataRemoteParity checks that an installer streamed
// to a staging object and parsed with ranged reads gives the same result as
// parsing it from a temp file, for every supported format.
func TestExtractInstallerMetadataRemoteParity(t *testing.T) {
	fixtures := []string{
		"../../server/service/testdata/software-installers/EchoApp.pkg",
		"../../server/service/testdata/software-installers/dummy_installer.pkg",
		"../../server/service/testdata/software-installers/no_version.pkg",
		"../../server/service/testdata/software-installers/emacs.deb",
		"../../server/service/testdata/software-installers/ruby.deb",
		"../../server/service/testdata/software-installers/ruby.rpm",
		"../../server/service/testdata/software-installers/fleet-osquery.msi",
		"../../server/service/testdata/software-installers/hello-world-installer.exe",
		"../../server/service/testdata/software-installers/ipa_test.ipa",
		"../../server/service/testdata/software-installers/test.tar.gz",
		"testdata/software-installers/msix_test.msix",
		"testdata/invalid.tar.gz",
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			content, err := os.ReadFile(fixture)
			require.NoError(t, err)

			tfr, err := fleet.NewTempFileReader(bytes.NewReader(content), t.TempDir)
			require.NoError(t, err)
			defer tfr.Close()
			wantMeta, wantErr := file.ExtractInstallerMetadata(tfr)

			check := func(t *testing.T, r fleet.InstallerReader) {
				meta, err := file.ExtractInstallerMetadata(r)
				if wantErr != nil {
					require.Error(t, err)
					require.Equal(t, wantErr.Error(), err.Error())
					return
				}
				require.NoError(t, err)
				require.Equal(t, wantMeta, meta)
			}

			t.Run("staged with known hash", func(t *testing.T) {
				store := mock.NewMemStreamingInstallerStore()
				rir, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content))
				require.NoError(t, err)
				t.Cleanup(func() { _ = rir.Close() })
				check(t, rir)
			})
			t.Run("staged with stream validator", func(t *testing.T) {
				store := mock.NewMemStreamingInstallerStore()
				rir, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content), file.NewTarGzStreamValidator())
				require.NoError(t, err)
				t.Cleanup(func() { _ = rir.Close() })
				if wantErr == nil && wantMeta.Extension == "tar.gz" {
					require.Equal(t, "tar.gz", rir.PrevalidatedFormat())
				} else {
					require.Empty(t, rir.PrevalidatedFormat())
				}
				check(t, rir)
			})
			t.Run("remote without hash", func(t *testing.T) {
				store := mock.NewMemStreamingInstallerStore()
				require.NoError(t, store.PutStaging(t.Context(), "id", bytes.NewReader(content)))
				rir, err := fleet.NewRemoteInstallerReader(t.Context(), store, "id", int64(len(content)), "")
				require.NoError(t, err)
				t.Cleanup(func() { _ = rir.Close() })
				check(t, rir)
			})
		})
	}
}

func TestTarGzStreamValidator(t *testing.T) {
	valid, err := os.ReadFile("../../server/service/testdata/software-installers/test.tar.gz")
	require.NoError(t, err)
	invalid, err := os.ReadFile("testdata/invalid.tar.gz")
	require.NoError(t, err)

	feed := func(content []byte, chunk int) string {
		v := file.NewTarGzStreamValidator()
		for len(content) > 0 {
			n := min(chunk, len(content))
			written, err := v.Write(content[:n])
			require.NoError(t, err)
			require.Equal(t, n, written)
			content = content[n:]
		}
		return v.Finish()
	}

	for _, chunk := range []int{1, 7, 1 << 20} {
		require.Equal(t, "tar.gz", feed(valid, chunk), "chunk %d", chunk)
		require.Empty(t, feed(invalid, chunk), "chunk %d", chunk)
	}
	require.Empty(t, feed(valid[:len(valid)/2], 4096), "truncated")
	require.Empty(t, feed([]byte("not a gzip stream"), 3))
	require.Empty(t, feed(nil, 1))
	require.Empty(t, feed([]byte{0x1f}, 1))

	// Validation failing early doesn't block the rest of the writes.
	garbage := append([]byte{0x1f, 0x8b}, bytes.Repeat([]byte{0xff}, 4<<20)...)
	require.Empty(t, feed(garbage, 64<<10))
}
