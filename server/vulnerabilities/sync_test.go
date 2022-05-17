package vulnerabilities

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/config"
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

func TestSync(t *testing.T) {
	nettest.Run(t)

	ds := new(mock.Store)

	var countCVSSScore int
	var countEPSSProbability int
	var countCISAKnownExploit int
	ds.InsertCVEScoresFunc = func(ctx context.Context, cveScores []fleet.CVEScore) error {
		for _, score := range cveScores {
			if score.CVSSScore != nil {
				countCVSSScore++
			}
			if score.EPSSProbability != nil {
				countEPSSProbability++
			}
			if score.CISAKnownExploit {
				countCISAKnownExploit++
			}
		}
		return nil
	}

	tempDir := t.TempDir()
	err := Sync(tempDir, config.FleetConfig{}, ds)
	require.NoError(t, err)

	require.True(t, ds.InsertCVEScoresFuncInvoked)

	// ensure some non NULL values were inserted
	require.True(t, countCVSSScore > 0)
	require.True(t, countEPSSProbability > 0)
	require.True(t, countCISAKnownExploit > 0)
}
