package oval

import (
	"compress/bzip2"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestOvalAnalyzer(t *testing.T) {
	t.Run("analyzing Ubuntu software", func(t *testing.T) {
		type softwareFixture struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}

		systems := []fleet.OSVersion{
			{Platform: "ubuntu", Name: "Ubuntu 16.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 18.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 20.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 21.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 21.10.0"},
			{Platform: "ubuntu", Name: "Ubuntu 22.4.0"},
		}

		ds := mysql.CreateMySQLDS(t)
		defer mysql.TruncateTables(t, ds)
		ctx := context.Background()

		vulnPath, err := ioutil.TempDir("", "oval_analyzer_ubuntu")
		defer os.RemoveAll(vulnPath)
		require.NoError(t, err)

		extract := func(src, dst string) {
			srcF, err := os.Open(src)
			defer srcF.Close()
			require.NoError(t, err)

			dstF, err := os.Create(dst)
			defer dstF.Close()
			require.NoError(t, err)

			r := bzip2.NewReader(srcF)
			_, err = io.Copy(dstF, r)
			require.NoError(t, err)
		}

		extractFixtures := func(p Platform) {
			fixtPath := "testdata/ubuntu"

			srcDefPath := path.Join(fixtPath, fmt.Sprintf("%s-oval_def.json.bz2", p))
			dstDefPath := path.Join(vulnPath, p.ToFilename(time.Now(), "json"))
			extract(srcDefPath, dstDefPath)

			srcSoftPath := path.Join(fixtPath, "software", fmt.Sprintf("%s-software.json.bz2", p))
			dstSoftPath := path.Join(vulnPath, fmt.Sprintf("%s-software.json", p))
			extract(srcSoftPath, dstSoftPath)

			srcCvesPath := path.Join(fixtPath, "software", fmt.Sprintf("%s-software_cves.csv.bz2", p))
			dstCvesPath := path.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", p))
			extract(srcCvesPath, dstCvesPath)
		}

		loadSoftware := func(p Platform, s fleet.OSVersion) *fleet.Host {
			h := test.NewHost(t, ds, string(p), "127.0.0.1", string(p), string(p), time.Now())
			h.Platform = s.Platform
			h.OSVersion = s.Name
			err := ds.UpdateHost(ctx, h)
			require.NoError(t, err)

			var fixtures []softwareFixture
			contents, err := ioutil.ReadFile(path.Join(vulnPath, fmt.Sprintf("%s-software.json", p)))
			require.NoError(t, err)

			err = json.Unmarshal(contents, &fixtures)
			require.NoError(t, err)

			var software []fleet.Software
			for _, fi := range fixtures {
				software = append(software, fleet.Software{
					Name:    fi.Name,
					Version: fi.Version,
				})
			}
			err = ds.UpdateHostSoftware(ctx, h.ID, software)
			require.NoError(t, err)

			err = ds.LoadHostSoftware(ctx, h, fleet.SoftwareListOptions{})
			require.NoError(t, err)

			for _, s := range h.Software {
				err = ds.AddCPEForSoftware(ctx, s, fmt.Sprintf("%s-%s", s.Name, s.Version))
				require.NoError(t, err)
			}

			return h
		}

		assertVulns := func(h *fleet.Host, p Platform) {
			fPath := path.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", p))
			f, err := os.Open(fPath)
			defer f.Close()
			require.NoError(t, err)

			r := csv.NewReader(f)
			var expected []string
			for {
				row, err := r.Read()
				if err == io.EOF {
					break
				}

				if len(row) < 1 {
					continue
				}

				expected = append(expected, row[0])
			}
			require.NotEmpty(t, expected)

			storedVulns, err := ds.ListSoftwareVulnerabilities(ctx, h.ID)
			require.NoError(t, err)

			uniq := make(map[string]bool)
			for _, v := range storedVulns {
				uniq[v.CVE] = true
			}
			actual := make([]string, 0, len(uniq))
			for k := range uniq {
				actual = append(actual, k)
			}

			require.ElementsMatch(t, actual, expected)
		}

		for _, s := range systems {
			p := NewPlatform(s.Platform, s.Name)

			extractFixtures(p)
			h := loadSoftware(p, s)

			err := ds.UpdateOSVersions(ctx)
			require.NoError(t, err)

			osVersions, err := ds.OSVersions(ctx, nil, nil)
			require.NoError(t, err)

			_, err = Analyze(ctx, ds, osVersions, vulnPath)
			require.NoError(t, err)

			assertVulns(h, p)

			err = ds.DeleteHost(ctx, h.ID)
			require.NoError(t, err)
		}
	})

	t.Run("#vulnsDelta", func(t *testing.T) {
		t.Run("no existing vulnerabilities", func(t *testing.T) {
			var found []fleet.SoftwareVulnerability
			var existing []fleet.SoftwareVulnerability

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Empty(t, toInsert)
			require.Empty(t, toDelete)
		})

		t.Run("existing match found", func(t *testing.T) {
			found := []fleet.SoftwareVulnerability{
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_1"},
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_2"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_3"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_4"},
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
			found := []fleet.SoftwareVulnerability{
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_1"},
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_2"},
				{CPE: "cpe_3", CPEID: 3, CVE: "cve_5"},
				{CPE: "cpe_3", CPEID: 3, CVE: "cve_6"},
			}

			existing := []fleet.SoftwareVulnerability{
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_1"},
				{CPE: "cpe_1", CPEID: 1, CVE: "cve_2"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_3"},
				{CPE: "cpe_2", CPEID: 2, CVE: "cve_4"},
			}

			expectedToInsert := []fleet.SoftwareVulnerability{
				{CPE: "cpe_3", CPEID: 3, CVE: "cve_5"},
				{CPE: "cpe_3", CPEID: 3, CVE: "cve_6"},
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
