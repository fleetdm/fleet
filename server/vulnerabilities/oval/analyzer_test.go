package oval

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestOvalAnalyzer(t *testing.T) {
	t.Run("#vulnsDelta", func(t *testing.T) {
		t.Run("no existing vulnerabilities", func(t *testing.T) {
			found := make(map[uint][]string)
			existing := make([]fleet.SoftwareVulnerability, 0)

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Empty(t, toInsert)
			require.Empty(t, toDelete)
		})

		t.Run("existing match found", func(t *testing.T) {
			found := map[uint][]string{
				1: {"cve_1", "cve_2"},
				2: {"cve_3", "cve_4"},
			}

			existing := []fleet.SoftwareVulnerability{
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_1"},
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_2"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_3"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_4"},
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Empty(t, toInsert)
			require.Empty(t, toDelete)
		})

		t.Run("existing differ from found", func(t *testing.T) {
			found := map[uint][]string{
				1: {"cve_1", "cve_2"},
				3: {"cve_10", "cve_20"},
			}

			existing := []fleet.SoftwareVulnerability{
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_1"},
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_2"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_3"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_4"},
			}

			expectedToInsert := map[uint][]string{
				3: {"cve_10", "cve_20"},
			}

			expectedToDelete := []fleet.SoftwareVulnerability{
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_3"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_4"},
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Equal(t, expectedToInsert, toInsert)
			require.ElementsMatch(t, expectedToDelete, toDelete)
		})
	})

	t.Run("#load", func(t *testing.T) {
		t.Run("invalid vuln path", func(t *testing.T) {
			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			_, err := loadDef(platform, "")
			require.Error(t, err, "invalid vulnerabity path")
		})
	})

	t.Run("#latestOvalDefFor", func(t *testing.T) {
		t.Run("definition matching platform for date exists", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()
			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def := filepath.Join(path, platform.ToFilename(today, "json"))

			f1, err := os.Create(def)
			require.NoError(t, err)
			f1.Close()

			result, err := latestOvalDefFor(platform, path, today)
			require.NoError(t, err)
			require.Equal(t, def, result)
		})

		t.Run("definition matching platform exists but not for date", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()
			yesterday := today.Add(-24 * time.Hour)

			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def := filepath.Join(path, platform.ToFilename(yesterday, "json"))

			f1, err := os.Create(def)
			require.NoError(t, err)
			f1.Close()

			result, err := latestOvalDefFor(platform, path, today)
			require.NoError(t, err)
			require.Equal(t, def, result)
		})

		t.Run("definition does not exists for platform", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()

			platform1 := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def1 := filepath.Join(path, platform1.ToFilename(today, "json"))
			f1, err := os.Create(def1)
			require.NoError(t, err)
			f1.Close()

			platform2 := NewPlatform("ubuntu", "Ubuntu 18.4.0")

			_, err = latestOvalDefFor(platform2, path, today)
			require.Error(t, err, "file not found for platform")
		})
	})
}
