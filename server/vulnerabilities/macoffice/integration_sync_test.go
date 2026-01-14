package macoffice

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func TestIntegrationsSync(t *testing.T) {
	nettest.Run(t)

	vulnPath := t.TempDir()
	ctx := context.Background()

	err := SyncFromGithub(ctx, vulnPath)
	require.NoError(t, err)

	entries, err := os.ReadDir(vulnPath)
	var filesInVulnPath []string
	for _, e := range entries {
		filesInVulnPath = append(filesInVulnPath, e.Name())
	}

	require.NoError(t, err)

	// Checking for the presence of the file from the last 7 days
	// in case the NVD repo is having delays publishing the data (weekends, holidays, infra issues, etc.)
	var expectedFilenames []string
	for i := range 7 {
		expectedFilenames = append(expectedFilenames, io.MacOfficeRelNotesFileName(time.Now().AddDate(0, 0, -i)))
	}

	require.Condition(t, func() bool {
		for _, expectedFilename := range expectedFilenames {
			if slices.Contains(filesInVulnPath, expectedFilename) {
				return true
			}
		}
		return false
	}, "Expected to find one of %v in %s", expectedFilenames, vulnPath)
}
