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
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func withTestFixture(
	version fleet.OSVersion,
	vulnPath string,
	ds *mysql.Datastore,
	afterLoad func(h *fleet.Host),
	t require.TestingT,
) {
	type softwareFixture struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	ctx := context.Background()

	extract := func(src, dst string) {
		srcF, err := os.Open(src)
		require.NoError(t, err)
		defer srcF.Close()

		dstF, err := os.Create(dst)
		require.NoError(t, err)
		defer dstF.Close()

		r := bzip2.NewReader(srcF)
		// ignoring "G110: Potential DoS vulnerability via decompression bomb", as this is test code.
		_, err = io.Copy(dstF, r) //nolint:gosec
		require.NoError(t, err)
	}

	extractFixtures := func(p Platform) {
		fixtPath := "testdata/ubuntu"

		srcDefPath := filepath.Join(fixtPath, fmt.Sprintf("%s-oval_def.json.bz2", p))
		dstDefPath := filepath.Join(vulnPath, p.ToFilename(time.Now(), "json"))
		extract(srcDefPath, dstDefPath)

		srcSoftPath := filepath.Join(fixtPath, "software", fmt.Sprintf("%s-software.json.bz2", p))
		dstSoftPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software.json", p))
		extract(srcSoftPath, dstSoftPath)

		srcCvesPath := filepath.Join(fixtPath, "software", fmt.Sprintf("%s-software_cves.csv.bz2", p))
		dstCvesPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", p))
		extract(srcCvesPath, dstCvesPath)
	}

	loadSoftware := func(p Platform, s fleet.OSVersion) *fleet.Host {
		osqueryHostID, err := server.GenerateRandomText(10)
		require.NoError(t, err)

		h, err := ds.NewHost(context.Background(), &fleet.Host{
			Hostname:        string(p),
			NodeKey:         string(p),
			UUID:            string(p),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   osqueryHostID,
			Platform:        s.Platform,
			OSVersion:       s.Name,
		})
		require.NoError(t, err)

		var fixtures []softwareFixture
		contents, err := ioutil.ReadFile(filepath.Join(vulnPath, fmt.Sprintf("%s-software.json", p)))
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

		err = ds.LoadHostSoftware(ctx, h, false)
		require.NoError(t, err)

		for _, s := range h.Software {
			err = ds.AddCPEForSoftware(ctx, s, fmt.Sprintf("%s-%s", s.Name, s.Version))
			require.NoError(t, err)
		}

		return h
	}

	p := NewPlatform(version.Platform, version.Name)

	extractFixtures(p)
	h := loadSoftware(p, version)

	err := ds.UpdateOSVersions(ctx)
	require.NoError(t, err)

	afterLoad(h)

	err = ds.DeleteHost(ctx, h.ID)
	require.NoError(t, err)
}

func BenchmarkTestOvalAnalyzer(b *testing.B) {
	ds := mysql.CreateMySQLDS(b)
	defer mysql.TruncateTables(b, ds)

	vulnPath, err := ioutil.TempDir("", "oval_analyzer_ubuntu")
	defer os.RemoveAll(vulnPath)
	require.NoError(b, err)

	systems := []fleet.OSVersion{
		{Platform: "ubuntu", Name: "Ubuntu 16.4.0"},
		{Platform: "ubuntu", Name: "Ubuntu 18.4.0"},
		{Platform: "ubuntu", Name: "Ubuntu 20.4.0"},
		{Platform: "ubuntu", Name: "Ubuntu 21.4.0"},
		{Platform: "ubuntu", Name: "Ubuntu 21.10.0"},
		{Platform: "ubuntu", Name: "Ubuntu 22.4.0"},
	}

	for _, v := range systems {
		b.Run(fmt.Sprintf("for %s %s", v.Platform, v.Name), func(b *testing.B) {
			withTestFixture(v, vulnPath, ds, func(h *fleet.Host) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err = Analyze(context.Background(), ds, v, vulnPath, true)
					require.NoError(b, err)
				}
			}, b)
		})
	}
}

func TestOvalAnalyzer(t *testing.T) {
	// For generating the vulnerability lists I used VMs and ran oscap (since it seems like oscap
	// does not work with Docker) and extracted all installed software vulnerabilities, then I had
	// the VMs join my local dev env, and extracted the installed software from the database.
	t.Run("analyzing Ubuntu software", func(t *testing.T) {
		ds := mysql.CreateMySQLDS(t)
		defer mysql.TruncateTables(t, ds)

		vulnPath, err := ioutil.TempDir("", "oval_analyzer_ubuntu")
		defer os.RemoveAll(vulnPath)
		require.NoError(t, err)

		ctx := context.Background()

		systems := []fleet.OSVersion{
			{Platform: "ubuntu", Name: "Ubuntu 16.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 18.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 20.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 21.4.0"},
			{Platform: "ubuntu", Name: "Ubuntu 21.10.0"},
			{Platform: "ubuntu", Name: "Ubuntu 22.4.0"},
		}

		assertVulns := func(h *fleet.Host, p Platform) {
			fPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", p))
			f, err := os.Open(fPath)
			require.NoError(t, err)
			defer f.Close()

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

			storedVulns, err := ds.ListSoftwareVulnerabilities(ctx, []uint{h.ID})
			require.NoError(t, err)

			uniq := make(map[string]bool)
			for _, v := range storedVulns[h.ID] {
				uniq[v.CVE] = true
			}
			actual := make([]string, 0, len(uniq))
			for k := range uniq {
				actual = append(actual, k)
			}

			require.ElementsMatch(t, actual, expected)
		}

		for _, v := range systems {
			withTestFixture(v, vulnPath, ds, func(h *fleet.Host) {
				_, err = Analyze(ctx, ds, v, vulnPath, true)
				require.NoError(t, err)

				p := NewPlatform(v.Platform, v.Name)
				assertVulns(h, p)
			}, t)
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
				{CPEID: 1, CVE: "cve_1"},
				{CPEID: 1, CVE: "cve_2"},
				{CPEID: 2, CVE: "cve_3"},
				{CPEID: 2, CVE: "cve_4"},
			}

			existing := []fleet.SoftwareVulnerability{
				{CPEID: 1, CVE: "cve_1"},
				{CPEID: 1, CVE: "cve_2"},
				{CPEID: 2, CVE: "cve_3"},
				{CPEID: 2, CVE: "cve_4"},
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Empty(t, toInsert)
			require.Empty(t, toDelete)
		})

		t.Run("existing differ from found", func(t *testing.T) {
			found := []fleet.SoftwareVulnerability{
				{CPEID: 1, CVE: "cve_1"},
				{CPEID: 1, CVE: "cve_2"},
				{CPEID: 3, CVE: "cve_5"},
				{CPEID: 3, CVE: "cve_6"},
			}

			existing := []fleet.SoftwareVulnerability{
				{CPEID: 1, CVE: "cve_1"},
				{CPEID: 1, CVE: "cve_2"},
				{CPEID: 2, CVE: "cve_3"},
				{CPEID: 2, CVE: "cve_4"},
			}

			expectedToInsert := []fleet.SoftwareVulnerability{
				{CPEID: 3, CVE: "cve_5"},
				{CPEID: 3, CVE: "cve_6"},
			}

			expectedToDelete := []fleet.SoftwareVulnerability{
				{CPEID: 2, CVE: "cve_3"},
				{CPEID: 2, CVE: "cve_4"},
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Equal(t, expectedToInsert, toInsert)
			require.ElementsMatch(t, expectedToDelete, toDelete)
		})

		t.Run("nothing found but vulns exist", func(t *testing.T) {
			var found []fleet.SoftwareVulnerability

			existing := []fleet.SoftwareVulnerability{
				{CPEID: 1, CVE: "cve_1"},
				{CPEID: 1, CVE: "cve_2"},
				{CPEID: 2, CVE: "cve_3"},
				{CPEID: 2, CVE: "cve_4"},
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			require.Empty(t, toInsert)
			require.ElementsMatch(t, existing, toDelete)
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
