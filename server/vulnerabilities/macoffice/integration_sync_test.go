package macoffice

import (
	"context"
	"os"
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

	// Checking for the presence of the file for today or yesterday
	// in case the NVD repo is having delays publishing the data
	todayFilename := io.MacOfficeRelNotesFileName(time.Now())
	yesterdayFilename := io.MacOfficeRelNotesFileName(time.Now().AddDate(0, 0, -1))

	require.Condition(t, func() bool {
		return contains(filesInVulnPath, todayFilename) || contains(filesInVulnPath, yesterdayFilename)
	}, "Expected to find %s or %s in %s", todayFilename, yesterdayFilename, vulnPath)
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
