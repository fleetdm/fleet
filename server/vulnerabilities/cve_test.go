package vulnerabilities

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

var cvetests = []struct {
	cpe, cve string
}{
	{"cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:macos:*:*", "CVE-2012-6369"},
	{"cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:*:*:*", "CVE-2012-6369"},
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestTranslateCPEToCVE(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	tempDir := t.TempDir()

	ds := new(mock.Store)
	ctx := context.Background()

	for _, tt := range cvetests {
		t.Run(tt.cpe, func(t *testing.T) {
			ds.AllCPEsFunc = func(ctx context.Context) ([]string, error) {
				return []string{tt.cpe}, nil
			}

			cveLock := &sync.Mutex{}
			cveToCPEs := make(map[string][]string)
			var cvesFound []string
			ds.InsertCVEForCPEFunc = func(ctx context.Context, cve string, cpes []string) error {
				cveLock.Lock()
				defer cveLock.Unlock()
				cveToCPEs[cve] = cpes
				cvesFound = append(cvesFound, cve)
				return nil
			}

			err := TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewLogfmtLogger(os.Stdout), config.FleetConfig{})
			require.NoError(t, err)

			PrintMemUsage()

			require.Equal(t, []string{tt.cve}, cvesFound)
			require.Equal(t, []string{tt.cpe}, cveToCPEs[tt.cve])
		})
	}
}

func TestSyncsCVEFromURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.RequestURI, ".meta") {
			fmt.Fprint(w, "lastModifiedDate:2021-08-04T11:10:30-04:00\r\n")
			fmt.Fprint(w, "size:20967174\r\n")
			fmt.Fprint(w, "zipSize:1453429\r\n")
			fmt.Fprint(w, "gzSize:1453293\r\n")
			fmt.Fprint(w, "sha256:10D7338A1E2D8DB344C381793110B67FCA7D729ADA21624EF089EBA78CCE7B53\r\n")
		}
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	err := SyncCVEData(
		tempDir, config.FleetConfig{Vulnerabilities: config.VulnerabilitiesConfig{CVEFeedPrefixURL: ts.URL}})
	require.Error(t, err)
	require.Equal(t,
		fmt.Sprintf("1 synchronisation error:\n\tunexpected size for \"%s/feeds/json/cve/1.1/nvdcve-1.1-2002.json.gz\" (200 OK): want 1453293, have 0", ts.URL),
		err.Error(),
	)
}

func TestSyncsCVEFromURLSkipsIfDisableSync(t *testing.T) {
	tempDir := t.TempDir()
	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DisableDataSync: true,
		},
	}
	err := SyncCVEData(tempDir, fleetConfig)
	require.NoError(t, err)
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if match, err := regexp.MatchString("nvdcve.*\\.gz$", path); !match || err != nil {
			return nil
		}

		t.FailNow()

		return nil
	})
	require.NoError(t, err)
}
