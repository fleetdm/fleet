package vulnerabilities

import (
	"context"
	"os"
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
	tempDir, err := os.MkdirTemp(os.TempDir(), "TestTranslateCPEToCVE-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

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

			err = TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewLogfmtLogger(os.Stdout))
			require.NoError(t, err)

			require.Equal(t, []string{tt.cve}, cvesFound)
			require.Equal(t, []string{tt.cpe}, cveToCPEs[tt.cve])
		})
	}

}
