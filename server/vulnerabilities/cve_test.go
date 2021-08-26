package vulnerabilities

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

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

func TestTranslateCPEToCVE(t *testing.T) {
	tempDir := t.TempDir()

	ds := new(mock.Store)
	ctx := context.Background()

	for _, tt := range cvetests {
		t.Run(tt.cpe, func(t *testing.T) {
			ds.AllCPEsFunc = func() ([]string, error) {
				return []string{tt.cpe}, nil
			}

			cveLock := &sync.Mutex{}
			cveToCPEs := make(map[string][]string)
			var cvesFound []string
			ds.InsertCVEForCPEFunc = func(cve string, cpes []string) error {
				cveLock.Lock()
				defer cveLock.Unlock()
				cveToCPEs[cve] = cpes
				cvesFound = append(cvesFound, cve)
				return nil
			}

			err := TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewLogfmtLogger(os.Stdout), "")
			require.NoError(t, err)

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
	err := syncCVEData(tempDir, ts.URL)
	require.Error(t, err)
	require.Equal(t,
		fmt.Sprintf("1 synchronisation error:\n\tunexpected size for \"%s/feeds/json/cve/1.1/nvdcve-1.1-2002.json.gz\" (200 OK): want 1453293, have 0", ts.URL),
		err.Error(),
	)
}
