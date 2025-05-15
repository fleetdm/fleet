package file_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Results @b81f69d16220524866fc90e9260a0af0d2aeb94c before any change:
//
// $ GO_TEST_EXTRA_FLAGS="--timeout 20m" FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 go test ./pkg/file -run zzz -bench . -benchmem | prettybench
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/pkg/file
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                                                                            iter       time/iter      bytes alloc             allocs
// ---------                                                                                            ----       ---------      -----------             ------
// BenchmarkExtractInstallerMetadata/.exe/file_size:_39712_kb-8                                            4    266.94 ms/op   257662412 B/op   251387 allocs/op
// BenchmarkExtractInstallerMetadata/Box.app.pkg/file_size:_67219_kb-8                                     3    464.89 ms/op   393721768 B/op     3962 allocs/op
// BenchmarkExtractInstallerMetadata/Fleet_osquery.msi/file_size:_43775_kb-8                               4    277.60 ms/op   252408264 B/op     3737 allocs/op
// BenchmarkExtractInstallerMetadata/Go_Programming_Language_amd64_go1.22.2.msi/file_size:_61680_kb-8      3    513.12 ms/op   402892376 B/op   161092 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg/file_size:_69628_kb-8                                          3    472.77 ms/op   393635344 B/op     1778 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg#01/file_size:_66444_kb-8                                       3    465.80 ms/op   393635776 B/op     1785 allocs/op
// BenchmarkExtractInstallerMetadata/NordVPN.app.pkg/file_size:_155592_kb-8                                1   1011.86 ms/op   961921824 B/op     1919 allocs/op
// BenchmarkExtractInstallerMetadata/Notion_3.11.1.exe/file_size:_77768_kb-8                               2    528.20 ms/op   492055496 B/op      567 allocs/op
// BenchmarkExtractInstallerMetadata/Python.pkg/file_size:_44601_kb-8                                      4    291.94 ms/op   251876720 B/op     5834 allocs/op
// BenchmarkExtractInstallerMetadata/TeamViewer.app.pkg/file_size:_93051_kb-8                              2    594.36 ms/op   492383088 B/op     6823 allocs/op
// BenchmarkExtractInstallerMetadata/Vim.exe/file_size:_10704_kb-8                                        15    117.45 ms/op    65504394 B/op      640 allocs/op
// BenchmarkExtractInstallerMetadata/Visual_Studio_Code.exe/file_size:_97156_kb-8                          2    637.27 ms/op   615259264 B/op      637 allocs/op
// BenchmarkExtractInstallerMetadata/code.deb/file_size:_99278_kb-8                                        3    379.23 ms/op     8455728 B/op      116 allocs/op
// BenchmarkExtractInstallerMetadata/code.rpm/file_size:_138886_kb-8                                       2    556.37 ms/op     3397024 B/op    11274 allocs/op
// BenchmarkExtractInstallerMetadata/fleet-osquery.deb/file_size:_79581_kb-8                               4    308.36 ms/op       59696 B/op       90 allocs/op
// BenchmarkExtractInstallerMetadata/htop.deb/file_size:_90_kb-8                                         822      1.96 ms/op     8446331 B/op      110 allocs/op
// BenchmarkExtractInstallerMetadata/ruby.deb/file_size:_11_kb-8                                         649      1.66 ms/op     8448424 B/op      122 allocs/op
// ok  	github.com/fleetdm/fleet/v4/pkg/file	36.644s

// Results @0c700ca40e5d3602b6206f12232c4c123b6c4ee9 with the use of TempFileReader but not change otherwise:
// $ GO_TEST_EXTRA_FLAGS="--timeout 20m" FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 go test ./pkg/file -run zzz -bench . -benchmem | prettybench
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/pkg/file
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                                                                            iter       time/iter      bytes alloc             allocs
// ---------                                                                                            ----       ---------      -----------             ------
// BenchmarkExtractInstallerMetadata/.exe/file_size:_39712_kb-8                                            4    315.43 ms/op   257661592 B/op   251389 allocs/op
// BenchmarkExtractInstallerMetadata/Box.app.pkg/file_size:_67219_kb-8                                     3    418.06 ms/op   393721613 B/op     3962 allocs/op
// BenchmarkExtractInstallerMetadata/Fleet_osquery.msi/file_size:_43775_kb-8                               4    296.01 ms/op   252408208 B/op     3737 allocs/op
// BenchmarkExtractInstallerMetadata/Go_Programming_Language_amd64_go1.22.2.msi/file_size:_61680_kb-8      3    475.20 ms/op   402892136 B/op   161092 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg/file_size:_69628_kb-8                                          2    508.10 ms/op   393635328 B/op     1779 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg#01/file_size:_66444_kb-8                                       2    533.04 ms/op   393635672 B/op     1785 allocs/op
// BenchmarkExtractInstallerMetadata/NordVPN.app.pkg/file_size:_155592_kb-8                                1   1363.48 ms/op   961921792 B/op     1920 allocs/op
// BenchmarkExtractInstallerMetadata/Notion_3.11.1.exe/file_size:_77768_kb-8                               2    621.95 ms/op   492055376 B/op      566 allocs/op
// BenchmarkExtractInstallerMetadata/Python.pkg/file_size:_44601_kb-8                                      2    513.91 ms/op   251876472 B/op     5832 allocs/op
// BenchmarkExtractInstallerMetadata/TeamViewer.app.pkg/file_size:_93051_kb-8                              2    573.12 ms/op   492382960 B/op     6823 allocs/op
// BenchmarkExtractInstallerMetadata/Vim.exe/file_size:_10704_kb-8                                        18    107.35 ms/op    65504316 B/op      640 allocs/op
// BenchmarkExtractInstallerMetadata/Visual_Studio_Code.exe/file_size:_97156_kb-8                          2    606.29 ms/op   615259376 B/op      639 allocs/op
// BenchmarkExtractInstallerMetadata/code.deb/file_size:_99278_kb-8                                        3    397.47 ms/op     8447426 B/op      114 allocs/op
// BenchmarkExtractInstallerMetadata/code.rpm/file_size:_138886_kb-8                                       2    594.10 ms/op     3396936 B/op    11274 allocs/op
// BenchmarkExtractInstallerMetadata/fleet-osquery.deb/file_size:_79581_kb-8                               3    335.98 ms/op       60384 B/op       90 allocs/op
// BenchmarkExtractInstallerMetadata/htop.deb/file_size:_90_kb-8                                         732      3.16 ms/op     8446791 B/op      110 allocs/op
// BenchmarkExtractInstallerMetadata/ruby.deb/file_size:_11_kb-8                                         578      3.48 ms/op     8449575 B/op      122 allocs/op
// ok  	github.com/fleetdm/fleet/v4/pkg/file	37.775s

// Results @64321f8d241bba9233a1de21845ac0c7a6f4dda6 with the .exe improvements (read from disk with mmap) - massively
// better memory usage (only exe benchmarks show):
//
// $ GO_TEST_EXTRA_FLAGS="--timeout 20m" FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 go test ./pkg/file -run zzz -bench . -benchmem | prettybench
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/pkg/file
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                                                                            iter       time/iter      bytes alloc             allocs
// ---------                                                                                            ----       ---------      -----------             ------
// BenchmarkExtractInstallerMetadata/.exe/file_size:_39712_kb-8                                            6    208.14 ms/op     6135304 B/op   251345 allocs/op
// BenchmarkExtractInstallerMetadata/Notion_3.11.1.exe/file_size:_77768_kb-8                               3    337.30 ms/op       61258 B/op      521 allocs/op
// BenchmarkExtractInstallerMetadata/Vim.exe/file_size:_10704_kb-8                                        22     47.32 ms/op       67321 B/op      604 allocs/op
// BenchmarkExtractInstallerMetadata/Visual_Studio_Code.exe/file_size:_97156_kb-8                          3    421.23 ms/op       65573 B/op      591 allocs/op
// ok  	github.com/fleetdm/fleet/v4/pkg/file	35.887s

// Results @e5ad9300701f0aa1f7b40efffdb6944988038dc7 with the .pkg improvements
// - massively better memory usage (only pkg benchmarks shown):
//
// $ GO_TEST_EXTRA_FLAGS="--timeout 20m" FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 go test ./pkg/file -run zzz -bench . -benchmem | prettybench
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/pkg/file
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                                                                            iter      time/iter      bytes alloc             allocs
// ---------                                                                                            ----      ---------      -----------             ------
// BenchmarkExtractInstallerMetadata/Box.app.pkg/file_size:_67219_kb-8                                     4   319.98 ms/op      285844 B/op     3915 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg/file_size:_69628_kb-8                                          4   280.80 ms/op      199696 B/op     1734 allocs/op
// BenchmarkExtractInstallerMetadata/Go.pkg#01/file_size:_66444_kb-8                                       4   286.77 ms/op      199824 B/op     1738 allocs/op
// BenchmarkExtractInstallerMetadata/NordVPN.app.pkg/file_size:_155592_kb-8                                2   713.27 ms/op      223144 B/op     1866 allocs/op
// BenchmarkExtractInstallerMetadata/Python.pkg/file_size:_44601_kb-8                                      6   247.57 ms/op      350997 B/op     5791 allocs/op
// BenchmarkExtractInstallerMetadata/TeamViewer.app.pkg/file_size:_93051_kb-8                              2   586.15 ms/op      389312 B/op     6776 allocs/op
// ok  	github.com/fleetdm/fleet/v4/pkg/file	39.536s

// Results @532daf10bebe7c432a2b5e6c3822639c5937dc29 with the .msi improvements
// - massively better memory usage (only msi benchmarks shown):
// $ GO_TEST_EXTRA_FLAGS="--timeout 20m" FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 go test ./pkg/file -run zzz -bench . -benchmem | prettybench
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/pkg/file
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                                                                            iter      time/iter    bytes alloc             allocs
// ---------                                                                                            ----      ---------    -----------             ------
// BenchmarkExtractInstallerMetadata/Fleet_osquery.msi/file_size:_43775_kb-8                               6   191.69 ms/op    879274 B/op     3752 allocs/op
// BenchmarkExtractInstallerMetadata/Go_Programming_Language_amd64_go1.22.2.msi/file_size:_61680_kb-8      4   305.72 ms/op   8430244 B/op   161054 allocs/op
// ok  	github.com/fleetdm/fleet/v4/pkg/file	32.193s

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
					tfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "installers", dent.Name()))
					require.NoError(b, err)

					meta, err := file.ExtractInstallerMetadata(tfr)
					require.NoError(b, err)
					tfr.Close()

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
