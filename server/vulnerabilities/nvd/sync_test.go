package nvd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestDownloadEPSSFeed(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	err := DownloadEPSSFeed(tempDir)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, strings.TrimSuffix(epssFilename, ".gz")))
}

func TestDownloadCISAKnownExploitsFeed(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	err := DownloadCISAKnownExploitsFeed(tempDir)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, cisaKnownExploitsFilename))
}

func TestLoadCVEMeta(t *testing.T) {
	ds := new(mock.Store)

	var cveMeta []fleet.CVEMeta
	ds.InsertCVEMetaFunc = func(ctx context.Context, x []fleet.CVEMeta) error {
		cveMeta = x
		return nil
	}

	logger := log.NewNopLogger()
	err := LoadCVEMeta(license.NewContext(context.Background(), &fleet.LicenseInfo{
		Tier: "premium",
	}), logger, "../testdata", ds)
	require.NoError(t, err)
	require.True(t, ds.InsertCVEMetaFuncInvoked)

	// check some cves to make sure they got loaded correctly
	metaMap := make(map[string]fleet.CVEMeta)
	for _, meta := range cveMeta {
		metaMap[meta.CVE] = meta
	}

	meta := metaMap["CVE-2022-29676"]
	require.Equal(t, float64(7.2), *meta.CVSSScore)
	require.Equal(t, float64(0.00885), *meta.EPSSProbability)
	require.Equal(t, false, *meta.CISAKnownExploit)
	require.Equal(
		t,
		"CSCMS Music Portal System v4.2 was discovered to contain a SQL injection vulnerability via the id parameter at /admin.php/pic/admin/lists/zhuan.",
		meta.Description,
	)

	meta = metaMap["CVE-2022-22587"]
	require.Equal(t, float64(9.8), *meta.CVSSScore)
	require.Equal(t, float64(0.01843), *meta.EPSSProbability)
	require.Equal(t, true, *meta.CISAKnownExploit)
	require.Equal(
		t,
		"A memory corruption issue was addressed with improved input validation. This issue is fixed in iOS 15.3 and iPadOS 15.3, macOS Big Sur 11.6.3, macOS Monterey 12.2. A malicious application may be able to execute arbitrary code with kernel privileges. Apple is aware of a report that this issue may have been actively exploited..",
		meta.Description,
	)
}

func TestDownloadCPETranslations(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	err := DownloadCPETranslationsFromGithub(tempDir, "")
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, cpeTranslationsFilename))
}
