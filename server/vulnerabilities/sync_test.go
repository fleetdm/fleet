package vulnerabilities

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestDownloadEPSSFeed(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient()

	tempDir := t.TempDir()

	err := DownloadEPSSFeed(tempDir, client)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, strings.TrimSuffix(epssFilename, ".gz")))
}

func TestDownloadCISAKnownExploitsFeed(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient()

	tempDir := t.TempDir()

	err := DownloadCISAKnownExploitsFeed(tempDir, client)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, cisaKnownExploitsFilename))
}

func TestLoadCVEMeta(t *testing.T) {
	nettest.Run(t)

	ds := new(mock.Store)

	var countCVSSScore int
	var countEPSSProbability int
	var countCISAKnownExploit int
	ds.InsertCVEMetaFunc = func(ctx context.Context, cveMeta []fleet.CVEMeta) error {
		for _, meta := range cveMeta {
			if meta.CVSSScore != nil {
				countCVSSScore++
			}
			if meta.EPSSProbability != nil {
				countEPSSProbability++
			}
			if meta.CISAKnownExploit != nil {
				countCISAKnownExploit++
			}
		}
		return nil
	}

	tempDir := t.TempDir()
	err := Sync(tempDir, "")
	require.NoError(t, err)

	err = LoadCVEMeta(tempDir, ds)
	require.NoError(t, err)
	require.True(t, ds.InsertCVEMetaFuncInvoked)

	// ensure some non NULL values were inserted
	require.True(t, countCVSSScore > 0)
	require.True(t, countEPSSProbability > 0)
	require.True(t, countCISAKnownExploit > 0)
}
