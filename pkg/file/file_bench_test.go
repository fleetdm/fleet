package file_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkExtractInstallerMetadata(b *testing.B) {
	dents, err := os.ReadDir(filepath.Join("testdata", "installers"))
	if err != nil {
		b.Fatal(err)
	}

	for _, dent := range dents {
		if !dent.Type().IsRegular() || strings.HasPrefix(dent.Name(), ".") {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(dent.Name(), filepath.Ext(dent.Name())), "$")
		if len(parts) < 4 {
			b.Fatalf("invalid filename, expected at least 4 sections, got %d: %s", len(parts), dent.Name())
		}
		wantName, wantVersion, wantHash, wantBundleIdentifier := parts[0], parts[1], parts[2], parts[3]
		wantExtension := strings.TrimPrefix(filepath.Ext(dent.Name()), ".")

		b.Run(wantName+"."+wantExtension, func(b *testing.B) {

			b.ResetTimer()
			b.ReportAllocs()
			info, err := dent.Info()
			require.NoError(b, err)

			b.Run(fmt.Sprintf("file size: %d kb", info.Size()/1024), func(b *testing.B) {
				// the goal of this benchmark is not so much accuracy of time performance, but
				// memory usage, so it doesn't matter that the file is read from disk on each
				// iteration.
				for i := 0; i < b.N; i++ {
					f, err := os.Open(filepath.Join("testdata", "installers", dent.Name()))
					require.NoError(b, err)

					meta, err := file.ExtractInstallerMetadata(f)
					require.NoError(b, err)
					f.Close()

					assert.Equal(b, wantName, meta.Name)
					assert.Equal(b, wantVersion, meta.Version)
					assert.Equal(b, wantHash, hex.EncodeToString(meta.SHASum))
					assert.Equal(b, wantExtension, meta.Extension)
					assert.Equal(b, wantBundleIdentifier, meta.BundleIdentifier)
				}
			})
		})
	}
}
