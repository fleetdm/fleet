package nvd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cvetests = []struct {
	cpe  string
	cves []string
}{
	{
		"cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:macos:*:*",
		[]string{"CVE-2012-6369"},
	},
	{
		"cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:*:*:*",
		[]string{"CVE-2012-6369"},
	},
	{
		"cpe:2.3:a:pypa:pip:9.0.3:*:*:*:*:python:*:*",
		[]string{
			"CVE-2019-20916",
			"CVE-2021-3572",
		},
	},
	{
		"cpe:2.3:a:mozilla:firefox:93.0:*:*:*:*:windows:*:*",
		[]string{
			"CVE-2021-43540",
			"CVE-2021-38503",
			"CVE-2021-38504",
			"CVE-2021-38506",
			"CVE-2021-38507",
			"CVE-2021-38508",
			"CVE-2021-38509",
			"CVE-2021-43534",
			"CVE-2021-43532",
			"CVE-2021-43531",
			"CVE-2021-43533",

			"CVE-2021-43538",
			"CVE-2021-43542",
			"CVE-2021-43543",
			"CVE-2021-30547",
			"CVE-2021-43546",
			"CVE-2021-43537",
			"CVE-2021-43541",
			"CVE-2021-43536",
			"CVE-2021-43545",
			"CVE-2021-43539",
		},
	},
	{
		"cpe:2.3:a:mozilla:firefox:93.0.100:*:*:*:*:windows:*:*",
		[]string{
			"CVE-2021-43540",
			"CVE-2021-38503",
			"CVE-2021-38504",
			"CVE-2021-38506",
			"CVE-2021-38507",
			"CVE-2021-38508",
			"CVE-2021-38509",
			"CVE-2021-43534",
			"CVE-2021-43532",
			"CVE-2021-43531",
			"CVE-2021-43533",

			"CVE-2021-43538",
			"CVE-2021-43542",
			"CVE-2021-43543",
			"CVE-2021-30547",
			"CVE-2021-43546",
			"CVE-2021-43537",
			"CVE-2021-43541",
			"CVE-2021-43536",
			"CVE-2021-43545",
			"CVE-2021-43539",
		},
	},
}

func printMemUsage() {
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

type threadSafeDSMock struct {
	mu sync.Mutex
	*mock.Store
}

func (d *threadSafeDSMock) ListSoftwareCPEs(ctx context.Context) ([]fleet.SoftwareCPE, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Store.ListSoftwareCPEs(ctx)
}

func (d *threadSafeDSMock) InsertVulnerabilities(ctx context.Context, vulns []fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Store.InsertVulnerabilities(ctx, vulns, src)
}

func TestTranslateCPEToCVE(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	ds := new(mock.Store)
	ctx := context.Background()

	// download the CVEs once for all sub-tests, and then disable syncing
	err := nettest.RunWithNetRetry(t, func() error {
		return DownloadNVDCVEFeed(tempDir, "")
	})
	require.NoError(t, err)

	for _, tt := range cvetests {
		t.Run(tt.cpe, func(t *testing.T) {
			ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
				return []fleet.SoftwareCPE{
					{CPE: tt.cpe},
				}, nil
			}

			cveLock := &sync.Mutex{}
			var cvesFound []string
			ds.InsertVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (int64, error) {
				cveLock.Lock()
				defer cveLock.Unlock()
				for _, v := range vulns {
					cvesFound = append(cvesFound, v.CVE)
				}

				return 0, nil
			}

			_, err := TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewLogfmtLogger(os.Stdout), false)
			require.NoError(t, err)

			printMemUsage()

			require.ElementsMatch(t, cvesFound, tt.cves, tt.cpe)
		})
	}

	t.Run("recent_vulns", func(t *testing.T) {
		safeDS := &threadSafeDSMock{Store: ds}

		softwareCPEs := []fleet.SoftwareCPE{
			{CPE: "cpe:2.3:a:google:chrome:-:*:*:*:*:*:*:*", ID: 1, SoftwareID: 1},
			{CPE: "cpe:2.3:a:mozilla:firefox:-:*:*:*:*:*:*:*", ID: 2, SoftwareID: 2},
			{CPE: "cpe:2.3:a:haxx:curl:-:*:*:*:*:*:*:*", ID: 3, SoftwareID: 3},
		}
		ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
			return softwareCPEs, nil
		}

		ds.InsertVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (int64, error) {
			return 1, nil
		}
		recent, err := TranslateCPEToCVE(ctx, safeDS, tempDir, kitlog.NewNopLogger(), true)
		require.NoError(t, err)

		byCPE := make(map[uint]int)
		for _, cpe := range recent {
			byCPE[cpe.SoftwareID]++
		}

		// even if it's somewhat far in the past, I've seen the exact numbers
		// change a bit between runs with different downloads, so allow for a bit
		// of wiggle room.
		assert.Greater(t, byCPE[softwareCPEs[0].SoftwareID], 150, "google chrome CVEs")
		assert.Greater(t, byCPE[softwareCPEs[1].SoftwareID], 280, "mozilla firefox CVEs")
		assert.Greater(t, byCPE[softwareCPEs[2].SoftwareID], 10, "curl CVEs")

		// call it again but now return 0 from this call, simulating CVE-CPE pairs
		// that already existed in the DB.
		ds.InsertVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (int64, error) {
			return 0, nil
		}
		recent, err = TranslateCPEToCVE(ctx, safeDS, tempDir, kitlog.NewNopLogger(), true)
		require.NoError(t, err)

		// no recent vulnerability should be reported
		assert.Len(t, recent, 0)
	})
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
	cveFeedPrefixURL := ts.URL + "/feeds/json/cve/1.1/"
	err := DownloadNVDCVEFeed(tempDir, cveFeedPrefixURL)
	require.Error(t, err)
	require.Contains(t,
		err.Error(),
		fmt.Sprintf("1 synchronisation error:\n\tunexpected size for \"%s/feeds/json/cve/1.1/nvdcve-1.1-2002.json.gz\" (200 OK): want 1453293, have 0", ts.URL),
	)
}
