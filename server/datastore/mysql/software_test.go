package mysql

import (
	"context"
	"crypto/md5" // nolint:gosec (only used for tests)
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

func TestSoftware(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SaveHost", testSoftwareSaveHost},
		{"CPE", testSoftwareCPE},
		{"HostDuplicates", testSoftwareHostDuplicates},
		{"LoadVulnerabilities", testSoftwareLoadVulnerabilities},
		{"ListSoftwareCPEs", testListSoftwareCPEs},
		{"NothingChanged", testSoftwareNothingChanged},
		{"LoadSupportsTonsOfCVEs", testSoftwareLoadSupportsTonsOfCVEs},
		{"List", testSoftwareList},
		{"SyncHostsSoftware", testSoftwareSyncHostsSoftware},
		{"DeleteSoftwareVulnerabilities", testDeleteSoftwareVulnerabilities},
		{"HostsByCVE", testHostsByCVE},
		{"HostVulnSummariesBySoftwareIDs", testHostVulnSummariesBySoftwareIDs},
		{"UpdateHostSoftware", testUpdateHostSoftware},
		{"UpdateHostSoftwareDeadlock", testUpdateHostSoftwareDeadlock},
		{"UpdateHostSoftwareUpdatesSoftware", testUpdateHostSoftwareUpdatesSoftware},
		{"ListSoftwareByHostIDShort", testListSoftwareByHostIDShort},
		{"ListSoftwareVulnerabilitiesByHostIDsSource", testListSoftwareVulnerabilitiesByHostIDsSource},
		{"InsertSoftwareVulnerability", testInsertSoftwareVulnerability},
		{"ListCVEs", testListCVEs},
		{"ListSoftwareForVulnDetection", testListSoftwareForVulnDetection},
		{"AllSoftwareIterator", testAllSoftwareIterator},
		{"UpsertSoftwareCPEs", testUpsertSoftwareCPEs},
		{"DeleteOutOfDateVulnerabilities", testDeleteOutOfDateVulnerabilities},
		{"DeleteSoftwareCPEs", testDeleteSoftwareCPEs},
		{"SoftwareByIDNoDuplicatedVulns", testSoftwareByIDNoDuplicatedVulns},
		{"SoftwareByIDIncludesCVEPublishedDate", testSoftwareByIDIncludesCVEPublishedDate},
		{"getHostSoftwareInstalledPaths", testGetHostSoftwareInstalledPaths},
		{"hostSoftwareInstalledPathsDelta", testHostSoftwareInstalledPathsDelta},
		{"deleteHostSoftwareInstalledPaths", testDeleteHostSoftwareInstalledPaths},
		{"insertHostSoftwareInstalledPaths", testInsertHostSoftwareInstalledPaths},
		{"VerifySoftwareChecksum", testVerifySoftwareChecksum},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSoftwareSaveHost(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: ""},
	}

	getHostSoftware := func(h *fleet.Host) []fleet.Software {
		var software []fleet.Software
		for _, s := range h.Software {
			software = append(software, s.Software)
		}
		return software
	}

	_, err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	host1Software := getHostSoftware(host1)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1Software)

	soft1ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID, nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, soft1ByID)
	assert.Equal(t, host1Software[0], *soft1ByID)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	host2Software := getHostSoftware(host2)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}
	software2 = []fleet.Software{}

	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	host1Software = getHostSoftware(host1)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1Software)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	host2Software = getHostSoftware(host2)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}

	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	host1Software = getHostSoftware(host1)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: "com.zoo"}, // "empty" -> "non-empty"
	}
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	host2Software = getHostSoftware(host2)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.other"}, // "non-empty" -> "non-empty"
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: ""},               // non-empty -> empty
	}
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	host2Software = getHostSoftware(host2)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2Software)
}

func testSoftwareCPE(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}

	software2 := []fleet.Software{
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.other"}, // "non-empty" -> "non-empty"
		{Name: "zoo", Version: "0.0.5", Source: "rpm_packages", BundleIdentifier: ""},               // non-empty -> empty
	}

	_, err := ds.UpdateHostSoftware(context.Background(), host1.ID, append(software1, software2...))
	require.NoError(t, err)

	q := fleet.SoftwareIterQueryOptions{ExcludedSources: oval.SupportedSoftwareSources}
	iterator, err := ds.AllSoftwareIterator(context.Background(), q)
	require.NoError(t, err)
	defer iterator.Close()

	loops := 0
	for iterator.Next() {
		software, err := iterator.Value()
		require.NoError(t, err)
		require.NoError(t, iterator.Err())

		require.NotEmpty(t, software.ID)
		require.NotEmpty(t, software.Name)
		require.NotEmpty(t, software.Version)
		require.NotEmpty(t, software.Source)

		require.NotEqual(t, software.Name, "bar")
		require.NotEqual(t, software.Name, "zoo")

		if loops > 2 {
			t.Error("Looping through more software than we have")
		}
		loops++
	}
	assert.Equal(t, len(software1), loops)
	require.NoError(t, iterator.Close())
}

func testSoftwareHostDuplicates(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	longName := strings.Repeat("a", fleet.SoftwareNameMaxLength+5)

	incoming := make(map[string]fleet.Software)
	sw, err := fleet.SoftwareFromOsqueryRow(longName+"b", "0.0.1", "chrome_extension", "", "", "", "", "", "", "", "")
	require.NoError(t, err)
	soft2Key := sw.ToUniqueStr()
	incoming[soft2Key] = *sw

	tx, err := ds.writer(context.Background()).Beginx()
	require.NoError(t, err)
	_, err = insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]fleet.Software), incoming)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Check that the software entry was stored for the host.
	var software []fleet.Software
	err = sqlx.SelectContext(context.Background(), ds.reader(context.Background()),
		&software, `SELECT s.id, s.name FROM software s JOIN host_software hs WHERE hs.host_id = ?`,
		host1.ID,
	)
	require.NoError(t, err)
	require.Len(t, software, 1)
	require.NotZero(t, software[0].ID)
	require.Equal(t, strings.Repeat("a", fleet.SoftwareNameMaxLength), software[0].Name)

	incoming = make(map[string]fleet.Software)
	sw, err = fleet.SoftwareFromOsqueryRow(longName+"c", "0.0.1", "chrome_extension", "", "", "", "", "", "", "", "")
	require.NoError(t, err)
	soft3Key := sw.ToUniqueStr()
	incoming[soft3Key] = *sw

	tx, err = ds.writer(context.Background()).Beginx()
	require.NoError(t, err)
	_, err = insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]fleet.Software), incoming)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Check that the software entry was not modified with the new insert because of the name trimming.
	var software2 []fleet.Software
	err = sqlx.SelectContext(context.Background(), ds.reader(context.Background()),
		&software2, `SELECT s.id, s.name FROM software s JOIN host_software hs WHERE hs.host_id = ?`,
		host1.ID,
	)
	require.NoError(t, err)
	require.Len(t, software2, 1)
	require.Equal(t, strings.Repeat("a", fleet.SoftwareNameMaxLength), software2[0].Name)
	require.Equal(t, software[0].ID, software2[0].ID)
}

func testSoftwareLoadVulnerabilities(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host.Software[0].ID, CPE: "somecpe"},
		{SoftwareID: host.Software[1].ID, CPE: "someothercpewithoutvulns"},
	}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2022-0001"},
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2022-0002"},
	}
	for _, v := range vulns {
		_, err = ds.InsertSoftwareVulnerability(context.Background(), v, fleet.NVDSource)
		require.NoError(t, err)
	}
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	softByID, err := ds.SoftwareByID(context.Background(), host.HostSoftware.Software[0].ID, nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, softByID)
	require.Len(t, softByID.Vulnerabilities, 2)

	assert.Equal(t, "somecpe", host.Software[0].GenerateCPE)
	require.Len(t, host.Software[0].Vulnerabilities, 2)
	assert.Equal(t, "CVE-2022-0001", host.Software[0].Vulnerabilities[0].CVE)
	assert.Equal(t,
		"https://nvd.nist.gov/vuln/detail/CVE-2022-0001", host.Software[0].Vulnerabilities[0].DetailsLink)
	assert.Equal(t, "CVE-2022-0002", host.Software[0].Vulnerabilities[1].CVE)
	assert.Equal(t,
		"https://nvd.nist.gov/vuln/detail/CVE-2022-0002", host.Software[0].Vulnerabilities[1].DetailsLink)

	assert.Equal(t, "someothercpewithoutvulns", host.Software[1].GenerateCPE)
	require.Len(t, host.Software[1].Vulnerabilities, 0)
}

func testListSoftwareCPEs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	debian := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
	debian.Platform = "debian"
	require.NoError(t, ds.UpdateHost(ctx, debian))

	ubuntu := test.NewHost(t, ds, "host4", "", "host4key", "host4uuid", time.Now())
	ubuntu.Platform = "ubuntu"
	require.NoError(t, ds.UpdateHost(ctx, ubuntu))

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "biz", Version: "0.0.1", Source: "deb_packages"},
		{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
	}
	_, err := ds.UpdateHostSoftware(ctx, debian.ID, software[:2])
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, debian, false))

	_, err = ds.UpdateHostSoftware(ctx, ubuntu.ID, software[2:])
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, ubuntu, false))

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: debian.Software[0].ID, CPE: "cpe1"},
		{SoftwareID: debian.Software[1].ID, CPE: "cpe2"},
		{SoftwareID: ubuntu.Software[0].ID, CPE: "cpe3"},
		{SoftwareID: ubuntu.Software[1].ID, CPE: "cpe4"},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	cpes, err = ds.ListSoftwareCPEs(ctx)
	expected := []string{
		"cpe1", "cpe2", "cpe3", "cpe4",
	}
	var actual []string
	for _, v := range cpes {
		actual = append(actual, v.CPE)
	}
	require.NoError(t, err)
	assert.ElementsMatch(t, actual, expected)
}

func testSoftwareNothingChanged(t *testing.T, ds *Datastore) {
	cases := []struct {
		desc     string
		current  []fleet.Software
		incoming []fleet.Software
		want     bool
	}{
		{"both nil", nil, nil, true},
		{"different len", nil, []fleet.Software{{}}, false},

		{
			"identical",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			true,
		},
		{
			"different version",
			[]fleet.Software{{Name: "A", Version: "1.1", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			false,
		},
		{
			"new software",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}, {Name: "B", Version: "1.0", Source: "ASD"}},
			false,
		},
		{
			"removed software",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}, {Name: "B", Version: "1.0", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			false,
		},
		{
			"identical with similar last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			true,
		},
		{
			"identical with no new last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			true,
		},
		{
			"identical but added last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			false,
		},
		{
			"identical but significantly changed last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now().Add(-365 * 24 * time.Hour))}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			false,
		},
		{
			"identical but insignificantly changed last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now().Add(-time.Second))}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())}},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := nothingChanged(c.current, c.incoming, defaultMinLastOpenedAtDiff)
			if c.want {
				require.True(t, got)
			} else {
				require.False(t, got)
			}
		})
	}
}

func generateCVEMeta(n int) fleet.CVEMeta {
	CVEID := fmt.Sprintf("CVE-2022-%05d", n)
	cvssScore := ptr.Float64(rand.Float64() * 10)
	epssProbability := ptr.Float64(rand.Float64())
	cisaKnownExploit := ptr.Bool(rand.Intn(2) == 1)
	return fleet.CVEMeta{
		CVE:              CVEID,
		CVSSScore:        cvssScore,
		EPSSProbability:  epssProbability,
		CISAKnownExploit: cisaKnownExploit,
	}
}

func testSoftwareLoadSupportsTonsOfCVEs(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	sort.Slice(host.Software, func(i, j int) bool { return host.Software[i].Name < host.Software[j].Name })

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host.Software[1].ID, CPE: "someothercpewithoutvulns"},
		{SoftwareID: host.Software[0].ID, CPE: "somecpe"},
	}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	var cveMeta []fleet.CVEMeta
	for i := 0; i < 1000; i++ {
		cveMeta = append(cveMeta, generateCVEMeta(i))
	}

	err = ds.InsertCVEMeta(context.Background(), cveMeta)
	require.NoError(t, err)

	values := strings.TrimSuffix(strings.Repeat("(?, ?), ", len(cveMeta)), ", ")
	query := `INSERT INTO software_cve (software_id, cve) VALUES ` + values
	var args []interface{}
	for _, cve := range cveMeta {
		args = append(args, host.Software[0].ID, cve.CVE)
	}
	_, err = ds.writer(context.Background()).ExecContext(context.Background(), query, args...)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	for _, software := range host.Software {
		switch software.Name {
		case "bar":
			assert.Equal(t, "somecpe", software.GenerateCPE)
			require.Len(t, software.Vulnerabilities, 1000)
			assert.True(t, strings.HasPrefix(software.Vulnerabilities[0].CVE, "CVE-"))
			assert.Equal(t,
				"https://nvd.nist.gov/vuln/detail/"+software.Vulnerabilities[0].CVE,
				software.Vulnerabilities[0].DetailsLink,
			)
		case "blah":
			assert.Len(t, software.Vulnerabilities, 0)
			assert.Equal(t, "someothercpewithoutvulns", software.GenerateCPE)
		case "foo":
			assert.Len(t, software.Vulnerabilities, 0)
		}
	}
}

func testSoftwareList(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	software3 := []fleet.Software{
		{Name: "baz", Version: "0.0.1", Source: "deb_packages"},
	}

	_, err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host3.ID, software3)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host3, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host1.Software[0].ID, CPE: "somecpe"},
		{SoftwareID: host1.Software[1].ID, CPE: "someothercpewithoutvulns"},
		{SoftwareID: host3.Software[0].ID, CPE: "somecpe2"},
	}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host3, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})

	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: host1.Software[0].ID, CVE: "CVE-2022-0001", ResolvedInVersion: ptr.String("2.0.0")},
		{SoftwareID: host1.Software[0].ID, CVE: "CVE-2022-0002", ResolvedInVersion: ptr.String("2.0.0")},
		{SoftwareID: host3.Software[0].ID, CVE: "CVE-2022-0003", ResolvedInVersion: ptr.String("2.0.0")},
	}

	for _, v := range vulns {
		_, err = ds.InsertSoftwareVulnerability(context.Background(), v, fleet.NVDSource)
		require.NoError(t, err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	cveMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2022-0001",
			CVSSScore:        ptr.Float64(2.0),
			EPSSProbability:  ptr.Float64(0.01),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(now.Add(-2 * time.Hour)),
			Description:      "this is a description for CVE-2022-0001",
		},
		{
			CVE:              "CVE-2022-0002",
			CVSSScore:        ptr.Float64(1.0),
			EPSSProbability:  ptr.Float64(0.99),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(now),
			Description:      "this is a description for CVE-2022-0002",
		},
		{
			CVE:              "CVE-2022-0003",
			CVSSScore:        ptr.Float64(3.0),
			EPSSProbability:  ptr.Float64(0.98),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(now.Add(-1 * time.Hour)),
			Description:      "this is a description for CVE-2022-0003",
		},
	}
	err = ds.InsertCVEMeta(context.Background(), cveMeta)
	require.NoError(t, err)

	foo001 := fleet.Software{
		Name:        "foo",
		Version:     "0.0.1",
		Source:      "chrome_extensions",
		GenerateCPE: "somecpe",
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:               "CVE-2022-0001",
				DetailsLink:       "https://nvd.nist.gov/vuln/detail/CVE-2022-0001",
				CVSSScore:         ptr.Float64Ptr(2.0),
				EPSSProbability:   ptr.Float64Ptr(0.01),
				CISAKnownExploit:  ptr.BoolPtr(false),
				CVEPublished:      ptr.TimePtr(now.Add(-2 * time.Hour)),
				Description:       ptr.StringPtr("this is a description for CVE-2022-0001"),
				ResolvedInVersion: ptr.StringPtr("2.0.0"),
			},
			{
				CVE:               "CVE-2022-0002",
				DetailsLink:       "https://nvd.nist.gov/vuln/detail/CVE-2022-0002",
				CVSSScore:         ptr.Float64Ptr(1.0),
				EPSSProbability:   ptr.Float64Ptr(0.99),
				CISAKnownExploit:  ptr.BoolPtr(false),
				CVEPublished:      ptr.TimePtr(now),
				Description:       ptr.StringPtr("this is a description for CVE-2022-0002"),
				ResolvedInVersion: ptr.StringPtr("2.0.0"),
			},
		},
	}
	foo002 := fleet.Software{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"}
	foo003 := fleet.Software{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "someothercpewithoutvulns"}
	bar003 := fleet.Software{Name: "bar", Version: "0.0.3", Source: "deb_packages"}
	baz001 := fleet.Software{
		Name:        "baz",
		Version:     "0.0.1",
		Source:      "deb_packages",
		GenerateCPE: "somecpe2",
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:               "CVE-2022-0003",
				DetailsLink:       "https://nvd.nist.gov/vuln/detail/CVE-2022-0003",
				CVSSScore:         ptr.Float64Ptr(3.0),
				EPSSProbability:   ptr.Float64Ptr(0.98),
				CISAKnownExploit:  ptr.BoolPtr(true),
				CVEPublished:      ptr.TimePtr(now.Add(-1 * time.Hour)),
				Description:       ptr.StringPtr("this is a description for CVE-2022-0003"),
				ResolvedInVersion: ptr.StringPtr("2.0.0"),
			},
		},
	}

	require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))

	t.Run("lists everything", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey: "name,version",
			},
			IncludeCVEScores: true,
		}
		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		expected := []fleet.Software{bar003, baz001, foo001, foo002, foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("paginates", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				Page:            1,
				PerPage:         1,
				OrderKey:        "version",
				IncludeMetadata: true,
			},
			IncludeCVEScores: true,
		}
		software := listSoftwareCheckCount(t, ds, 1, 5, opts, true)
		require.Len(t, software, 1)
		var expected []fleet.Software
		// Both foo001 and baz001 have the same version, thus we check which one the database picked
		// for the second page.
		if software[0].Name == "foo" {
			expected = []fleet.Software{foo001}
		} else {
			expected = []fleet.Software{baz001}
		}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))

		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey: "version",
			},
			TeamID:           &team1.ID,
			IncludeCVEScores: true,
		}
		software := listSoftwareCheckCount(t, ds, 2, 2, opts, true)
		expected := []fleet.Software{foo001, foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team and paginates", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1-" + t.Name()})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))

		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				PerPage:         1,
				Page:            1,
				OrderKey:        "id",
				IncludeMetadata: true,
			},
			TeamID: &team1.ID,
		}
		software := listSoftwareCheckCount(t, ds, 1, 2, opts, true)
		expected := []fleet.Software{foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters vulnerable software", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey: "name",
			},
			VulnerableOnly:   true,
			IncludeCVEScores: true,
		}
		software := listSoftwareCheckCount(t, ds, 2, 2, opts, true)
		expected := []fleet.Software{foo001, baz001}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by CVE", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				MatchQuery: "CVE-2022-0001",
			},
			IncludeCVEScores: true,
		}
		software := listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected := []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		opts.ListOptions.MatchQuery = "CVE-2022-0002"
		software = listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// partial CVE
		opts.ListOptions.MatchQuery = "0002"
		software = listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// unknown CVE
		opts.ListOptions.MatchQuery = "CVE-2022-0000"
		listSoftwareCheckCount(t, ds, 0, 0, opts, true)
	})

	t.Run("filters by query", func(t *testing.T) {
		// query by name (case insensitive)
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				MatchQuery: "baR",
			},
		}
		software := listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected := []fleet.Software{bar003}
		test.ElementsMatchSkipID(t, software, expected)

		// query by version
		opts.ListOptions.MatchQuery = "0.0.3"
		software = listSoftwareCheckCount(t, ds, 2, 2, opts, true)
		expected = []fleet.Software{foo003, bar003}
		test.ElementsMatchSkipID(t, software, expected)

		// query by version (case insensitive)
		opts.ListOptions.MatchQuery = "V0.0.2"
		software = listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected = []fleet.Software{foo002}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("order by name and id", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name,id",
				OrderDirection: fleet.OrderAscending,
			},
		}
		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		assert.Equal(t, bar003.Name, software[0].Name)
		assert.Equal(t, bar003.Version, software[0].Version)

		assert.Equal(t, baz001.Name, software[1].Name)
		assert.Equal(t, baz001.Version, software[1].Version)

		// foo's ordered by id, descending
		assert.Greater(t, software[3].ID, software[2].ID)
		assert.Greater(t, software[4].ID, software[3].ID)
	})

	t.Run("order by hosts_count", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 5, 5, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}, WithHostCounts: true}, false)
		// ordered by counts descending, so foo003 is first
		assert.Equal(t, foo003.Name, software[0].Name)
		assert.Equal(t, 2, software[0].HostsCount)
	})

	t.Run("order by epss_probability", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "epss_probability",
				OrderDirection: fleet.OrderDescending,
			},
			IncludeCVEScores: true,
		}

		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		assert.Equal(t, foo001.Name, software[0].Name)
		assert.Equal(t, foo001.Version, software[0].Version)
	})

	t.Run("order by cvss_score", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "cvss_score",
				OrderDirection: fleet.OrderDescending,
			},
			IncludeCVEScores: true,
		}

		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		assert.Equal(t, baz001.Name, software[0].Name)
		assert.Equal(t, baz001.Version, software[0].Version)
	})

	t.Run("order by cisa_known_exploit", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "cisa_known_exploit",
				OrderDirection: fleet.OrderDescending,
			},
			IncludeCVEScores: true,
		}

		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		assert.Equal(t, baz001.Name, software[0].Name)
		assert.Equal(t, baz001.Version, software[0].Version)
	})

	t.Run("order by cve_published", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "cve_published",
				OrderDirection: fleet.OrderDescending,
			},
			IncludeCVEScores: true,
		}

		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		assert.Equal(t, foo001.Name, software[0].Name)
		assert.Equal(t, foo001.Version, software[0].Version)
	})

	t.Run("nil cve scores if IncludeCVEScores is false", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name,version",
				OrderDirection: fleet.OrderDescending,
			},
			IncludeCVEScores: false,
		}

		software := listSoftwareCheckCount(t, ds, 5, 5, opts, false)
		for _, s := range software {
			for _, vuln := range s.Vulnerabilities {
				assert.Nil(t, vuln.CVSSScore)
				assert.Nil(t, vuln.EPSSProbability)
				assert.Nil(t, vuln.CISAKnownExploit)
			}
		}
	})
}

func listSoftwareCheckCount(t *testing.T, ds *Datastore, expectedListCount int, expectedFullCount int, opts fleet.SoftwareListOptions, returnSorted bool) []fleet.Software {
	software, meta, err := ds.ListSoftware(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, software, expectedListCount)
	count, err := ds.CountSoftware(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, expectedFullCount, count)

	if opts.ListOptions.IncludeMetadata {
		require.NotNil(t, meta)
		if expectedListCount == expectedFullCount {
			require.False(t, meta.HasPreviousResults)
			require.True(t, meta.HasNextResults)
		}
		if expectedFullCount > expectedListCount {
			shouldHavePrevious := opts.ListOptions.Page > 0
			require.Equal(t, shouldHavePrevious, meta.HasPreviousResults)

			shouldHaveNext := uint(expectedFullCount) > (opts.ListOptions.Page+1)*opts.ListOptions.PerPage // page is 0-indexed
			require.Equal(t, shouldHaveNext, meta.HasNextResults)
		}
	} else {
		require.Nil(t, meta)
	}

	for _, s := range software {
		sort.Slice(s.Vulnerabilities, func(i, j int) bool {
			return s.Vulnerabilities[i].CVE < s.Vulnerabilities[j].CVE
		})
	}

	if returnSorted {
		sort.Slice(software, func(i, j int) bool {
			return software[i].Name+software[i].Version < software[j].Name+software[j].Version
		})
	}
	return software
}

func testSoftwareSyncHostsSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	cmpNameVersionCount := func(want, got []fleet.Software) {
		cmp := make([]fleet.Software, len(got))
		for i, sw := range got {
			cmp[i] = fleet.Software{Name: sw.Name, Version: sw.Version, HostsCount: sw.HostsCount}
		}
		require.ElementsMatch(t, want, cmp)
	}

	// this check ensures that the total number of rows in software_host_counts
	// matches the expected value.  we can't rely on ds.CountSoftware alone, as
	// that method (rightfully) ignores orphaned software counts.
	checkTableTotalCount := func(want int) {
		var tableCount int
		err := ds.writer(context.Background()).Get(&tableCount, "SELECT COUNT(*) FROM software_host_counts")
		require.NoError(t, err)
		require.Equal(t, want, tableCount)
	}

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	globalOpts := fleet.SoftwareListOptions{WithHostCounts: true, ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	globalCounts := listSoftwareCheckCount(t, ds, 4, 4, globalOpts, false)

	want := []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
		{Name: "bar", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(4)

	// update host2, remove "bar" software
	software2 = []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(3)

	// create a software entry without any host and any counts
	_, err = ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(`INSERT INTO software (name, version, source, checksum) VALUES ('baz', '0.0.1', 'testing', %s)`, softwareChecksumComputedColumn("")))
	require.NoError(t, err)

	// listing does not return the new software entry
	allSw := listSoftwareCheckCount(t, ds, 3, 3, fleet.SoftwareListOptions{}, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 0},
		{Name: "foo", Version: "0.0.1", HostsCount: 0},
		{Name: "foo", Version: "v0.0.2", HostsCount: 0},
	}
	cmpNameVersionCount(want, allSw)

	// create 2 teams and assign a new host to each
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{host3.ID}))
	host4 := test.NewHost(t, ds, "host4", "", "host4key", "host4uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team2.ID, []uint{host4.ID}))

	// assign existing host1 to team1 too, so we have a team with multiple hosts
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	// use some software for host3 and host4
	software3 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software4 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host3.ID, software3)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)

	// at this point, there's no counts per team, only global counts
	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(3)

	team1Opts := fleet.SoftwareListOptions{WithHostCounts: true, TeamID: ptr.Uint(team1.ID), ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	team1Counts := listSoftwareCheckCount(t, ds, 0, 0, team1Opts, false)
	want = []fleet.Software{}
	cmpNameVersionCount(want, team1Counts)
	checkTableTotalCount(3)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	nilSoftware, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID, &team1.ID, false, nil)
	assert.Nil(t, nilSoftware)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// after a call to Calculate, the global counts are updated and the team counts appear
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	globalCounts = listSoftwareCheckCount(t, ds, 4, 4, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 4},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
		{Name: "bar", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareCheckCount(t, ds, 2, 2, team1Opts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
	}
	cmpNameVersionCount(want, team1Counts)

	// composite pk (software_id, team_id), so we expect more rows
	checkTableTotalCount(8)

	soft1ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID, &team1.ID, false, nil)
	require.NoError(t, err)
	software1[0].ID = host1.HostSoftware.Software[0].ID
	assert.Equal(t, software1[0], *soft1ByID)

	team2Opts := fleet.SoftwareListOptions{WithHostCounts: true, TeamID: ptr.Uint(team2.ID), ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	team2Counts := listSoftwareCheckCount(t, ds, 2, 2, team2Opts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 1},
		{Name: "bar", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, team2Counts)

	// update host4 (team2), remove "bar" software
	software4 = []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 4},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareCheckCount(t, ds, 2, 2, team1Opts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
	}
	cmpNameVersionCount(want, team1Counts)

	team2Counts = listSoftwareCheckCount(t, ds, 1, 1, team2Opts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, team2Counts)

	checkTableTotalCount(6)

	// update host4 (team2), remove all software and delete team
	software4 = []fleet.Software{}
	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)
	require.NoError(t, ds.DeleteTeam(ctx, team2.ID))

	// this call will remove team2 from the software host counts table
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 3},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareCheckCount(t, ds, 2, 2, team1Opts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
	}
	cmpNameVersionCount(want, team1Counts)

	listSoftwareCheckCount(t, ds, 0, 0, team2Opts, false)
	checkTableTotalCount(5)
}

func insertVulnSoftwareForTest(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithComputerName("computer1"))
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{
			Name:        "foo.rpm",
			Version:     "0.0.1",
			Source:      "rpm_packages",
			GenerateCPE: "cpe_foo_rpm",
		},
		{
			Name:        "foo.chrome",
			Version:     "0.0.3",
			Source:      "chrome_extensions",
			GenerateCPE: "cpe_foo_chrome_3",
		},
	}
	software2 := []fleet.Software{
		{
			Name:        "foo.chrome",
			Version:     "0.0.2",
			Source:      "chrome_extensions",
			GenerateCPE: "cpe_foo_chrome_2",
		},
		{
			Name:        "foo.chrome",
			Version:     "0.0.3",
			Source:      "chrome_extensions",
			GenerateCPE: "cpe_foo_chrome_3",
			Vulnerabilities: fleet.Vulnerabilities{
				{
					CVE:         "CVE-2022-0001",
					DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2022-0001",
				},
			},
		},
		{
			Name:        "bar.rpm",
			Version:     "0.0.3",
			Source:      "rpm_packages",
			GenerateCPE: "cpe_bar_rpm",
			Vulnerabilities: fleet.Vulnerabilities{
				{
					CVE:         "CVE-2022-0002",
					DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2022-0002",
				},
				{
					CVE:         "CVE-2022-0003",
					DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-333-444-555",
				},
			},
		},
	}

	mutationResults, err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)

	// Insert paths for software1
	s1Paths := map[string]struct{}{}
	for _, s := range software1 {
		key := fmt.Sprintf("%s%s%s", fmt.Sprintf("/some/path/%s", s.Name), fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		s1Paths[key] = struct{}{}
	}
	require.NoError(t, ds.UpdateHostSoftwareInstalledPaths(context.Background(), host1.ID, s1Paths, mutationResults))

	mutationResults, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)

	// Insert paths for software2
	s2Paths := map[string]struct{}{}
	for _, s := range software2 {
		key := fmt.Sprintf("%s%s%s", fmt.Sprintf("/some/path/%s", s.Name), fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		s2Paths[key] = struct{}{}
	}
	require.NoError(t, ds.UpdateHostSoftwareInstalledPaths(context.Background(), host2.ID, s2Paths, mutationResults))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host1.Software[0].ID, CPE: "cpe_foo_chrome_3"},
		{SoftwareID: host1.Software[1].ID, CPE: "cpe_foo_rpm"},
		{SoftwareID: host2.Software[0].ID, CPE: "cpe_bar_rpm"},
		{SoftwareID: host2.Software[1].ID, CPE: "cpe_foo_chrome_2"},
		{SoftwareID: host2.Software[2].ID, CPE: "cpe_foo_chrome_3"},
	}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	chrome3 := host2.Software[2]
	inserted, err := ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: chrome3.ID,
		CVE:        "CVE-2022-0001",
	}, fleet.NVDSource)

	require.NoError(t, err)
	require.True(t, inserted)

	barRpm := host2.Software[0]
	vulns := []fleet.SoftwareVulnerability{
		{
			SoftwareID: barRpm.ID,
			CVE:        "CVE-2022-0002",
		},
		{
			SoftwareID: barRpm.ID,
			CVE:        "CVE-2022-0003",
		},
	}

	for _, v := range vulns {
		inserted, err := ds.InsertSoftwareVulnerability(context.Background(), v, fleet.NVDSource)
		require.NoError(t, err)
		require.True(t, inserted)
	}

	require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))
}

func testDeleteSoftwareVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	err := ds.DeleteSoftwareVulnerabilities(ctx, nil)
	require.NoError(t, err)

	insertVulnSoftwareForTest(t, ds)

	err = ds.DeleteSoftwareVulnerabilities(ctx, []fleet.SoftwareVulnerability{
		{
			SoftwareID: 999, // unknown software
			CVE:        "CVE-2022-0003",
		},
	})
	require.NoError(t, err)

	host2, err := ds.HostByIdentifier(ctx, "host2")
	require.NoError(t, err)

	err = ds.LoadHostSoftware(ctx, host2, false)
	require.NoError(t, err)
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	barRPM := host2.Software[0]
	require.Len(t, barRPM.Vulnerabilities, 2)

	err = ds.DeleteSoftwareVulnerabilities(ctx, []fleet.SoftwareVulnerability{
		{
			SoftwareID: barRPM.ID,
			CVE:        "CVE-0000-0000", // unknown CVE
		},
	})
	require.NoError(t, err)

	err = ds.DeleteSoftwareVulnerabilities(ctx, []fleet.SoftwareVulnerability{
		{
			SoftwareID: barRPM.ID,
			CVE:        "CVE-2022-0003",
		},
	})
	require.NoError(t, err)

	err = ds.LoadHostSoftware(ctx, host2, false)
	require.NoError(t, err)
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	barRPM = host2.Software[0]
	require.Len(t, barRPM.Vulnerabilities, 1)

	err = ds.DeleteSoftwareVulnerabilities(ctx, []fleet.SoftwareVulnerability{
		{
			SoftwareID: barRPM.ID,
			CVE:        "CVE-2022-0002",
		},
	})
	require.NoError(t, err)

	err = ds.LoadHostSoftware(ctx, host2, false)
	require.NoError(t, err)
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	barRPM = host2.Software[0]
	require.Empty(t, barRPM.Vulnerabilities)
}

func testHostsByCVE(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hosts, err := ds.HostsByCVE(ctx, "CVE-0000-0000")
	require.NoError(t, err)
	require.Len(t, hosts, 0)

	insertVulnSoftwareForTest(t, ds)

	// CVE of foo chrome 0.0.3, both hosts have it
	hosts, err = ds.HostsByCVE(ctx, "CVE-2022-0001")
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.ElementsMatch(t, hosts, []fleet.HostVulnerabilitySummary{
		{
			ID:          1,
			Hostname:    "host1",
			DisplayName: "computer1",
			SoftwareInstalledPaths: []string{
				"/some/path/foo.chrome",
			},
		}, {
			ID:          2,
			Hostname:    "host2",
			DisplayName: "host2",
			SoftwareInstalledPaths: []string{
				"/some/path/foo.chrome",
			},
		},
	})

	// CVE of bar.rpm 0.0.3, only host 2 has it
	hosts, err = ds.HostsByCVE(ctx, "CVE-2022-0002")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hosts[0].Hostname, "host2")
}

func testHostVulnSummariesBySoftwareIDs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Invalid non-existing host id
	hosts, err := ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{0})
	require.NoError(t, err)
	require.Len(t, hosts, 0)

	insertVulnSoftwareForTest(t, ds)

	allSoftware, _, err := ds.ListSoftware(ctx, fleet.SoftwareListOptions{})
	require.NoError(t, err)

	var fooRpm fleet.Software
	var chrome3 fleet.Software
	var barRpm fleet.Software
	for _, s := range allSoftware {
		switch s.GenerateCPE {
		case "cpe_foo_rpm":
			fooRpm = s
		case "cpe_foo_chrome_3":
			chrome3 = s
		case "cpe_bar_rpm":
			barRpm = s
		}
	}
	require.NotZero(t, chrome3.ID)
	require.NotZero(t, barRpm.ID)

	hosts, err = ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{chrome3.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, hosts, []fleet.HostVulnerabilitySummary{
		{
			ID:                     1,
			Hostname:               "host1",
			DisplayName:            "computer1",
			SoftwareInstalledPaths: []string{"/some/path/foo.chrome"},
		}, {
			ID:                     2,
			Hostname:               "host2",
			DisplayName:            "host2",
			SoftwareInstalledPaths: []string{"/some/path/foo.chrome"},
		},
	})

	hosts, err = ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{barRpm.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, hosts, []fleet.HostVulnerabilitySummary{
		{
			ID:                     2,
			Hostname:               "host2",
			DisplayName:            "host2",
			SoftwareInstalledPaths: []string{"/some/path/bar.rpm"},
		},
	})

	// Duplicates should not be returned if cpes are found on the same host ie host2 should only appear once
	hosts, err = ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{chrome3.ID, barRpm.ID, fooRpm.ID})
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.Equal(t, hosts[0].Hostname, "host1")
	require.Equal(t, hosts[1].Hostname, "host2")
	require.ElementsMatch(t, hosts[0].SoftwareInstalledPaths, []string{"/some/path/foo.rpm", "/some/path/foo.chrome"})
	require.ElementsMatch(t, hosts[1].SoftwareInstalledPaths, []string{"/some/path/bar.rpm", "/some/path/foo.chrome"})
}

// testUpdateHostSoftwareUpdatesSoftware tests that uninstalling applications
// from hosts (ds.UpdateHostSoftware) will remove the corresponding entry in
// `software` if no more hosts have the application installed.
func testUpdateHostSoftwareUpdatesSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	h1 := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())
	h2 := test.NewHost(t, ds, "host2", "", "hostkey2", "hostuuid2", time.Now())

	// Set the initial software list.
	sw1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar"},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz"},
	}
	_, err := ds.UpdateHostSoftware(ctx, h1.ID, sw1)
	require.NoError(t, err)
	sw2 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar"},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz"},
		{Name: "baz2", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz"},
	}
	_, err = ds.UpdateHostSoftware(ctx, h2.ID, sw2)
	require.NoError(t, err)

	// ListSoftware uses host_software_counts table.
	err = ds.SyncHostsSoftware(ctx, time.Now())
	require.NoError(t, err)

	// Check the returned software.
	cmpNameVersionCount := func(expected, got []fleet.Software) {
		cmp := make([]fleet.Software, len(got))
		for i, sw := range got {
			cmp[i] = fleet.Software{Name: sw.Name, Version: sw.Version, HostsCount: sw.HostsCount}
		}
		require.ElementsMatch(t, expected, cmp)
	}
	opts := fleet.SoftwareListOptions{WithHostCounts: true}
	software := listSoftwareCheckCount(t, ds, 4, 4, opts, false)
	expectedSoftware := []fleet.Software{
		{Name: "foo", Version: "0.0.1", HostsCount: 2},
		{Name: "bar", Version: "0.0.2", HostsCount: 2},
		{Name: "baz", Version: "0.0.3", HostsCount: 2},
		{Name: "baz2", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(expectedSoftware, software)

	// Update software for the two hosts.
	//
	//	- foo is still present in both hosts
	//	- new is added to h1.
	//	- baz is removed from h2.
	//	- baz2 is removed from h2.
	//	- bar is removed from both hosts.
	sw1Updated := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz"},
		{Name: "new", Version: "0.0.4", Source: "test", GenerateCPE: "cpe_new"},
	}
	_, err = ds.UpdateHostSoftware(ctx, h1.ID, sw1Updated)
	require.NoError(t, err)
	sw2Updated := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
	}
	_, err = ds.UpdateHostSoftware(ctx, h2.ID, sw2Updated)
	require.NoError(t, err)

	var (
		bazSoftwareID  uint
		barSoftwareID  uint
		baz2SoftwareID uint
	)
	for _, s := range software {
		if s.Name == "baz" {
			bazSoftwareID = s.ID
		}
		if s.Name == "baz2" {
			baz2SoftwareID = s.ID
		}
		if s.Name == "bar" {
			barSoftwareID = s.ID
		}
	}
	require.NotZero(t, bazSoftwareID)
	require.NotZero(t, barSoftwareID)
	require.NotZero(t, baz2SoftwareID)

	// "new" is not returned until ds.SyncHostsSoftware is executed.
	// "baz2" is gone from the software list.
	// "baz" still has the wrong count because ds.SyncHostsSoftware hasn't run yet.
	//
	// So... counts are "off" until ds.SyncHostsSoftware is run.
	software = listSoftwareCheckCount(t, ds, 2, 2, opts, false)
	expectedSoftware = []fleet.Software{
		{Name: "foo", Version: "0.0.1", HostsCount: 2},
		{Name: "baz", Version: "0.0.3", HostsCount: 2},
	}
	cmpNameVersionCount(expectedSoftware, software)

	hosts, err := ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{bazSoftwareID})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hosts[0].ID, h1.ID)

	hosts, err = ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{barSoftwareID})
	require.NoError(t, err)
	require.Empty(t, hosts)
	hosts, err = ds.HostVulnSummariesBySoftwareIDs(ctx, []uint{baz2SoftwareID})
	require.NoError(t, err)
	require.Empty(t, hosts)

	// ListSoftware uses host_software_counts table.
	err = ds.SyncHostsSoftware(ctx, time.Now())
	require.NoError(t, err)

	software = listSoftwareCheckCount(t, ds, 3, 3, opts, false)
	expectedSoftware = []fleet.Software{
		{Name: "foo", Version: "0.0.1", HostsCount: 2},
		{Name: "baz", Version: "0.0.3", HostsCount: 1},
		{Name: "new", Version: "0.0.4", HostsCount: 1},
	}
	cmpNameVersionCount(expectedSoftware, software)
}

func testUpdateHostSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)

	// sort software slice by last opened at timestamp
	genSortFn := func(sl []fleet.HostSoftwareEntry) func(l, r int) bool {
		return func(l, r int) bool {
			lsw, rsw := sl[l], sl[r]
			lts, rts := lsw.LastOpenedAt, rsw.LastOpenedAt
			switch {
			case lts == nil && rts == nil:
				return true
			case lts == nil && rts != nil:
				return true
			case lts != nil && rts == nil:
				return false
			default:
				return (*lts).Before(*rts) || ((*lts).Equal(*rts) && lsw.Name < rsw.Name)
			}
		}
	}

	host := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())

	type tup struct {
		name string
		ts   time.Time
	}
	validateSoftware := func(expect ...tup) {
		err := ds.LoadHostSoftware(ctx, host, false)
		require.NoError(t, err)

		require.Len(t, host.Software, len(expect))
		sort.Slice(host.Software, genSortFn(host.Software))

		for i, sw := range host.Software {
			want := expect[i]
			require.Equal(t, want.name, sw.Name)

			if want.ts.IsZero() {
				require.Nil(t, sw.LastOpenedAt)
			} else {
				require.WithinDuration(t, want.ts, *sw.LastOpenedAt, time.Second)
			}
		}
	}

	// set the initial software list
	sw := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar", LastOpenedAt: &lastYear},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz", LastOpenedAt: &now},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, sw)
	require.NoError(t, err)
	validateSoftware(tup{name: "foo"}, tup{"bar", lastYear}, tup{"baz", now})

	// make changes: remove foo, add qux, no new timestamp on bar, small ts change on baz
	nowish := now.Add(3 * time.Second)
	sw = []fleet.Software{
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar"},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz", LastOpenedAt: &nowish},
		{Name: "qux", Version: "0.0.4", Source: "test", GenerateCPE: "cpe_qux"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, sw)
	require.NoError(t, err)
	validateSoftware(tup{name: "qux"}, tup{"bar", lastYear}, tup{"baz", now}) // baz hasn't been updated to nowish, too small diff

	// more changes: bar receives a date further in the past, baz and qux to future
	lastLastYear := lastYear.Add(-365 * 24 * time.Hour)
	future := now.Add(3 * 24 * time.Hour)
	sw = []fleet.Software{
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar", LastOpenedAt: &lastLastYear},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz", LastOpenedAt: &future},
		{Name: "qux", Version: "0.0.4", Source: "test", GenerateCPE: "cpe_qux", LastOpenedAt: &future},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, sw)
	require.NoError(t, err)
	validateSoftware(tup{"bar", lastYear}, tup{"baz", future}, tup{"qux", future})
}

func testListSoftwareByHostIDShort(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))

	software, err := ds.ListSoftwareByHostIDShort(context.Background(), host1.ID)
	require.NoError(t, err)
	test.ElementsMatchSkipID(t, software1, software)

	software, err = ds.ListSoftwareByHostIDShort(context.Background(), host2.ID)
	require.NoError(t, err)
	test.ElementsMatchSkipID(t, software2, software)

	// bad host id returns no software
	badHostID := uint(3)
	software, err = ds.ListSoftwareByHostIDShort(context.Background(), badHostID)
	require.NoError(t, err)
	require.Len(t, software, 0)
}

func testListSoftwareVulnerabilitiesByHostIDsSource(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host.Software[0].ID, CPE: "foo_cpe"},
		{SoftwareID: host.Software[1].ID, CPE: "bar_cpe"},
		{SoftwareID: host.Software[2].ID, CPE: "blah_cpe"},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	cveMap := map[int]string{
		0: "cve-123",
		1: "cve-456",
	}

	for i, s := range host.Software {
		cve, ok := cveMap[i]
		if ok {
			inserted, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        cve,
			}, fleet.NVDSource)
			require.NoError(t, err)
			require.True(t, inserted)
		}

	}
	result, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.NVDSource)
	require.NoError(t, err)

	var actualCVEs []string
	for _, r := range result[host.ID] {
		actualCVEs = append(actualCVEs, r.CVE)
	}

	expectedCVEs := []string{"cve-123", "cve-456"}
	require.ElementsMatch(t, expectedCVEs, actualCVEs)

	for _, r := range result[host.ID] {
		require.NotEqual(t, r.SoftwareID, 0)
	}
}

func testInsertSoftwareVulnerability(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("no vulnerabilities to insert", func(t *testing.T) {
		inserted, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.False(t, inserted)
	})

	t.Run("duplicated vulnerabilities", func(t *testing.T) {
		host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
		software := fleet.Software{
			Name: "foo", Version: "0.0.1", Source: "chrome_extensions",
		}

		_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{software})
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		cpes := []fleet.SoftwareCPE{
			{SoftwareID: host.Software[0].ID, CPE: "foo_cpe_1"},
		}
		_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
		require.NoError(t, err)

		inserted, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: host.Software[0].ID, CVE: "cve-1",
		}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.True(t, inserted)

		inserted, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: host.Software[0].ID, CVE: "cve-1",
		}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.False(t, inserted)

		storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)

		occurrence := make(map[string]int)
		for _, v := range storedVulns[host.ID] {
			occurrence[v.CVE] = occurrence[v.CVE] + 1
		}
		require.Equal(t, 1, occurrence["cve-1"])
	})

	t.Run("a vulnerability already exists", func(t *testing.T) {
		host := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
		software := fleet.Software{
			Name: "foo", Version: "0.0.1", Source: "chrome_extensions",
		}

		_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{software})
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		cpes := []fleet.SoftwareCPE{
			{SoftwareID: host.Software[0].ID, CPE: "foo_cpe_2"},
		}
		_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
		require.NoError(t, err)

		var vulns []fleet.SoftwareVulnerability
		for _, s := range host.Software {
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        "cve-2",
			})
		}

		inserted, err := ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.True(t, inserted)

		inserted, err = ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.False(t, inserted)

		storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)

		occurrence := make(map[string]int)
		for _, v := range storedVulns[host.ID] {
			occurrence[v.CVE] = occurrence[v.CVE] + 1
		}
		require.Equal(t, 1, occurrence["cve-1"])
		require.Equal(t, 1, occurrence["cve-2"])
	})

	t.Run("vulnerability includes version range", func(t *testing.T) {
		// new host
		host := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

		// new software
		software := fleet.Software{
			Name: "host3software", Version: "0.0.1", Source: "chrome_extensions",
		}

		_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{software})
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

		// new software cpe
		cpes := []fleet.SoftwareCPE{
			{SoftwareID: host.Software[0].ID, CPE: "cpe:2.3:a:foo:foo:0.0.1:*:*:*:*:*:*:*"},
		}

		_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
		require.NoError(t, err)

		// new vulnerability
		vuln := fleet.SoftwareVulnerability{
			SoftwareID:        host.Software[0].ID,
			CVE:               "cve-3",
			ResolvedInVersion: ptr.String("1.2.3"),
		}

		inserted, err := ds.InsertSoftwareVulnerability(ctx, vuln, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.True(t, inserted)

		// vulnerability with no ResolvedInVersion
		vuln = fleet.SoftwareVulnerability{
			SoftwareID: host.Software[0].ID,
			CVE:        "cve-4",
		}

		inserted, err = ds.InsertSoftwareVulnerability(ctx, vuln, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.True(t, inserted)

		storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)

		require.Len(t, storedVulns[host.ID], 2)
		require.Equal(t, "cve-3", storedVulns[host.ID][0].CVE)
		require.Equal(t, "1.2.3", *storedVulns[host.ID][0].ResolvedInVersion)
		require.Equal(t, "cve-4", storedVulns[host.ID][1].CVE)
		require.Nil(t, storedVulns[host.ID][1].ResolvedInVersion)
	})
}

func testListCVEs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	now := time.Now().UTC()
	threeDaysAgo := now.Add(-3 * 24 * time.Hour)
	twoWeeksAgo := now.Add(-14 * 24 * time.Hour)
	twoMonthsAgo := now.Add(-60 * 24 * time.Hour)

	testCases := []fleet.CVEMeta{
		{CVE: "cve-1", Published: &threeDaysAgo, Description: "cve-1 description"},
		{CVE: "cve-2", Published: &twoWeeksAgo, Description: "cve-2 description"},
		{CVE: "cve-3", Published: &twoMonthsAgo}, // past maxAge
		{CVE: "cve-4"},                           // no published date
	}

	err := ds.InsertCVEMeta(ctx, testCases)
	require.NoError(t, err)

	result, err := ds.ListCVEs(ctx, 30*24*time.Hour)
	require.NoError(t, err)

	expected := []string{"cve-1", "cve-1 description", "cve-2", "cve-2 description"}
	var actual []string
	for _, r := range result {
		actual = append(actual, r.CVE)
		actual = append(actual, r.Description)
	}
	require.ElementsMatch(t, expected, actual)
}

func testListSoftwareForVulnDetection(t *testing.T, ds *Datastore) {
	t.Run("returns software without CPE entries", func(t *testing.T) {
		ctx := context.Background()

		host := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
		host.Platform = "debian"
		require.NoError(t, ds.UpdateHost(ctx, host))

		software := []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
			{Name: "biz", Version: "0.0.1", Source: "deb_packages"},
			{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
		}
		_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		_, err = ds.UpsertSoftwareCPEs(ctx, []fleet.SoftwareCPE{{SoftwareID: host.Software[0].ID, CPE: "cpe1"}})
		require.NoError(t, err)
		// Load software again so that CPE data is included.
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

		result, err := ds.ListSoftwareForVulnDetection(ctx, host.ID)
		require.NoError(t, err)

		sort.Slice(host.Software, func(i, j int) bool { return host.Software[i].ID < host.Software[j].ID })
		sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })

		require.Equal(t, len(host.Software), len(result))

		for i := range host.Software {
			require.Equal(t, host.Software[i].ID, result[i].ID)
			require.Equal(t, host.Software[i].Name, result[i].Name)
			require.Equal(t, host.Software[i].Version, result[i].Version)
			require.Equal(t, host.Software[i].Release, result[i].Release)
			require.Equal(t, host.Software[i].Arch, result[i].Arch)
			require.Equal(t, host.Software[i].GenerateCPE, result[i].GenerateCPE)
		}
	})
}

func testSoftwareByIDNoDuplicatedVulns(t *testing.T, ds *Datastore) {
	t.Run("software installed in multiple hosts does not have duplicated vulnerabilities", func(t *testing.T) {
		ctx := context.Background()
		hostA := test.NewHost(t, ds, "hostA", "", "hostAkey", "hostAuuid", time.Now())
		hostA.Platform = "ubuntu"
		require.NoError(t, ds.UpdateHost(ctx, hostA))

		hostB := test.NewHost(t, ds, "hostB", "", "hostBkey", "hostBuuid", time.Now())
		hostB.Platform = "ubuntu"
		require.NoError(t, ds.UpdateHost(ctx, hostB))

		software := []fleet.Software{
			{Name: "foo_123", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar_123", Version: "0.0.3", Source: "apps"},
			{Name: "biz_123", Version: "0.0.1", Source: "deb_packages"},
			{Name: "baz_123", Version: "0.0.3", Source: "deb_packages"},
		}

		_, err := ds.UpdateHostSoftware(ctx, hostA.ID, software)
		require.NoError(t, err)
		_, err = ds.UpdateHostSoftware(ctx, hostB.ID, software)
		require.NoError(t, err)

		require.NoError(t, ds.LoadHostSoftware(ctx, hostA, false))
		require.NoError(t, ds.LoadHostSoftware(ctx, hostB, false))

		// Add one vulnerability to each software
		for i, s := range hostA.Software {
			inserted, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        fmt.Sprintf("cve-%d", i),
			}, fleet.UbuntuOVALSource)
			require.NoError(t, err)
			require.True(t, inserted)
		}

		for _, s := range hostA.Software {
			result, err := ds.SoftwareByID(ctx, s.ID, nil, true, nil)
			require.NoError(t, err)
			require.Len(t, result.Vulnerabilities, 1)
		}
	})
}

func testSoftwareByIDIncludesCVEPublishedDate(t *testing.T, ds *Datastore) {
	t.Run("software.vulnerabilities includes the published date", func(t *testing.T) {
		ctx := context.Background()
		host := test.NewHost(t, ds, "hostA", "", "hostAkey", "hostAuuid", time.Now())
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host.ID}))
		now := time.Now().UTC().Truncate(time.Second)

		testCases := []struct {
			name             string
			hasVuln          bool
			hasMeta          bool
			hasPublishedDate bool
		}{
			{"foo_123", true, true, true},
			{"bar_123", true, true, false},
			{"foo_456", true, false, false},
			{"bar_456", false, true, true},
			{"foo_789", false, true, false},
			{"bar_789", false, false, false},
		}

		// Add software
		var software []fleet.Software
		for _, t := range testCases {
			software = append(software, fleet.Software{
				Name:    t.name,
				Version: "0.0.1",
				Source:  "apps",
			})
		}
		_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

		// Add vulnerabilities and CVEMeta
		var meta []fleet.CVEMeta
		for _, tC := range testCases {
			idx := -1
			for i, s := range host.Software {
				if s.Name == tC.name {
					idx = i
					break
				}
			}
			require.NotEqual(t, -1, idx, "software not found")

			if tC.hasVuln {
				inserted, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
					SoftwareID: host.Software[idx].ID,
					CVE:        fmt.Sprintf("cve-%s", tC.name),
				}, fleet.UbuntuOVALSource)
				require.NoError(t, err)
				require.True(t, inserted)
			}

			if tC.hasMeta {
				var published *time.Time
				if tC.hasPublishedDate {
					published = &now
				}

				meta = append(meta, fleet.CVEMeta{
					CVE:              fmt.Sprintf("cve-%s", tC.name),
					CVSSScore:        ptr.Float64(5.4),
					EPSSProbability:  ptr.Float64(0.5),
					CISAKnownExploit: ptr.Bool(true),
					Published:        published,
				})
			}
		}
		require.NoError(t, ds.InsertCVEMeta(ctx, meta))

		for _, tC := range testCases {
			idx := -1
			for i, s := range host.Software {
				if s.Name == tC.name {
					idx = i
					break
				}
			}
			require.NotEqual(t, -1, idx, "software not found")

			for _, teamID := range []*uint{nil, &team1.ID} {
				// Test that scores are not included if includeCVEScores = false
				withoutScores, err := ds.SoftwareByID(ctx, host.Software[idx].ID, teamID, false, nil)
				require.NoError(t, err)
				if tC.hasVuln {
					require.Len(t, withoutScores.Vulnerabilities, 1)
					require.Equal(t, fmt.Sprintf("cve-%s", tC.name), withoutScores.Vulnerabilities[0].CVE)

					require.Nil(t, withoutScores.Vulnerabilities[0].CVSSScore)
					require.Nil(t, withoutScores.Vulnerabilities[0].EPSSProbability)
					require.Nil(t, withoutScores.Vulnerabilities[0].CISAKnownExploit)
				} else {
					require.Empty(t, withoutScores.Vulnerabilities)
				}

				withScores, err := ds.SoftwareByID(ctx, host.Software[idx].ID, teamID, true, nil)
				require.NoError(t, err)
				if tC.hasVuln {
					require.Len(t, withScores.Vulnerabilities, 1)
					require.Equal(t, fmt.Sprintf("cve-%s", tC.name), withoutScores.Vulnerabilities[0].CVE)

					if tC.hasMeta {
						require.NotNil(t, withScores.Vulnerabilities[0].CVSSScore)
						require.NotNil(t, *withScores.Vulnerabilities[0].CVSSScore)
						require.Equal(t, **withScores.Vulnerabilities[0].CVSSScore, 5.4)

						require.NotNil(t, withScores.Vulnerabilities[0].EPSSProbability)
						require.NotNil(t, *withScores.Vulnerabilities[0].EPSSProbability)
						require.Equal(t, **withScores.Vulnerabilities[0].EPSSProbability, 0.5)

						require.NotNil(t, withScores.Vulnerabilities[0].CISAKnownExploit)
						require.NotNil(t, *withScores.Vulnerabilities[0].CISAKnownExploit)
						require.Equal(t, **withScores.Vulnerabilities[0].CISAKnownExploit, true)

						if tC.hasPublishedDate {
							require.NotNil(t, withScores.Vulnerabilities[0].CVEPublished)
							require.NotNil(t, *withScores.Vulnerabilities[0].CVEPublished)
							require.Equal(t, (**withScores.Vulnerabilities[0].CVEPublished), now)
						}
					}
				} else {
					require.Empty(t, withoutScores.Vulnerabilities)
				}
			}
		}
	})
}

func testAllSoftwareIterator(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "foo", Version: "v0.0.2", Source: "apps"},
		{Name: "foo", Version: "0.0.3", Source: "apps"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	foo_ce_v1 := slices.IndexFunc(host.Software, func(c fleet.HostSoftwareEntry) bool {
		return c.Name == "foo" && c.Version == "0.0.1" && c.Source == "chrome_extensions"
	})
	foo_app_v2 := slices.IndexFunc(host.Software, func(c fleet.HostSoftwareEntry) bool {
		return c.Name == "foo" && c.Version == "v0.0.2" && c.Source == "apps"
	})
	bar_v3 := slices.IndexFunc(host.Software, func(c fleet.HostSoftwareEntry) bool {
		return c.Name == "bar" && c.Version == "0.0.3" && c.Source == "deb_packages"
	})

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host.Software[foo_ce_v1].ID, CPE: "cpe:foo_ce_v1"},
		{SoftwareID: host.Software[foo_app_v2].ID, CPE: "cpe:foo_app_v2"},
		{SoftwareID: host.Software[bar_v3].ID, CPE: "cpe:bar_v3"},
	}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	testCases := []struct {
		q        fleet.SoftwareIterQueryOptions
		expected []fleet.Software
	}{
		{
			expected: []fleet.Software{
				{Name: "foo", Version: "v0.0.2", Source: "apps", GenerateCPE: "cpe:foo_app_v2"},
				{Name: "foo", Version: "0.0.3", Source: "apps"},
			},
			q: fleet.SoftwareIterQueryOptions{IncludedSources: []string{"apps"}},
		},
		{
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "bar", Version: "0.0.3", Source: "deb_packages", GenerateCPE: "cpe:bar_v3"},
			},
			q: fleet.SoftwareIterQueryOptions{ExcludedSources: []string{"apps"}},
		},
		{
			expected: []fleet.Software{
				{Name: "foo", Version: "v0.0.2", Source: "apps", GenerateCPE: "cpe:foo_app_v2"},
				{Name: "foo", Version: "0.0.3", Source: "apps"},
			},
			q: fleet.SoftwareIterQueryOptions{IncludedSources: []string{"apps"}},
		},
		{
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "v0.0.2", Source: "apps", GenerateCPE: "cpe:foo_app_v2"},
				{Name: "foo", Version: "0.0.3", Source: "apps"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "bar", Version: "0.0.3", Source: "deb_packages", GenerateCPE: "cpe:bar_v3"},
			},
			q: fleet.SoftwareIterQueryOptions{},
		},
	}

	for _, tC := range testCases {
		var actual []fleet.Software

		iter, err := ds.AllSoftwareIterator(context.Background(), tC.q)
		require.NoError(t, err)
		for iter.Next() {
			software, err := iter.Value()
			require.NoError(t, err)
			actual = append(actual, *software)
		}
		iter.Close()
		test.ElementsMatchSkipID(t, tC.expected, actual)
	}
}

func testUpsertSoftwareCPEs(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	cpes := []fleet.SoftwareCPE{
		{SoftwareID: host.Software[0].ID, CPE: "cpe:foo_ce_v1"},
		{SoftwareID: host.Software[0].ID, CPE: "cpe:foo_ce_v2"},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	cpes, err = ds.ListSoftwareCPEs(ctx)
	require.NoError(t, err)
	require.Equal(t, len(cpes), 1)
	require.Equal(t, cpes[0].CPE, "cpe:foo_ce_v2")

	cpes = []fleet.SoftwareCPE{
		{SoftwareID: host.Software[0].ID, CPE: "cpe:foo_ce_v3"},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	cpes = []fleet.SoftwareCPE{
		{SoftwareID: host.Software[0].ID, CPE: "cpe:foo_ce_v4"},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	cpes, err = ds.ListSoftwareCPEs(ctx)
	require.NoError(t, err)
	require.Equal(t, len(cpes), 1)
	require.Equal(t, cpes[0].CPE, "cpe:foo_ce_v4")
}

func testDeleteOutOfDateVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	vulns := []fleet.SoftwareVulnerability{
		{
			SoftwareID: host.Software[0].ID,
			CVE:        "CVE-2023-001",
		},
		{
			SoftwareID: host.Software[0].ID,
			CVE:        "CVE-2023-002",
		},
	}

	inserted, err := ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	inserted, err = ds.InsertSoftwareVulnerability(ctx, vulns[1], fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	_, err = ds.writer(ctx).ExecContext(ctx, "UPDATE software_cve SET updated_at = '2020-10-10 12:00:00'")
	require.NoError(t, err)

	// This should update the 'updated_at' timestamp.
	inserted, err = ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.NVDSource)
	require.NoError(t, err)
	require.False(t, inserted)

	err = ds.DeleteOutOfDateVulnerabilities(ctx, fleet.NVDSource, 2*time.Hour)
	require.NoError(t, err)

	storedSoftware, err := ds.SoftwareByID(ctx, host.Software[0].ID, nil, false, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(storedSoftware.Vulnerabilities))
	require.Equal(t, "CVE-2023-001", storedSoftware.Vulnerabilities[0].CVE)
}

func testDeleteSoftwareCPEs(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	cpes := []fleet.SoftwareCPE{
		{
			SoftwareID: host.Software[0].ID,
			CPE:        "CPE-001",
		},
		{
			SoftwareID: host.Software[1].ID,
			CPE:        "CPE-002",
		},
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	t.Run("nothing to delete", func(t *testing.T) {
		affected, err := ds.DeleteSoftwareCPEs(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, affected)
	})

	t.Run("with invalid software id", func(t *testing.T) {
		toDelete := []fleet.SoftwareCPE{cpes[0], {
			SoftwareID: host.Software[1].ID + 1234,
			CPE:        "CPE-002",
		}}

		affected, err := ds.DeleteSoftwareCPEs(ctx, toDelete)
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)

		storedCPEs, err := ds.ListSoftwareCPEs(ctx)
		require.NoError(t, err)
		test.ElementsMatchSkipID(t, cpes[1:], storedCPEs)

		storedSoftware, err := ds.SoftwareByID(ctx, cpes[0].SoftwareID, nil, false, nil)
		require.NoError(t, err)
		require.Empty(t, storedSoftware.GenerateCPE)
	})
}

func testGetHostSoftwareInstalledPaths(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	// No installed_path entries
	actual, err := ds.getHostSoftwareInstalledPaths(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, actual)

	// Insert an installed_path for a single software entry
	query := `INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`
	args := []interface{}{host.ID, host.Software[0].ID, "/some/path"}
	_, err = ds.writer(ctx).ExecContext(ctx, query, args...)
	require.NoError(t, err)

	actual, err = ds.getHostSoftwareInstalledPaths(ctx, host.ID)
	require.Len(t, actual, 1)
	require.Equal(t, actual[0].SoftwareID, host.Software[0].ID)
	require.Equal(t, actual[0].HostID, host.ID)
	require.Equal(t, actual[0].InstalledPath, "/some/path")
	require.NoError(t, err)
}

func testHostSoftwareInstalledPathsDelta(t *testing.T, ds *Datastore) {
	host := fleet.Host{ID: 1}

	software := []fleet.Software{
		{
			ID:      2,
			Name:    "foo",
			Version: "0.0.1",
			Source:  "chrome_extensions",
		},
		{
			ID:      3,
			Name:    "bar",
			Version: "0.0.2",
			Source:  "chrome_extensions",
		},
		{
			ID:      4,
			Name:    "zub",
			Version: "0.0.3",
			Source:  "chrome_extensions",
		},
		{
			ID:      5,
			Name:    "zib",
			Version: "0.0.4",
			Source:  "chrome_extensions",
		},
	}

	t.Run("empty args", func(t *testing.T) {
		toI, toD, err := hostSoftwareInstalledPathsDelta(host.ID, nil, nil, nil)
		require.Empty(t, toI)
		require.Empty(t, toD)
		require.NoError(t, err)
	})

	t.Run("nothing reported from osquery", func(t *testing.T) {
		var stored []fleet.HostSoftwareInstalledPath
		for i, s := range software {
			stored = append(stored, fleet.HostSoftwareInstalledPath{
				ID:            uint(i),
				HostID:        host.ID,
				SoftwareID:    s.ID,
				InstalledPath: fmt.Sprintf("/some/path/%d", s.ID),
			})
		}

		toI, toD, err := hostSoftwareInstalledPathsDelta(host.ID, nil, stored, software)
		require.NoError(t, err)

		require.Empty(t, toI)

		// Kind of an edge case ... but if nothing is reported by osquery we want the state of the
		// DB to reflect that.
		require.Len(t, toD, len(stored))
		var expected []uint
		for _, s := range stored {
			expected = append(expected, s.ID)
		}
		require.ElementsMatch(t, toD, expected)
	})

	t.Run("host has no software but some paths were reported", func(t *testing.T) {
		reported := make(map[string]struct{})
		reported[fmt.Sprintf("/some/path/%d%s%s", software[0].ID, fleet.SoftwareFieldSeparator, software[0].ToUniqueStr())] = struct{}{}
		reported[fmt.Sprintf("/some/path/%d%s%s", software[1].ID+1, fleet.SoftwareFieldSeparator, software[1].ToUniqueStr())] = struct{}{}
		reported[fmt.Sprintf("/some/path/%d%s%s", software[2].ID, fleet.SoftwareFieldSeparator, software[2].ToUniqueStr())] = struct{}{}

		var stored []fleet.HostSoftwareInstalledPath
		_, _, err := hostSoftwareInstalledPathsDelta(host.ID, reported, stored, nil)
		require.Error(t, err)
	})

	t.Run("we have some deltas", func(t *testing.T) {
		getKey := func(s fleet.Software, change uint) string {
			return fmt.Sprintf("/some/path/%d%s%s", s.ID+change, fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		}
		reported := make(map[string]struct{})
		reported[getKey(software[0], 0)] = struct{}{}
		reported[getKey(software[1], 1)] = struct{}{}
		reported[getKey(software[2], 0)] = struct{}{}

		var stored []fleet.HostSoftwareInstalledPath
		stored = append(stored, fleet.HostSoftwareInstalledPath{
			ID:            1,
			HostID:        host.ID,
			SoftwareID:    software[0].ID,
			InstalledPath: fmt.Sprintf("/some/path/%d", software[0].ID),
		})
		stored = append(stored, fleet.HostSoftwareInstalledPath{
			ID:            2,
			HostID:        host.ID,
			SoftwareID:    software[1].ID,
			InstalledPath: fmt.Sprintf("/some/path/%d", software[1].ID),
		})
		stored = append(stored, fleet.HostSoftwareInstalledPath{
			ID:            3,
			HostID:        host.ID,
			SoftwareID:    software[2].ID,
			InstalledPath: fmt.Sprintf("/some/path/%d", software[2].ID+1),
		})
		stored = append(stored, fleet.HostSoftwareInstalledPath{
			ID:            4,
			HostID:        host.ID,
			SoftwareID:    software[3].ID,
			InstalledPath: fmt.Sprintf("/some/path/%d", software[3].ID),
		})

		toI, toD, err := hostSoftwareInstalledPathsDelta(host.ID, reported, stored, software)
		require.NoError(t, err)

		require.Len(t, toD, 3)
		require.ElementsMatch(t,
			[]uint{toD[0], toD[1], toD[2]},
			[]uint{stored[1].ID, stored[2].ID, stored[3].ID},
		)

		require.Len(t, toI, 2)
		for i := range toI {
			require.Equal(t, toI[i].HostID, host.ID)
		}

		require.ElementsMatch(t,
			[]uint{toI[0].SoftwareID, toI[1].SoftwareID},
			[]uint{software[1].ID, software[2].ID},
		)
		require.ElementsMatch(t,
			[]string{toI[0].InstalledPath, toI[1].InstalledPath},
			[]string{fmt.Sprintf("/some/path/%d", software[1].ID+1), fmt.Sprintf("/some/path/%d", software[2].ID)},
		)
	})
}

func testDeleteHostSoftwareInstalledPaths(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := fleet.Host{ID: 1}
	host2 := fleet.Host{ID: 2}

	software1 := []fleet.Software{
		{ID: 1, Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{ID: 2, Name: "bar", Version: "0.0.1", Source: "chrome_extensions"},
		{ID: 3, Name: "zoo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{ID: 4, Name: "zip", Version: "0.0.1", Source: "apps"},
		{ID: 5, Name: "bur", Version: "0.0.1", Source: "apps"},
	}

	query := `INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`
	for _, s := range software1 {
		args := []interface{}{host1.ID, s.ID, fmt.Sprintf("/some/path/%d", s.ID)}
		_, err := ds.writer(ctx).ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}

	args := []interface{}{host2.ID, software2[0].ID, fmt.Sprintf("/some/path/%d", software2[0].ID)}
	_, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	require.NoError(t, err)

	storedOnHost1, err := ds.getHostSoftwareInstalledPaths(ctx, host1.ID)
	require.NoError(t, err)

	storedOnHost2, err := ds.getHostSoftwareInstalledPaths(ctx, host2.ID)
	require.NoError(t, err)

	var toDelete []uint
	for _, r := range storedOnHost1 {
		if r.SoftwareID == software1[0].ID || r.SoftwareID == software1[1].ID {
			toDelete = append(toDelete, r.ID)
		}
	}

	for _, r := range storedOnHost2 {
		if r.SoftwareID == software2[0].ID {
			toDelete = append(toDelete, r.ID)
		}
	}

	require.NoError(t, deleteHostSoftwareInstalledPaths(ctx, ds.writer(ctx), toDelete))

	var actual []fleet.HostSoftwareInstalledPath
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &actual, `SELECT host_id, software_id, installed_path FROM host_software_installed_paths`))

	expected := []fleet.HostSoftwareInstalledPath{
		{
			HostID:        host1.ID,
			SoftwareID:    software1[2].ID,
			InstalledPath: fmt.Sprintf("/some/path/%d", software1[2].ID),
		},
	}

	test.ElementsMatchSkipID(t, actual, expected)
}

func testInsertHostSoftwareInstalledPaths(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	toInsert := []fleet.HostSoftwareInstalledPath{
		{
			HostID:        1,
			SoftwareID:    1,
			InstalledPath: "1",
		},
		{
			HostID:        1,
			SoftwareID:    2,
			InstalledPath: "2",
		},
		{
			HostID:        1,
			SoftwareID:    3,
			InstalledPath: "3",
		},
	}
	require.NoError(t, insertHostSoftwareInstalledPaths(ctx, ds.writer(ctx), toInsert))

	var actual []fleet.HostSoftwareInstalledPath
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &actual, `SELECT host_id, software_id, installed_path FROM host_software_installed_paths`))

	require.ElementsMatch(t, actual, toInsert)
}

func TestReconcileSoftwareTitles(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

	expectedSoftware := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		{Name: "baz", Version: "0.0.1", Source: "deb_packages"},
	}

	software1 := []fleet.Software{expectedSoftware[0], expectedSoftware[2]}
	software2 := []fleet.Software{expectedSoftware[1], expectedSoftware[2], expectedSoftware[3]}
	software3 := []fleet.Software{expectedSoftware[4]}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, software3)
	require.NoError(t, err)

	getSoftware := func() ([]fleet.Software, error) {
		var sw []fleet.Software
		err := ds.writer(ctx).SelectContext(ctx, &sw, `SELECT
			id, name, version, bundle_identifier, source, extension_id, browser, `+"`release`"+`, vendor, arch, title_id
		FROM software ORDER BY name, source, browser, version`)
		if err != nil {
			return nil, err
		}
		return sw, nil
	}

	getTitles := func() ([]fleet.SoftwareTitle, error) {
		var swt []fleet.SoftwareTitle
		err := ds.writer(ctx).SelectContext(ctx, &swt, `SELECT id, name, source, browser FROM software_titles ORDER BY name, source, browser`)
		if err != nil {
			return nil, err
		}
		return swt, nil
	}

	expectedTitlesByNSB := map[string]fleet.SoftwareTitle{}
	assertSoftware := func(t *testing.T, wantSoftware []fleet.Software, wantNilTitleID []fleet.Software) {
		gotSoftware, err := getSoftware()
		require.NoError(t, err)
		require.Len(t, gotSoftware, len(wantSoftware))

		byNSBV := map[string]fleet.Software{}
		for _, s := range wantSoftware {
			byNSBV[s.Name+s.Source+s.Browser+s.Version] = s
		}

		for _, r := range gotSoftware {
			_, ok := byNSBV[r.Name+r.Source+r.Browser+r.Version]
			require.True(t, ok)

			if r.TitleID == nil {
				var found bool
				for _, s := range wantNilTitleID {
					if s.Name == r.Name && s.Source == r.Source && s.Browser == r.Browser && s.Version == r.Version {
						found = true
						break
					}
				}
				require.True(t, found)
			} else {
				require.NotNil(t, r.TitleID)
				swt, ok := expectedTitlesByNSB[r.Name+r.Source+r.Browser]
				require.True(t, ok)
				require.NotNil(t, r.TitleID)
				require.Equal(t, swt.ID, *r.TitleID)
				require.Equal(t, swt.Name, r.Name)
				require.Equal(t, swt.Source, r.Source)
				require.Equal(t, swt.Browser, r.Browser)
			}
		}
	}

	assertTitles := func(t *testing.T, gotTitles []fleet.SoftwareTitle, expectMissing []string) {
		for _, r := range gotTitles {
			if len(expectMissing) > 0 {
				require.NotContains(t, expectMissing, r.Name)
			}
			e, ok := expectedTitlesByNSB[r.Name+r.Source+r.Browser]
			require.True(t, ok)
			require.Equal(t, e.ID, r.ID)
			require.Equal(t, e.Name, r.Name)
			require.Equal(t, e.Source, r.Source)
			require.Equal(t, e.Browser, r.Browser)
		}
	}

	// title_id is initially nil for all software entries
	assertSoftware(t, expectedSoftware, expectedSoftware)

	// reconcile software titles
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	swt, err := getTitles()
	require.NoError(t, err)
	require.Len(t, swt, 4)

	require.Equal(t, swt[0].Name, "bar")
	require.Equal(t, swt[0].Source, "deb_packages")
	require.Equal(t, swt[0].Browser, "")
	expectedTitlesByNSB[swt[0].Name+swt[0].Source+swt[0].Browser] = swt[0]

	require.Equal(t, swt[1].Name, "baz")
	require.Equal(t, swt[1].Source, "deb_packages")
	require.Equal(t, swt[1].Browser, "")
	expectedTitlesByNSB[swt[1].Name+swt[1].Source+swt[1].Browser] = swt[1]

	require.Equal(t, swt[2].Name, "foo")
	require.Equal(t, swt[2].Source, "chrome_extensions")
	require.Equal(t, swt[2].Browser, "")
	expectedTitlesByNSB[swt[2].Name+swt[2].Source+swt[2].Browser] = swt[2]

	require.Equal(t, swt[3].Name, "foo")
	require.Equal(t, swt[3].Source, "chrome_extensions")
	require.Equal(t, swt[3].Browser, "chrome")
	expectedTitlesByNSB[swt[3].Name+swt[3].Source+swt[3].Browser] = swt[3]

	// title_id is now populated for all software entries
	assertSoftware(t, expectedSoftware, nil)

	// remove the bar software title from host 2
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2[:2])
	require.NoError(t, err)
	assertSoftware(t, []fleet.Software{expectedSoftware[0], expectedSoftware[1], expectedSoftware[2], expectedSoftware[4]}, nil)

	// bar is no longer associated with any host so the title should be deleted
	require.NoError(t, ds.ReconcileSoftwareTitles(context.Background()))
	gotTitles, err := getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 3)
	assertTitles(t, gotTitles, []string{"bar"})

	// add bar to host 3
	_, err = ds.UpdateHostSoftware(context.Background(), host3.ID, []fleet.Software{expectedSoftware[3], expectedSoftware[4]})
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))

	// title_id is initially nil for new software entries
	assertSoftware(t, expectedSoftware, []fleet.Software{expectedSoftware[3]})

	// bar isn't added back to software titles until we reconcile software titles
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 3)
	assertTitles(t, gotTitles, []string{"bar"})

	// reconcile software titles
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 4)

	// bar was added back to software titles with a new ID
	require.Equal(t, "bar", gotTitles[0].Name)
	require.Equal(t, "deb_packages", gotTitles[0].Source)
	require.NotEqual(t, expectedTitlesByNSB[gotTitles[0].Name+gotTitles[0].Source], gotTitles[0].ID)
	expectedTitlesByNSB[gotTitles[0].Name+gotTitles[0].Source] = gotTitles[0]
	assertTitles(t, gotTitles, nil)

	// title_id is now populated for bar
	assertSoftware(t, expectedSoftware, nil)

	// add a new version of foo to host 3
	expectedSoftware = append(expectedSoftware, fleet.Software{Name: "foo", Version: "0.0.4", Source: "chrome_extensions"})
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, expectedSoftware[3:])
	require.NoError(t, err)

	// title_id is initially nil for new software entries
	assertSoftware(t, expectedSoftware, []fleet.Software{expectedSoftware[5]})

	// new version of foo doesn't result in a new software title entry
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 4)
	assertTitles(t, gotTitles, nil)

	// title_id is now populated for new version of foo
	assertSoftware(t, expectedSoftware, nil)

	// add a new source of foo to host 3
	expectedSoftware = append(expectedSoftware, fleet.Software{Name: "foo", Version: "0.0.4", Source: "rpm_packages"})
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, expectedSoftware[3:])
	require.NoError(t, err)

	// title_id is initially nil for new software entries
	assertSoftware(t, expectedSoftware, []fleet.Software{expectedSoftware[6]})

	// new source of foo results in a new software title entry
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 5)
	require.Equal(t, "foo", gotTitles[4].Name)
	require.Equal(t, "rpm_packages", gotTitles[4].Source)
	require.Equal(t, "", gotTitles[4].Browser)
	expectedTitlesByNSB[gotTitles[4].Name+gotTitles[4].Source+gotTitles[4].Browser] = gotTitles[4]
	assertTitles(t, gotTitles, nil)

	// title_id is now populated for new source of foo
	assertSoftware(t, expectedSoftware, nil)
}

func testUpdateHostSoftwareDeadlock(t *testing.T, ds *Datastore) {
	// To increase chance of deadlock increase these numbers.
	// We are keeping them low to not cause CI issues ("too many connections" errors
	// due to concurrent tests).
	const (
		hostCount   = 10
		updateCount = 10
	)
	ctx := context.Background()
	var hosts []*fleet.Host
	for i := 1; i <= hostCount; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			ID:              uint(i),
			OsqueryHostID:   ptr.String(fmt.Sprintf("id-%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("key-%d", i)),
			Platform:        "linux",
			Hostname:        fmt.Sprintf("host-%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}
	var g errgroup.Group
	for _, h := range hosts {
		hostID := h.ID
		g.Go(func() error {
			for i := 0; i < updateCount; i++ {
				software := []fleet.Software{
					{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo"},
					{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar"},
					{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz"},
				}
				removeIdx := rand.Intn(len(software))
				software = append(software[:removeIdx], software[removeIdx+1:]...)
				if _, err := ds.UpdateHostSoftware(ctx, hostID, software); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func testVerifySoftwareChecksum(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	computeChecksum := func(sw fleet.Software) string {
		h := md5.New()
		// compute the same way as the DB, see the softwareChecksumComputedColumn function
		cols := []string{sw.Name, sw.Version, sw.Source, sw.BundleIdentifier, sw.Release, sw.Arch, sw.Vendor, sw.Browser, sw.ExtensionID}
		fmt.Fprint(h, strings.Join(cols, "\x00"))
		checksum := h.Sum(nil)
		return hex.EncodeToString(checksum)
	}

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test"},
		{Name: "foo", Version: "0.0.1", Source: "test", Browser: "firefox"},
		{Name: "foo", Version: "0.0.1", Source: "test", ExtensionID: "ext"},
		{Name: "foo", Version: "0.0.2", Source: "test"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	checksums := make([]string, len(software))
	for i, sw := range software {
		checksums[i] = computeChecksum(sw)
	}
	for i, cs := range checksums {
		var got fleet.Software
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &got,
				`SELECT name, version, source, bundle_identifier, `+"`release`"+`, arch, vendor, browser, extension_id FROM software WHERE checksum = UNHEX(?)`, cs)
		})
		require.Equal(t, software[i], got)
	}
}
