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

func TestIntegrationSync(t *testing.T) {
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
	require.Contains(t, filesInVulnPath, io.MacOfficeRelNotesFileName(time.Now()))
}
