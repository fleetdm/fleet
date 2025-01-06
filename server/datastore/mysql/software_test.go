package mysql

import (
	"context"
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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/google/uuid"
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
		{"AllSoftwareIteratorForCustomLinuxImages", testSoftwareIteratorForLinuxKernelCustomImages},
		{"UpsertSoftwareCPEs", testUpsertSoftwareCPEs},
		{"DeleteOutOfDateVulnerabilities", testDeleteOutOfDateVulnerabilities},
		{"DeleteSoftwareCPEs", testDeleteSoftwareCPEs},
		{"SoftwareByIDNoDuplicatedVulns", testSoftwareByIDNoDuplicatedVulns},
		{"SoftwareByIDIncludesCVEPublishedDate", testSoftwareByIDIncludesCVEPublishedDate},
		{"GetHostSoftwareInstalledPaths", testGetHostSoftwareInstalledPaths},
		{"HostSoftwareInstalledPathsDelta", testHostSoftwareInstalledPathsDelta},
		{"DeleteHostSoftwareInstalledPaths", testDeleteHostSoftwareInstalledPaths},
		{"InsertHostSoftwareInstalledPaths", testInsertHostSoftwareInstalledPaths},
		{"VerifySoftwareChecksum", testVerifySoftwareChecksum},
		{"ListHostSoftware", testListHostSoftware},
		{"ListIOSHostSoftware", testListIOSHostSoftware},
		{"SetHostSoftwareInstallResult", testSetHostSoftwareInstallResult},
		{"ListHostSoftwareInstallThenTransferTeam", testListHostSoftwareInstallThenTransferTeam},
		{"ListHostSoftwareInstallThenDeleteInstallers", testListHostSoftwareInstallThenDeleteInstallers},
		{"ListSoftwareVersionsVulnerabilityFilters", testListSoftwareVersionsVulnerabilityFilters},
		{"TestListHostSoftwareWithLabelScoping", testListHostSoftwareWithLabelScoping},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Helper()
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

	incomingByChecksum, existingSoftware, existingTitlesForNewSoftware, err := ds.getExistingSoftware(
		context.Background(), make(map[string]fleet.Software), incoming,
	)
	require.NoError(t, err)
	tx, err := ds.writer(context.Background()).Beginx()
	require.NoError(t, err)
	_, err = ds.insertNewInstalledHostSoftwareDB(
		context.Background(), tx, host1.ID, incomingByChecksum, existingSoftware, existingTitlesForNewSoftware,
	)
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

	incomingByChecksum, existingSoftware, existingTitlesForNewSoftware, err = ds.getExistingSoftware(
		context.Background(), make(map[string]fleet.Software), incoming,
	)
	require.NoError(t, err)
	tx, err = ds.writer(context.Background()).Beginx()
	require.NoError(t, err)
	_, err = ds.insertNewInstalledHostSoftwareDB(
		context.Background(), tx, host1.ID, incomingByChecksum, existingSoftware, existingTitlesForNewSoftware,
	)
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
	assert.GreaterOrEqual(t, softByID.Vulnerabilities[0].CreatedAt, time.Now().Add(-time.Minute))

	assert.Equal(t, "somecpe", host.Software[0].GenerateCPE)
	require.Len(t, host.Software[0].Vulnerabilities, 2)

	sort.Slice(host.Software[0].Vulnerabilities, func(i, j int) bool {
		return host.Software[0].Vulnerabilities[i].CVE < host.Software[0].Vulnerabilities[j].CVE
	})

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
		{
			"identical with duplicates incoming",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}, {Name: "A", Version: "1.0", Source: "ASD"}},
			true,
		},
		{
			"identical with duplicates incoming and insignificantly changed last open",
			[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now().Add(-time.Second))}},
			[]fleet.Software{
				{Name: "A", Version: "1.0", Source: "ASD"},
				{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now().Add(-time.Hour))},
				{Name: "A", Version: "1.0", Source: "ASD", LastOpenedAt: ptr.Time(time.Now())},
			},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			current, incoming, got := nothingChanged(c.current, c.incoming, defaultMinLastOpenedAtDiff)
			if c.want {
				assert.True(t, got)
				assert.Equal(t, len(current), len(incoming))
			} else {
				assert.False(t, got)
			}
			assert.Equal(t, len(c.current), len(current))
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
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
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

		// Now that we have the software, we can test pagination.
		// Figure out which software has the highest ID.
		targetSoftware := software[0]
		if targetSoftware.ID < software[1].ID {
			targetSoftware = software[1]
		}
		expected = []fleet.Software{foo001}
		if targetSoftware.Name == "foo" && targetSoftware.Version == "0.0.3" {
			expected = []fleet.Software{foo003}
		}

		opts = fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				PerPage:         1,
				Page:            1, // 2nd item, since 1st item is on page 0
				OrderKey:        "id",
				IncludeMetadata: true,
			},
			TeamID:           &team1.ID,
			IncludeCVEScores: true,
		}
		software = listSoftwareCheckCount(t, ds, 1, 2, opts, true)
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by no team (team 0)", func(t *testing.T) {
		opts := fleet.SoftwareListOptions{
			TeamID:           ptr.Uint(0),
			IncludeCVEScores: true,
		}

		software := listSoftwareCheckCount(t, ds, 4, 4, opts, true)
		expected := []fleet.Software{bar003, baz001, foo002, foo003}
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
	t.Helper()
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

	for i, s := range software {
		sort.Slice(s.Vulnerabilities, func(i, j int) bool {
			return s.Vulnerabilities[i].CVE < s.Vulnerabilities[j].CVE
		})
		for i2, v := range s.Vulnerabilities {
			require.Greater(t, v.CreatedAt, time.Now().Add(-time.Hour)) // assert non-zero
			software[i].Vulnerabilities[i2].CreatedAt = time.Time{}     // zero out for comparison
		}
	}

	if returnSorted {
		sort.Slice(software, func(i, j int) bool {
			return software[i].Name+software[i].Version < software[j].Name+software[j].Version
		})
	}
	return software
}

func testSoftwareSyncHostsSoftware(t *testing.T, ds *Datastore) {
	countHostSoftwareBatchSizeOrig := countHostSoftwareBatchSize
	softwareInsertBatchSizeOrig := softwareInsertBatchSize
	t.Cleanup(
		func() {
			countHostSoftwareBatchSize = countHostSoftwareBatchSizeOrig
			softwareInsertBatchSize = softwareInsertBatchSizeOrig
		},
	)
	countHostSoftwareBatchSize = 2
	softwareInsertBatchSize = 2

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

	host0 := test.NewHost(t, ds, "host0", "", "host0key", "host0uuid", time.Now())
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	hostTemp := test.NewHost(t, ds, "hostTemp", "", "hostTempKey", "hostTempUuid", time.Now())

	// Get counts without any software.
	globalOpts := fleet.SoftwareListOptions{
		WithHostCounts: true, ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending},
	}
	_ = listSoftwareCheckCount(t, ds, 0, 0, globalOpts, false)

	software0 := []fleet.Software{
		{Name: "abc", Version: "0.0.1", Source: "apps"},
		{Name: "def", Version: "0.0.1", Source: "apps"},
	}
	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	softwareTemp := make([]fleet.Software, 0, 10)
	for i := 0; i < 10; i++ {
		softwareTemp = append(
			softwareTemp, fleet.Software{Name: fmt.Sprintf("foo%d", i), Version: fmt.Sprintf("%d.0.1", i), Source: "deb_packages"},
		)
	}

	_, err := ds.UpdateHostSoftware(ctx, host0.ID, software0)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, hostTemp.ID, softwareTemp)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	_ = listSoftwareCheckCount(t, ds, 16, 16, globalOpts, false)
	checkTableTotalCount(32)

	// Now, delete 2 hosts. Software with the lowest ID is removed, and there should be a chunk with missing software IDs from the deleted hostTemp software.
	require.NoError(t, ds.DeleteHost(ctx, host0.ID))
	require.NoError(t, ds.DeleteHost(ctx, hostTemp.ID))

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	globalCounts := listSoftwareCheckCount(t, ds, 4, 4, globalOpts, false)
	want := []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
		{Name: "bar", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(8)

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
	checkTableTotalCount(6)

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
	checkTableTotalCount(6)

	team1Opts := fleet.SoftwareListOptions{WithHostCounts: true, TeamID: ptr.Uint(team1.ID), ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	team1Counts := listSoftwareCheckCount(t, ds, 0, 0, team1Opts, false)
	want = []fleet.Software{}
	cmpNameVersionCount(want, team1Counts)
	checkTableTotalCount(6)
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

	// composite pk (software_id, team_id, global_stats), so we expect more rows
	checkTableTotalCount(11)

	soft1ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID, &team1.ID, false, nil)
	require.NoError(t, err)
	soft2ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[1].ID, &team1.ID, false, nil)
	require.NoError(t, err)
	test.ElementsMatchSkipIDAndHostCount(t, software1, []fleet.Software{*soft1ByID, *soft2ByID})

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

	checkTableTotalCount(9)

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
	checkTableTotalCount(8)
}

// softwareChecksumComputedColumn computes the checksum for a software entry
// The calculation must match the one in computeRawChecksum
func softwareChecksumComputedColumn(tableAlias string) string {
	if tableAlias != "" && !strings.HasSuffix(tableAlias, ".") {
		tableAlias += "."
	}

	// concatenate with separator \x00
	return fmt.Sprintf(
		` UNHEX(
		MD5(
			CONCAT_WS(CHAR(0),
				%sname,
				%[1]sversion,
				%[1]ssource,
				COALESCE(%[1]sbundle_identifier, ''),
				`+"%[1]s`release`"+`,
				%[1]sarch,
				%[1]svendor,
				%[1]sbrowser,
				%[1]sextension_id
			)
		)
	) `, tableAlias,
	)
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
		key := fmt.Sprintf("%s%s%s%s%s", fmt.Sprintf("/some/path/%s", s.Name), fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		s1Paths[key] = struct{}{}
	}
	require.NoError(t, ds.UpdateHostSoftwareInstalledPaths(context.Background(), host1.ID, s1Paths, mutationResults))

	mutationResults, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2)
	require.NoError(t, err)

	// Insert paths for software2
	s2Paths := map[string]struct{}{}
	for _, s := range software2 {
		key := fmt.Sprintf("%s%s%s%s%s", fmt.Sprintf("/some/path/%s", s.Name), fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, s.ToUniqueStr())
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

	// "baz2" is still present in the database, even though no hosts are using it, until ds.SyncHostsSoftware is executed.
	soft, err := ds.SoftwareByID(ctx, baz2SoftwareID, nil, false, nil)
	require.NoError(t, err)
	assert.Equal(t, "baz2", soft.Name)
	assert.Zero(t, soft.HostsCount)

	// "new" is not returned until ds.SyncHostsSoftware is executed.
	// "bar" and "baz2" are gone from host_software, but will not be deleted until ds.SyncHostsSoftware is executed.
	// "baz" still has the wrong count because ds.SyncHostsSoftware hasn't run yet.
	//
	// So... counts are "off" until ds.SyncHostsSoftware is run.
	software = listSoftwareCheckCount(t, ds, 4, 4, opts, false)
	expectedSoftware = []fleet.Software{
		{Name: "foo", Version: "0.0.1", HostsCount: 2},
		{Name: "baz", Version: "0.0.3", HostsCount: 2},
		{Name: "bar", Version: "0.0.2", HostsCount: 2},
		{Name: "baz2", Version: "0.0.3", HostsCount: 1},
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
	softwareInsertBatchSizeOrig := softwareInsertBatchSize
	t.Cleanup(
		func() {
			softwareInsertBatchSize = softwareInsertBatchSizeOrig
		},
	)
	softwareInsertBatchSize = 2

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
				return lts.Before(*rts) || (lts.Equal(*rts) && lsw.Name < rsw.Name)
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

			var titleID uint
			require.NoError(
				t, ds.writer(ctx).GetContext(
					ctx, &titleID,
					`SELECT s.title_id FROM software s INNER JOIN software_titles st ON (s.name = st.name AND s.source = st.source AND s.browser = st.browser) WHERE st.id = ?`,
					sw.ID,
				),
			)
			assert.NotZero(t, titleID)

			if want.ts.IsZero() {
				require.Nil(t, sw.LastOpenedAt)
			} else {
				require.WithinDuration(t, want.ts, *sw.LastOpenedAt, time.Second)
			}
		}
	}

	// set the initial software list
	sw := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "test", GenerateCPE: "cpe_foo", Browser: "chrome"},
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

	// more changes: all software receives a date further in the future, so all should be updated
	farFuture := now.Add(4 * 24 * time.Hour)
	sw = []fleet.Software{
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar", LastOpenedAt: &farFuture},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz", LastOpenedAt: &farFuture},
		{Name: "qux", Version: "0.0.4", Source: "test", GenerateCPE: "cpe_qux", LastOpenedAt: &farFuture},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, sw)
	require.NoError(t, err)
	validateSoftware(tup{"bar", farFuture}, tup{"baz", farFuture}, tup{"qux", farFuture})
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

		// Sleep so that the updated_at timestamp is guaranteed to be updated.
		time.Sleep(1 * time.Second)
		insertedOrUpdated, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: host.Software[0].ID, CVE: "cve-1",
		}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		// This will always return true because we always update the timestamp
		assert.True(t, insertedOrUpdated)

		storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)

		occurrence := make(map[string]int)
		for _, v := range storedVulns[host.ID] {
			occurrence[v.CVE]++
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

		// Sleep so that the updated_at timestamp is guaranteed to be updated.
		time.Sleep(1 * time.Second)
		insertedOrUpdated, err := ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.UbuntuOVALSource)
		require.NoError(t, err)
		// This will always return true because we always update the timestamp
		assert.True(t, insertedOrUpdated)

		storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)

		occurrence := make(map[string]int)
		for _, v := range storedVulns[host.ID] {
			occurrence[v.CVE]++
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

		filter := fleet.VulnSoftwareFilter{HostID: &host.ID}
		result, err := ds.ListSoftwareForVulnDetection(ctx, filter)
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

		// test name filter
		filter = fleet.VulnSoftwareFilter{Name: "fo"} // LIKE match
		result, err = ds.ListSoftwareForVulnDetection(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "foo", result[0].Name)

		// test source filter
		filter = fleet.VulnSoftwareFilter{Source: "deb_packages"}
		result, err = ds.ListSoftwareForVulnDetection(ctx, filter)
		sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "baz", result[0].Name)
		require.Equal(t, "biz", result[1].Name)
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
		{Name: "foobar", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "foo", Version: "v0.0.2", Source: "apps"},
		{Name: "foo", Version: "0.0.3", Source: "apps"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
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
		name     string
		q        fleet.SoftwareIterQueryOptions
		expected []fleet.Software
	}{
		{
			name: "include apps source",
			expected: []fleet.Software{
				{Name: "foo", Version: "v0.0.2", Source: "apps", GenerateCPE: "cpe:foo_app_v2"},
				{Name: "foo", Version: "0.0.3", Source: "apps"},
			},
			q: fleet.SoftwareIterQueryOptions{IncludedSources: []string{"apps"}},
		},
		{
			name: "exclude apps source",
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "bar", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "foobar", Version: "0.0.1", Source: "chrome_extensions"},
				{Name: "bar", Version: "0.0.3", Source: "deb_packages", GenerateCPE: "cpe:bar_v3"},
				{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
			},
			q: fleet.SoftwareIterQueryOptions{ExcludedSources: []string{"apps"}},
		},
		{
			name: "no filter",
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "v0.0.2", Source: "apps", GenerateCPE: "cpe:foo_app_v2"},
				{Name: "foo", Version: "0.0.3", Source: "apps"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "bar", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "foobar", Version: "0.0.1", Source: "chrome_extensions"},
				{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
				{Name: "bar", Version: "0.0.3", Source: "deb_packages", GenerateCPE: "cpe:bar_v3"},
			},
			q: fleet.SoftwareIterQueryOptions{},
		},
		{
			name: "partial name filter includes deb_packages",
			expected: []fleet.Software{
				{Name: "bar", Version: "0.0.3", Source: "deb_packages", GenerateCPE: "cpe:bar_v3"},
			},
			q: fleet.SoftwareIterQueryOptions{NameMatch: `ba[r|f]`, IncludedSources: []string{"deb_packages"}},
		},
		{
			name: "name filter includes chrome_extensions",
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				{Name: "foobar", Version: "0.0.1", Source: "chrome_extensions"},
			},
			q: fleet.SoftwareIterQueryOptions{NameMatch: "foo\\.*", IncludedSources: []string{"chrome_extensions"}},
		},
		{
			name: "name filter and not name filter",
			expected: []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "cpe:foo_ce_v1"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			},
			q: fleet.SoftwareIterQueryOptions{NameMatch: "foo\\.*", NameExclude: "bar$", IncludedSources: []string{"chrome_extensions"}},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
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
		})
	}
}

func testSoftwareIteratorForLinuxKernelCustomImages(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "linux-image-5.4.0-42-generic", Version: "5.4.0-42.46", Source: "deb_packages"},
		{Name: "linux-image-6.5.0-42-generic", Version: "6.5.0-100.27", Source: "deb_packages"},
		{Name: "linux-image-5.4.0-42-custom", Version: "5.4.0-42.46", Source: "deb_packages"},
		{Name: "linux-image-6.5.0-42-1234-foo", Version: "6.5.0-100.27", Source: "deb_packages"},
		{Name: "linux-image-generic", Version: "1.0.0", Source: "deb_packages"},
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	expected := []fleet.Software{
		{Name: "linux-image-5.4.0-42-custom", Version: "5.4.0-42.46", Source: "deb_packages"},
		{Name: "linux-image-6.5.0-42-1234-foo", Version: "6.5.0-100.27", Source: "deb_packages"},
	}

	opts := fleet.SoftwareIterQueryOptions{
		NameMatch:       nvd.LinuxImageRegex,
		NameExclude:     nvd.BuildLinuxExclusionRegex(),
		IncludedSources: []string{"deb_packages"},
	}

	iterator, err := ds.AllSoftwareIterator(context.Background(), opts)
	require.NoError(t, err)

	var actual []fleet.Software
	for iterator.Next() {
		software, err := iterator.Value()
		require.NoError(t, err)
		actual = append(actual, *software)
	}
	iterator.Close()
	test.ElementsMatchSkipID(t, expected, actual)
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
	insertedOrUpdated, err := ds.InsertSoftwareVulnerability(ctx, vulns[0], fleet.NVDSource)
	require.NoError(t, err)
	assert.True(t, insertedOrUpdated)

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
		reported[fmt.Sprintf("/some/path/%d%s%s%s%s", software[0].ID, fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, software[0].ToUniqueStr())] = struct{}{}
		reported[fmt.Sprintf("/some/path/%d%s%s%s%s", software[1].ID+1, fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, software[1].ToUniqueStr())] = struct{}{}
		reported[fmt.Sprintf("/some/path/%d%s%s%s%s", software[2].ID, fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, software[2].ToUniqueStr())] = struct{}{}

		var stored []fleet.HostSoftwareInstalledPath
		_, _, err := hostSoftwareInstalledPathsDelta(host.ID, reported, stored, nil)
		require.Error(t, err)
	})

	t.Run("we have some deltas", func(t *testing.T) {
		getKey := func(s fleet.Software, change uint) string {
			return fmt.Sprintf("/some/path/%d%s%s%s%s", s.ID+change, fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, s.ToUniqueStr())
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
	expectedTitlesByNSB := map[string]fleet.SoftwareTitle{}
	for _, s := range expectedSoftware {
		expectedTitlesByNSB[s.Name+s.Source+s.Browser] = fleet.SoftwareTitle{
			Name:    s.Name,
			Source:  s.Source,
			Browser: s.Browser,
		}
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

	assertSoftware := func(t *testing.T, wantSoftware []fleet.Software) {
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

			assert.NotNil(t, r.TitleID)
			swt, ok := expectedTitlesByNSB[r.Name+r.Source+r.Browser]
			require.True(t, ok)
			assert.Equal(t, swt.ID, *r.TitleID)
			assert.Equal(t, swt.Name, r.Name)
			assert.Equal(t, swt.Source, r.Source)
			assert.Equal(t, swt.Browser, r.Browser)
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

	swTitles, err := getTitles()
	require.NoError(t, err)
	for _, swt := range swTitles {
		if _, ok := expectedTitlesByNSB[swt.Name+swt.Source+swt.Browser]; ok {
			expectedTitlesByNSB[swt.Name+swt.Source+swt.Browser] = swt
		}
	}

	assertSoftware(t, expectedSoftware)

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

	// Double check software and titles
	assertSoftware(t, expectedSoftware)

	// remove the bar software title from host 2
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software2[:2])
	require.NoError(t, err)
	// SyncHostsSoftware will remove the above software item from the software table
	require.NoError(t, ds.SyncHostsSoftware(context.Background(), time.Now()))
	assertSoftware(t, []fleet.Software{expectedSoftware[0], expectedSoftware[1], expectedSoftware[2], expectedSoftware[4]})

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
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 4)

	// bar was added back to software titles with a new ID
	require.Equal(t, "bar", gotTitles[0].Name)
	require.Equal(t, "deb_packages", gotTitles[0].Source)
	require.NotEqual(t, expectedTitlesByNSB[gotTitles[0].Name+gotTitles[0].Source], gotTitles[0].ID)
	expectedTitlesByNSB[gotTitles[0].Name+gotTitles[0].Source] = gotTitles[0]
	assertTitles(t, gotTitles, nil)
	assertSoftware(t, expectedSoftware)

	// add a new version of foo to host 3
	expectedSoftware = append(expectedSoftware, fleet.Software{Name: "foo", Version: "0.0.4", Source: "chrome_extensions"})
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, expectedSoftware[3:])
	require.NoError(t, err)
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 4)
	assertTitles(t, gotTitles, nil)
	assertSoftware(t, expectedSoftware)

	// add a new source of foo to host 3
	expectedSoftware = append(expectedSoftware, fleet.Software{Name: "foo", Version: "0.0.4", Source: "rpm_packages"})
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, expectedSoftware[3:])
	require.NoError(t, err)

	// new source of foo results in a new software title entry
	gotTitles, err = getTitles()
	require.NoError(t, err)
	require.Len(t, gotTitles, 5)
	require.Equal(t, "foo", gotTitles[4].Name)
	require.Equal(t, "rpm_packages", gotTitles[4].Source)
	require.Equal(t, "", gotTitles[4].Browser)
	expectedTitlesByNSB[gotTitles[4].Name+gotTitles[4].Source+gotTitles[4].Browser] = gotTitles[4]
	assertTitles(t, gotTitles, nil)
	assertSoftware(t, expectedSoftware)

	// Test duplicate key handling in `ReconcileSoftwareTitles`.
	// Since the existing software_titles and software entries have different `source` values,
	// the code will attempt to insert into `software_titles`, but the bundle_identifier + additional_identifier
	// key (com.example.app1-0) will conflict.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, `INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES (7, 'App1', 'some_source', 'Chrome', 'com.example.app1')`)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, `INSERT INTO software (name, source, browser, bundle_identifier) VALUES ('App1', 'some_other_source', 'Chrome', 'com.example.app1')`)
		require.NoError(t, err)
		require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
		return nil
	})
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
		checksum, err := computeRawChecksum(sw)
		require.NoError(t, err)
		checksums[i] = hex.EncodeToString(checksum)
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

func testListHostSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithPlatform("darwin"))
	nanoEnroll(t, ds, host, false)
	otherHost := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now(), test.WithPlatform("linux"))
	opts := fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{PerPage: 11, IncludeMetadata: true, OrderKey: "name", TestSecondaryOrderKey: "source"}}

	user, err := ds.NewUser(ctx, &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	expectStatus := func(s fleet.SoftwareInstallerStatus) *fleet.SoftwareInstallerStatus {
		return &s
	}

	// no software yet
	sw, meta, err := ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	require.Equal(t, &fleet.PaginationMetadata{}, meta)

	// works with available software too
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	require.Equal(t, &fleet.PaginationMetadata{}, meta)

	// available for install only works too
	opts.OnlyAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Empty(t, sw)
	assert.Equal(t, &fleet.PaginationMetadata{}, meta)

	// self-service only works too
	opts.OnlyAvailableForInstall = false
	opts.SelfServiceOnly = true
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	require.Equal(t, &fleet.PaginationMetadata{}, meta)

	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	require.Equal(t, &fleet.PaginationMetadata{}, meta)

	// add software to the host
	software := []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "a", Version: "0.0.2", Source: "deb_packages"}, // different source, so different title than a-chrome
		{Name: "b", Version: "0.0.3", Source: "apps"},
		{Name: "c", Version: "0.0.4", Source: "deb_packages"},
		{Name: "c", Version: "0.0.5", Source: "deb_packages"},
		{Name: "d", Version: "0.0.6", Source: "deb_packages"},
		{Name: "e", Version: "0.0.2", Source: "deb_packages"}, // not vulnerable version
	}
	byNSV := map[string]fleet.Software{}
	for _, s := range software {
		byNSV[s.Name+s.Source+s.Version] = s
	}

	mutationResults, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.Len(t, mutationResults.Inserted, len(software))
	for _, m := range mutationResults.Inserted {
		s, ok := byNSV[m.Name+m.Source+m.Version]
		require.True(t, ok)
		require.Equal(t, m.Name, s.Name, "name")
		require.Equal(t, m.Version, s.Version, "version")
		require.Equal(t, m.Source, s.Source, "source")
		require.Zero(t, s.ID) // not set in the map yet
		require.NotZero(t, m.ID)
		s.ID = m.ID
		byNSV[s.Name+s.Source+s.Version] = s

	}

	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
	require.Equal(t, len(host.Software), len(software))
	for _, hs := range host.Software {
		s, ok := byNSV[hs.Name+hs.Source+hs.Version]
		require.True(t, ok)
		require.Equal(t, hs.Name, s.Name, "name")
		require.Equal(t, hs.Version, s.Version, "version")
		require.Equal(t, hs.Source, s.Source, "source")
		require.Equal(t, hs.ID, s.ID)
	}

	// add other software to the other host, won't be returned
	otherSoftware := []fleet.Software{
		{Name: "a", Version: "0.0.7", Source: "chrome_extensions"},
		{Name: "f", Version: "0.0.8", Source: "chrome_extensions"},
		{Name: "e", Version: "0.0.1", Source: "deb_packages"}, // vulnerable version
	}
	otherSoftwareByNSV := map[string]fleet.Software{}
	for _, s := range otherSoftware {
		otherSoftwareByNSV[s.Name+s.Source+s.Version] = s
	}
	otherMutationResults, err := ds.UpdateHostSoftware(ctx, otherHost.ID, otherSoftware)
	require.NoError(t, err)
	for _, m := range otherMutationResults.Inserted {
		s, ok := otherSoftwareByNSV[m.Name+m.Source+m.Version]
		require.True(t, ok)
		s.ID = m.ID
		otherSoftwareByNSV[s.Name+s.Source+s.Version] = s
	}
	require.NoError(t, ds.LoadHostSoftware(ctx, otherHost, false))

	// shorthand keys for expected software
	a1 := software[0].Name + software[0].Source + software[0].Version
	a2 := software[1].Name + software[1].Source + software[1].Version
	b := software[2].Name + software[2].Source + software[2].Version
	c1 := software[3].Name + software[3].Source + software[3].Version
	c2 := software[4].Name + software[4].Source + software[4].Version
	d := software[5].Name + software[5].Source + software[5].Version
	e2 := software[6].Name + software[6].Source + software[6].Version

	// shorthand keys for other software
	e1 := otherSoftware[2].Name + otherSoftware[2].Source + otherSoftware[2].Version

	// add some vulnerabilities and installed paths
	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0001"},
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0002"},
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0003"},
		{SoftwareID: byNSV[b].ID, CVE: "CVE-b-0001"},
		{SoftwareID: otherSoftwareByNSV[e1].ID, CVE: "CVE-e-0001"},
	}
	for _, v := range vulns {
		_, err = ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
		require.NoError(t, err)
	}

	swPaths := map[string]struct{}{}
	installPaths := make([]string, 0, len(software))
	for _, s := range software {
		path := fmt.Sprintf("/some/path/%s", s.Name)
		key := fmt.Sprintf("%s%s%s%s%s", path, fleet.SoftwareFieldSeparator, "", fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		swPaths[key] = struct{}{}
		installPaths = append(installPaths, path)
	}
	err = ds.UpdateHostSoftwareInstalledPaths(ctx, host.ID, swPaths, mutationResults)
	require.NoError(t, err)

	err = ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)

	expected := map[string]fleet.HostSoftwareWithInstaller{
		byNSV[a1].Name + byNSV[a1].Source: {Name: byNSV[a1].Name, Source: byNSV[a1].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[a1].Version, Vulnerabilities: []string{vulns[0].CVE, vulns[1].CVE, vulns[2].CVE}, InstalledPaths: []string{installPaths[0]}},
		}},
		// a1 and a2 are different software titles because they have different sources
		byNSV[a2].Name + byNSV[a2].Source: {Name: byNSV[a2].Name, Source: byNSV[a2].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[a2].Version, InstalledPaths: []string{installPaths[1]}},
		}},
		byNSV[b].Name + byNSV[b].Source: {Name: byNSV[b].Name, Source: byNSV[b].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[b].Version, Vulnerabilities: []string{vulns[3].CVE}, InstalledPaths: []string{installPaths[2]}},
		}},
		// c1 and c2 are the same software title because they have the same name and source
		byNSV[c1].Name + byNSV[c1].Source: {Name: byNSV[c1].Name, Source: byNSV[c1].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[c1].Version, InstalledPaths: []string{installPaths[3]}},
			{Version: byNSV[c2].Version, InstalledPaths: []string{installPaths[4]}},
		}},
		byNSV[d].Name + byNSV[d].Source: {Name: byNSV[d].Name, Source: byNSV[d].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[d].Version, InstalledPaths: []string{installPaths[5]}},
		}},
		byNSV[e2].Name + byNSV[e2].Source: {Name: byNSV[e2].Name, Source: byNSV[e2].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[e2].Version, InstalledPaths: []string{installPaths[6]}},
		}},
	}

	compareResults := func(expected map[string]fleet.HostSoftwareWithInstaller, got []*fleet.HostSoftwareWithInstaller, expectAsc bool, expectOmitted ...string) {
		gotToString := func() string {
			var builder strings.Builder
			builder.WriteString("Got:\n")
			for _, g := range got {
				builder.WriteString(fmt.Sprintf("%+v\n", g))
			}
			return builder.String()
		}
		require.Len(t, got, len(expected)-len(expectOmitted), gotToString())
		prev := ""
		for _, g := range got {

			for _, omit := range expectOmitted {
				if g.Name+g.Source == omit {
					t.Errorf("Did not expect %s in results", omit)
					continue
				}
			}

			e, ok := expected[g.Name+g.Source]
			require.True(t, ok, "unexpected software %s%s", g.Name, g.Source)
			require.Equal(t, e.Name, g.Name)
			require.Equal(t, e.Source, g.Source)
			if e.SoftwarePackage != nil {
				require.Equal(t, e.SoftwarePackage.SelfService, g.SoftwarePackage.SelfService)
				require.Equal(t, e.SoftwarePackage.IconURL, g.SoftwarePackage.IconURL)
				require.Equal(t, e.SoftwarePackage.AppStoreID, g.SoftwarePackage.AppStoreID)
				require.Equal(t, e.SoftwarePackage.Name, g.SoftwarePackage.Name)
				require.Equal(t, e.SoftwarePackage.Version, g.SoftwarePackage.Version)
				if e.SoftwarePackage.LastInstall != nil {
					require.Equal(t, e.SoftwarePackage.LastInstall.CommandUUID, g.SoftwarePackage.LastInstall.CommandUUID)
					require.Equal(t, e.SoftwarePackage.LastInstall.InstallUUID, g.SoftwarePackage.LastInstall.InstallUUID)
					require.NotNil(t, g.SoftwarePackage.LastInstall.InstalledAt)
				}
				if e.SoftwarePackage.LastUninstall != nil {
					assert.Equal(t, e.SoftwarePackage.LastUninstall.ExecutionID, g.SoftwarePackage.LastUninstall.ExecutionID)
					assert.NotNil(t, g.SoftwarePackage.LastUninstall.UninstalledAt)
				}
			}

			if e.AppStoreApp != nil {
				require.Equal(t, e.AppStoreApp.SelfService, g.AppStoreApp.SelfService)
				require.Equal(t, e.AppStoreApp.IconURL, g.AppStoreApp.IconURL)
				require.Equal(t, e.AppStoreApp.AppStoreID, g.AppStoreApp.AppStoreID)
				require.Equal(t, e.AppStoreApp.Name, g.AppStoreApp.Name)
				require.Equal(t, e.AppStoreApp.Version, g.AppStoreApp.Version)
				if e.AppStoreApp.LastInstall != nil {
					require.Equal(t, e.AppStoreApp.LastInstall.InstallUUID, g.AppStoreApp.LastInstall.InstallUUID)
					require.Equal(t, e.AppStoreApp.LastInstall.CommandUUID, g.AppStoreApp.LastInstall.CommandUUID)
					require.NotNil(t, g.AppStoreApp.LastInstall.InstalledAt)
				}
			}
			// require.Equal(t, e.SoftwarePackage, g.SoftwarePackage)
			// require.Equal(t, e.AppStoreApp, g.AppStoreApp)
			require.Len(t, g.InstalledVersions, len(e.InstalledVersions))
			if len(e.InstalledVersions) > 0 {
				byVers := make(map[string]fleet.HostSoftwareInstalledVersion, len(e.InstalledVersions))
				for _, v := range e.InstalledVersions {
					byVers[v.Version] = *v
				}
				for _, v := range g.InstalledVersions {
					ev, ok := byVers[v.Version]
					require.True(t, ok)
					require.Equal(t, ev.Version, v.Version)
					require.ElementsMatch(t, ev.InstalledPaths, v.InstalledPaths)
					require.ElementsMatch(t, ev.Vulnerabilities, v.Vulnerabilities)
				}
			}
			if prev != "" {
				if expectAsc {
					require.Greater(t, g.Name+g.Source, prev)
				} else {
					require.Less(t, g.Name+g.Source, prev)
				}
			}
			prev = g.Name + g.Source
		}
	}

	// it now returns the software with vulnerabilities and installed paths
	opts.SelfServiceOnly = false
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: 6}, meta)
	compareResults(expected, sw, true)

	opts.VulnerableOnly = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: 2}, meta)
	compareResults(expected, sw, true, byNSV[a2].Name+byNSV[a2].Source, byNSV[c1].Name+byNSV[c1].Source, byNSV[d].Name+byNSV[d].Source, byNSV[e2].Name+byNSV[e2].Source)
	opts.VulnerableOnly = false

	// No software that is available for install
	opts.OnlyAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Empty(t, sw)
	assert.Equal(t, &fleet.PaginationMetadata{}, meta)
	opts.OnlyAvailableForInstall = false

	// create some Fleet installers and map them to a software title,
	// including one for a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	const numberOfSoftwareInstallers = 8
	var swi1Pending, swi2Installed, swi3Failed, swi4Available, swi5Tm, swi6PendingUninstall, swi7FailedUninstall, swi8Uninstalled uint
	var otherHostI1UUID, otherHostI2UUID string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		// keep title id of software B, will use it to associate an installer with it
		var swbTitleID uint
		err := sqlx.GetContext(ctx, q, &swbTitleID, `SELECT id FROM software_titles WHERE name = 'b' AND source = 'apps'`)
		if err != nil {
			return err
		}

		// create the install script content (same for all installers, doesn't matter)
		installScript := `echo 'foo'`
		res, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`, installScript, installScript)
		if err != nil {
			return err
		}
		scriptContentID, _ := res.LastInsertId()

		// create the uninstall script content (same for all installers, doesn't matter)
		uninstallScript := `echo 'bar'`
		resUninstall, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`,
			uninstallScript, uninstallScript)
		if err != nil {
			return err
		}
		uninstallScriptContentID, _ := resUninstall.LastInsertId()

		// create software titles for all but swi1Pending (will be linked to
		// existing software title b)
		var titleIDs []uint
		for i := 0; i < numberOfSoftwareInstallers-1; i++ {
			res, err := q.ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES (?, 'apps')`, fmt.Sprintf("i%d", i))
			if err != nil {
				return err
			}
			id, _ := res.LastInsertId()
			titleIDs = append(titleIDs, uint(id))
		}

		var swiIDs []uint
		for i := 0; i < numberOfSoftwareInstallers; i++ {
			var (
				titleID        uint
				teamID         *uint
				globalOrTeamID uint
			)
			if i == 0 {
				titleID = swbTitleID
			} else {
				titleID = titleIDs[i-1]
			}
			if i == 4 {
				teamID = &tm.ID
				globalOrTeamID = tm.ID
			}
			res, err := q.ExecContext(ctx, `
						INSERT INTO software_installers
							(team_id, global_or_team_id, title_id, filename, extension, version, install_script_content_id, uninstall_script_content_id, storage_id, platform, self_service)
						VALUES
							(?, ?, ?, ?, ?, ?, ?, ?, unhex(?), ?, ?)`,
				teamID, globalOrTeamID, titleID, fmt.Sprintf("installer-%d.pkg", i), "pkg", fmt.Sprintf("v%d.0.0", i), scriptContentID,
				uninstallScriptContentID,
				hex.EncodeToString([]byte("test")), "darwin", i < 2)
			if err != nil {
				return err
			}
			id, _ := res.LastInsertId()
			swiIDs = append(swiIDs, uint(id))
		}
		// sw1Pending and swi2Installed are self-service installers
		swi1Pending, swi2Installed, swi3Failed, swi4Available, swi5Tm,
			swi6PendingUninstall, swi7FailedUninstall, swi8Uninstalled = swiIDs[0], swiIDs[1], swiIDs[2], swiIDs[3], swiIDs[4], swiIDs[5], swiIDs[6], swiIDs[7]

		// create the results for the host

		// swi1 is pending (all results are NULL)
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES (?, ?, ?)`,
			"uuid1", host.ID, swi1Pending)
		if err != nil {
			return err
		}

		// swi2 is installed
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, pre_install_query_output, install_script_exit_code, post_install_script_exit_code)
							VALUES (?, ?, ?, ?, ?, ?)`,
			"uuid2", host.ID, swi2Installed, "ok", 0, 0)
		if err != nil {
			return err
		}

		// swi3 is failed, also add an install request on the other host
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, pre_install_query_output, install_script_exit_code)
							VALUES (?, ?, ?, ?, ?)`,
			"uuid3", host.ID, swi3Failed, "ok", 1)
		if err != nil {
			return err
		}
		otherHostI1UUID = uuid.NewString()
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES (?, ?, ?)`,
			otherHostI1UUID, otherHost.ID, swi3Failed)
		if err != nil {
			return err
		}

		// swi4 is available (no install request), but add a pending request on the other host
		otherHostI2UUID = uuid.NewString()
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES (?, ?, ?)`,
			otherHostI2UUID, otherHost.ID, swi4Available)
		if err != nil {
			return err
		}

		// swi5 is for another team
		_ = swi5Tm

		// swi6 has been installed, and is pending uninstall
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, pre_install_query_output, install_script_exit_code, post_install_script_exit_code)
							VALUES (?, ?, ?, ?, ?, ?)`,
			"uuid6-pre", host.ID, swi6PendingUninstall, "ok", 0, 0)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, uninstall)
							VALUES (?, ?, ?, ?)`,
			"uuid6", host.ID, swi6PendingUninstall, 1)
		require.NoError(t, err)

		// swi7 is failed uninstall
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, uninstall, uninstall_script_exit_code)
							VALUES (?, ?, ?, ?, ?)`,
			"uuid7", host.ID, swi7FailedUninstall, 1, 1)
		require.NoError(t, err)

		// swi8 is successful uninstall
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, uninstall, uninstall_script_exit_code)
							VALUES (?, ?, ?, ?, ?)`,
			"uuid8", host.ID, swi8Uninstalled, 1, 0)
		require.NoError(t, err)

		return nil
	})

	// swi1Pending uses software title id of "b"
	expected[byNSV[b].Name+byNSV[b].Source] = fleet.HostSoftwareWithInstaller{
		Name:            "b",
		Source:          "apps",
		Status:          expectStatus(fleet.SoftwareInstallPending),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-0.pkg", Version: "v0.0.0", SelfService: ptr.Bool(true), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: "uuid1"}},
		InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[b].Version, Vulnerabilities: []string{vulns[3].CVE}, InstalledPaths: []string{installPaths[2]}},
		},
	}
	i0 := fleet.HostSoftwareWithInstaller{
		Name:            "i0",
		Source:          "apps",
		Status:          expectStatus(fleet.SoftwareInstalled),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-1.pkg", Version: "v1.0.0", SelfService: ptr.Bool(true), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: "uuid2"}},
	}
	expected[i0.Name+i0.Source] = i0

	i1 := fleet.HostSoftwareWithInstaller{
		Name:            "i1",
		Source:          "apps",
		Status:          expectStatus(fleet.SoftwareInstallFailed),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-2.pkg", Version: "v2.0.0", SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: "uuid3"}},
	}
	expected[i1.Name+i1.Source] = i1

	i4 := fleet.HostSoftwareWithInstaller{
		Name:   "i4",
		Source: "apps",
		Status: expectStatus(fleet.SoftwareUninstallPending),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{
			Name: "installer-5.pkg", Version: "v5.0.0", SelfService: ptr.Bool(false),
			LastInstall:   &fleet.HostSoftwareInstall{InstallUUID: "uuid6-pre"},
			LastUninstall: &fleet.HostSoftwareUninstall{ExecutionID: "uuid6"},
		},
	}
	expected[i4.Name+i4.Source] = i4

	i5 := fleet.HostSoftwareWithInstaller{
		Name:   "i5",
		Source: "apps",
		Status: expectStatus(fleet.SoftwareUninstallFailed),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{
			Name: "installer-6.pkg", Version: "v6.0.0", SelfService: ptr.Bool(false),
			LastUninstall: &fleet.HostSoftwareUninstall{ExecutionID: "uuid7"},
		},
	}
	expected[i5.Name+i5.Source] = i5

	i6 := fleet.HostSoftwareWithInstaller{
		Name:   "i6",
		Source: "apps",
		Status: nil,
		SoftwarePackage: &fleet.SoftwarePackageOrApp{
			Name: "installer-7.pkg", Version: "v7.0.0", SelfService: ptr.Bool(false),
			LastUninstall: &fleet.HostSoftwareUninstall{ExecutionID: "uuid8"},
		},
	}
	expected[i6.Name+i6.Source] = i6

	// request without available software
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected))}, meta)
	compareResults(expected, sw, true)

	// request with available software
	i2 := fleet.HostSoftwareWithInstaller{
		Name:            "i2",
		Source:          "apps",
		Status:          nil,
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-3.pkg", Version: "v3.0.0", SelfService: ptr.Bool(false)},
	}
	expected[i2.Name+i2.Source] = i2

	i3 := fleet.HostSoftwareWithInstaller{
		Name:            "i3",
		Source:          "apps",
		Status:          nil,
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-4.pkg", Version: "v4.0.0", SelfService: ptr.Bool(false)},
	}
	expected[i3.Name+i3.Source] = i3

	opts.IncludeAvailableForInstall = true
	opts.ListOptions.PerPage = 20
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 1}, meta)
	compareResults(expected, sw, true, i3.Name+i3.Source)

	// request with available software only (attempted to install and never attempted to install)
	expectedAvailableOnly := map[string]fleet.HostSoftwareWithInstaller{}
	expectedAvailableOnly[byNSV[b].Name+byNSV[b].Source] = expected[byNSV[b].Name+byNSV[b].Source]
	expectedAvailableOnly[i0.Name+i0.Source] = i0
	expectedAvailableOnly[i1.Name+i1.Source] = i1
	expectedAvailableOnly[i2.Name+i2.Source] = i2
	expectedAvailableOnly[i4.Name+i4.Source] = i4
	expectedAvailableOnly[i5.Name+i5.Source] = i5
	expectedAvailableOnly[i6.Name+i6.Source] = i6
	opts.OnlyAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expectedAvailableOnly))}, meta)
	compareResults(expectedAvailableOnly, sw, true)
	opts.OnlyAvailableForInstall = false

	// request in descending order
	opts.ListOptions.OrderDirection = fleet.OrderDescending
	opts.ListOptions.TestSecondaryOrderDirection = fleet.OrderDescending
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 2}, meta)
	compareResults(expected, sw, false, i2.Name+i2.Source, i3.Name+i3.Source)
	opts.ListOptions.OrderDirection = fleet.OrderAscending
	opts.ListOptions.TestSecondaryOrderDirection = fleet.OrderAscending

	// record a new install request for i1, this time as pending, and mark install request for b (swi1) as failed
	time.Sleep(time.Second) // ensure the timestamp is later
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           "uuid1",
		InstallScriptExitCode: ptr.Int(2),
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		// swi3 has a new install request pending
		_, err = q.ExecContext(ctx, `
							INSERT INTO host_software_installs (execution_id, host_id, software_installer_id)
							VALUES (?, ?, ?)`,
			"uuid4", host.ID, swi3Failed)
		if err != nil {
			return err
		}
		return nil
	})

	expected[byNSV[b].Name+byNSV[b].Source] = fleet.HostSoftwareWithInstaller{
		Name:            "b",
		Source:          "apps",
		Status:          expectStatus(fleet.SoftwareInstallFailed),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-0.pkg", Version: "v0.0.0", SelfService: ptr.Bool(true), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: "uuid1"}},
		InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: byNSV[b].Version, Vulnerabilities: []string{vulns[3].CVE}, InstalledPaths: []string{installPaths[2]}},
		},
	}
	expected[i1.Name+i1.Source] = fleet.HostSoftwareWithInstaller{
		Name:            "i1",
		Source:          "apps",
		Status:          expectStatus(fleet.SoftwareInstallPending),
		SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-2.pkg", Version: "v2.0.0", SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: "uuid4"}},
	}
	expectedAvailableOnly[byNSV[b].Name+byNSV[b].Source] = expected[byNSV[b].Name+byNSV[b].Source]
	expectedAvailableOnly[i1.Name+i1.Source] = expected[i1.Name+i1.Source]

	// request without available software
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 2}, meta)
	compareResults(expected, sw, true, i2.Name+i2.Source, i3.Name+i3.Source)

	// request with available software
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 1}, meta)
	compareResults(expected, sw, true, i3.Name+i3.Source)

	// create a new host in the team, with no software
	tmHost := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now(), test.WithPlatform("darwin"))
	nanoEnroll(t, ds, tmHost, false)
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{tmHost.ID})
	require.NoError(t, err)
	tmHost.TeamID = &tm.ID

	// no installed software for this host
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, tmHost, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	require.Equal(t, &fleet.PaginationMetadata{}, meta)

	// sees the available installer in its team
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, tmHost, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: 1}, meta)
	compareResults(map[string]fleet.HostSoftwareWithInstaller{
		i3.Name + i3.Source: expected[i3.Name+i3.Source],
	}, sw, true)

	// test with a search query (searches on name), with and without available software
	opts.ListOptions.MatchQuery = "a"
	opts.IncludeAvailableForInstall = false
	sw, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	compareResults(map[string]fleet.HostSoftwareWithInstaller{
		byNSV[a1].Name + byNSV[a1].Source: expected[byNSV[a1].Name+byNSV[a1].Source],
		byNSV[a2].Name + byNSV[a2].Source: expected[byNSV[a2].Name+byNSV[a2].Source],
	}, sw, true)
	opts.IncludeAvailableForInstall = true
	sw, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	compareResults(map[string]fleet.HostSoftwareWithInstaller{
		byNSV[a1].Name + byNSV[a1].Source: expected[byNSV[a1].Name+byNSV[a1].Source],
		byNSV[a2].Name + byNSV[a2].Source: expected[byNSV[a2].Name+byNSV[a2].Source],
	}, sw, true)

	opts.ListOptions.MatchQuery = "zz"
	opts.IncludeAvailableForInstall = false
	sw, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)
	opts.IncludeAvailableForInstall = true
	sw, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, sw)

	// add VPP apps, one for both no team and team, and two for no-team only.
	va1, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.MacOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.IOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.MacOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, &tm.ID)
	require.NoError(t, err)
	vpp1 := va1.AdamID
	va2, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_2", Platform: fleet.MacOSPlatform}}, Name: "vpp2",
			BundleIdentifier: "com.app.vpp2",
		}, nil)
	require.NoError(t, err)
	// create vpp3 app that allows self-service
	va3, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_3", Platform: fleet.MacOSPlatform}, SelfService: true}, Name: "vpp3",
			BundleIdentifier: "com.app.vpp3",
		}, nil)
	require.NoError(t, err)
	vpp2, vpp3 := va2.AdamID, va3.AdamID

	// create an installation request for vpp1 and vpp2, leaving vpp3 as
	// available only
	vpp1CmdUUID := createVPPAppInstallRequest(t, ds, host, vpp1, user.ID)
	vpp2CmdUUID := createVPPAppInstallRequest(t, ds, host, vpp2, user.ID)
	// make vpp1 install a success, while vpp2 has its initial request as failed
	// and a subsequent request as pending.
	createVPPAppInstallResult(t, ds, host, vpp1CmdUUID, fleet.MDMAppleStatusAcknowledged)
	createVPPAppInstallResult(t, ds, host, vpp2CmdUUID, fleet.MDMAppleStatusError)
	time.Sleep(time.Second) // ensure a different created_at timestamp
	vpp2bCmdUUID := createVPPAppInstallRequest(t, ds, host, vpp2, user.ID)
	require.NotEmpty(t, vpp2bCmdUUID)
	// add an install request for the team host on vpp1, should not impact
	// main host
	vpp1TmCmdUUID := createVPPAppInstallRequest(t, ds, tmHost, vpp1, user.ID)
	require.NotEmpty(t, vpp1TmCmdUUID)

	expected["vpp1apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp1",
		Source:      "apps",
		Status:      expectStatus(fleet.SoftwareInstalled),
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp1, SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{CommandUUID: vpp1CmdUUID}},
	}
	expected["vpp2apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp2",
		Source:      "apps",
		Status:      expectStatus(fleet.SoftwareInstallPending),
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp2, SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{CommandUUID: vpp2bCmdUUID}},
	}

	opts.IncludeAvailableForInstall = false
	opts.ListOptions.MatchQuery = ""
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 2}, meta)
	compareResults(expected, sw, true, i3.Name+i3.Source, i2.Name+i2.Source) // i3 is for team, i2 is available (excluded)

	expected["vpp3apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp3",
		Source:      "apps",
		Status:      nil,
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp3, SelfService: ptr.Bool(true)},
	}

	expectedAvailableOnly["vpp1apps"] = expected["vpp1apps"]
	expectedAvailableOnly["vpp2apps"] = expected["vpp2apps"]
	expectedAvailableOnly["vpp3apps"] = expected["vpp3apps"]
	opts.IncludeAvailableForInstall = true
	opts.IsMDMEnrolled = true
	opts.ListOptions.PerPage = 20
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 1}, meta)
	compareResults(expected, sw, true, i3.Name+i3.Source) // i3 is for team

	// Host is not MDM enrolled - we should not get "vpp3papps" because we can't install it.
	opts.IsMDMEnrolled = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected)) - 2}, meta)
	compareResults(expected, sw, true, "vpp3apps", i3.Name+i3.Source) // i3 is for team

	// Available for install only
	opts.OnlyAvailableForInstall = true
	opts.IsMDMEnrolled = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expectedAvailableOnly))}, meta)
	compareResults(expectedAvailableOnly, sw, true)

	// Available for install only with host not MDM enrolled
	// We should only exclude "vpp3apps", because it was not installed previously and we can't
	// install it without MDM
	opts.IsMDMEnrolled = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expectedAvailableOnly) - 1)}, meta)
	compareResults(expectedAvailableOnly, sw, true, "vpp3apps")
	opts.IsMDMEnrolled = false
	opts.OnlyAvailableForInstall = false

	// team host sees available i3 and pending vpp1
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, tmHost, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: 2}, meta)
	compareResults(map[string]fleet.HostSoftwareWithInstaller{
		i3.Name + i3.Source: expected[i3.Name+i3.Source],
		"vpp1apps": {
			Name:        "vpp1",
			Source:      "apps",
			Status:      expectStatus(fleet.SoftwareInstallPending),
			AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp1, SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{CommandUUID: vpp1TmCmdUUID}},
		},
	}, sw, true)

	// other host does not see available VPP apps because it is a linux host
	opts.IncludeAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, otherHost, opts)
	require.NoError(t, err)
	require.Equal(t, &fleet.PaginationMetadata{TotalResults: 5}, meta)

	expectedOther := map[string]fleet.HostSoftwareWithInstaller{
		otherSoftware[0].Name + otherSoftware[0].Source: {Name: otherSoftware[0].Name, Source: otherSoftware[0].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: otherSoftware[0].Version},
		}},
		otherSoftware[1].Name + otherSoftware[1].Source: {Name: otherSoftware[1].Name, Source: otherSoftware[1].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: otherSoftware[1].Version},
		}},
		otherSoftware[2].Name + otherSoftware[2].Source: {Name: otherSoftware[2].Name, Source: otherSoftware[2].Source, InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
			{Version: otherSoftware[2].Version, Vulnerabilities: []string{vulns[4].CVE}},
		}},
		"i1apps": {
			Name:            "i1",
			Source:          "apps",
			Status:          expectStatus(fleet.SoftwareInstallPending),
			SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-2.pkg", Version: "v2.0.0", SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: otherHostI1UUID}},
		},
		"i2apps": {
			Name:            "i2",
			Source:          "apps",
			Status:          expectStatus(fleet.SoftwareInstallPending),
			SoftwarePackage: &fleet.SoftwarePackageOrApp{Name: "installer-3.pkg", Version: "v3.0.0", SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{InstallUUID: otherHostI2UUID}},
		},
	}
	compareResults(expectedOther, sw, true)

	// test the pagination
	cases := []struct {
		name      string
		opts      fleet.HostSoftwareTitleListOptions
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			name:      "No available for install software, page 0",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{PerPage: 5}, IncludeAvailableForInstall: false},
			wantNames: []string{byNSV[a1].Name, byNSV[a2].Name, byNSV[b].Name, byNSV[c1].Name, byNSV[d].Name},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 13},
		},
		{
			name:      "No available for install software, page 1",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 1, PerPage: 5}, IncludeAvailableForInstall: false},
			wantNames: []string{byNSV[e2].Name, i0.Name, i1.Name, i4.Name, i5.Name},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 13},
		},
		{
			name:      "No available for install software, page 2",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 2, PerPage: 5}, IncludeAvailableForInstall: false},
			wantNames: []string{i6.Name, "vpp1", "vpp2"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 13},
		},
		{
			name:      "No available for install software, page 3",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 3, PerPage: 5}, IncludeAvailableForInstall: false},
			wantNames: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 13},
		},
		{
			name:      "Include Available for install software with MDM on, page 0",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{PerPage: 5}, IncludeAvailableForInstall: true, IsMDMEnrolled: true},
			wantNames: []string{byNSV[a1].Name, byNSV[a2].Name, byNSV[b].Name, byNSV[c1].Name, byNSV[d].Name},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 15},
		},
		{
			name:      "Include Available for install software with MDM on, page 1",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 1, PerPage: 5}, IncludeAvailableForInstall: true, IsMDMEnrolled: true},
			wantNames: []string{byNSV[e2].Name, i0.Name, i1.Name, i2.Name, i4.Name},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 15},
		},
		{
			name: "Include Available for install software with MDM on, page 2",
			opts: fleet.HostSoftwareTitleListOptions{
				ListOptions:                fleet.ListOptions{Page: 2, PerPage: 5},
				IncludeAvailableForInstall: true,
				IsMDMEnrolled:              true,
			},
			wantNames: []string{i5.Name, i6.Name, "vpp1", "vpp2", "vpp3"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 15},
		},

		{
			// Excludes vpp3 because it was never installed, and we can't install it with MDM off
			name: "Include Available for install software with MDM off, page 2",
			opts: fleet.HostSoftwareTitleListOptions{
				ListOptions:                fleet.ListOptions{Page: 2, PerPage: 5},
				IncludeAvailableForInstall: true,
			},
			wantNames: []string{i5.Name, i6.Name, "vpp1", "vpp2"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 14},
		},
		{
			name: "Include Available for install software with MDM on, page 3",
			opts: fleet.HostSoftwareTitleListOptions{
				ListOptions:                fleet.ListOptions{Page: 3, PerPage: 5},
				IncludeAvailableForInstall: true,
				IsMDMEnrolled:              true,
			},
			wantNames: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 15},
		},
		{
			name:      "Available for install and self-service only software with MDM on, page 0",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{PerPage: 3}, IncludeAvailableForInstall: true, SelfServiceOnly: true, IsMDMEnrolled: true},
			wantNames: []string{byNSV[b].Name, i0.Name, "vpp3"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 3},
		},
		{
			name:      "Available for install and self-service only software with MDM on, page 1",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 1, PerPage: 3}, IncludeAvailableForInstall: true, SelfServiceOnly: true, IsMDMEnrolled: true},
			wantNames: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 3},
		},
		{
			name:      "Available for install and self-service only software with MDM off, page 0",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{PerPage: 3}, IncludeAvailableForInstall: true, SelfServiceOnly: true},
			wantNames: []string{byNSV[b].Name, i0.Name},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 2},
		},
		{
			name:      "Only available for install software with MDM on, page 0",
			opts:      fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{Page: 0, PerPage: 4}, OnlyAvailableForInstall: true, IsMDMEnrolled: true},
			wantNames: []string{byNSV[b].Name, "i0", "i1", "i2"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 10},
		},
		{
			name: "Only available for install with MDM on, page 2",
			opts: fleet.HostSoftwareTitleListOptions{
				ListOptions:             fleet.ListOptions{Page: 2, PerPage: 4},
				OnlyAvailableForInstall: true,
				IsMDMEnrolled:           true,
			},
			wantNames: []string{"vpp2", "vpp3"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 10},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// always include metadata
			c.opts.ListOptions.IncludeMetadata = true
			c.opts.ListOptions.OrderKey = "name"
			c.opts.ListOptions.TestSecondaryOrderKey = "source"

			sw, meta, err := ds.ListHostSoftware(ctx, host, c.opts)
			require.NoError(t, err)

			names := make([]string, 0, len(sw))
			for _, s := range sw {
				names = append(names, s.Name)
			}
			assert.Equal(t, c.wantNames, names)
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}

func testListIOSHostSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithPlatform("ios"))
	nanoEnroll(t, ds, host, false)
	opts := fleet.HostSoftwareTitleListOptions{ListOptions: fleet.ListOptions{
		PerPage: 10, IncludeMetadata: true, OrderKey: "name",
		TestSecondaryOrderKey: "source",
	}}

	test.CreateInsertGlobalVPPToken(t, ds)

	user, err := ds.NewUser(ctx, &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "userIOS",
		Email:      "userIOS@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	expectStatus := func(s fleet.SoftwareInstallerStatus) *fleet.SoftwareInstallerStatus {
		return &s
	}

	// no software yet
	sw, meta, err := ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Empty(t, sw)
	assert.Equal(t, &fleet.PaginationMetadata{}, meta)

	// add software to the host
	software := []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "ios_apps"},
		{Name: "b", Version: "0.0.2", Source: "ios_apps"},
		{Name: "c", Version: "0.0.3", Source: "ios_apps"},
		{Name: "c", Version: "0.0.4", Source: "ios_apps"},
	}
	byNSV := map[string]fleet.Software{}
	for _, s := range software {
		byNSV[s.Name+s.Source+s.Version] = s
	}

	mutationResults, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	assert.Len(t, mutationResults.Inserted, len(software))
	for _, m := range mutationResults.Inserted {
		s, ok := byNSV[m.Name+m.Source+m.Version]
		assert.True(t, ok)
		assert.Equal(t, m.Name, s.Name, "name")
		assert.Equal(t, m.Version, s.Version, "version")
		assert.Equal(t, m.Source, s.Source, "source")
		assert.Zero(t, s.ID) // not set in the map yet
		assert.NotZero(t, m.ID)
		s.ID = m.ID
		byNSV[s.Name+s.Source+s.Version] = s

	}

	assert.NoError(t, ds.LoadHostSoftware(ctx, host, false))
	assert.Equal(t, len(host.Software), len(software))
	for _, hs := range host.Software {
		s, ok := byNSV[hs.Name+hs.Source+hs.Version]
		assert.True(t, ok)
		assert.Equal(t, hs.Name, s.Name, "name")
		assert.Equal(t, hs.Version, s.Version, "version")
		assert.Equal(t, hs.Source, s.Source, "source")
		assert.Equal(t, hs.ID, s.ID)
	}

	// shorthand keys for expected software
	getKey := func(i int) string {
		return software[i].Name + software[i].Source + software[i].Version
	}
	a1 := getKey(0)
	b := getKey(1)
	c1 := getKey(2)
	c2 := getKey(3)

	// add some vulnerabilities
	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0001"},
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0002"},
		{SoftwareID: byNSV[a1].ID, CVE: "CVE-a-0003"},
		{SoftwareID: byNSV[b].ID, CVE: "CVE-b-0001"},
	}
	for _, v := range vulns {
		_, err = ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
		require.NoError(t, err)
	}

	err = ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)

	expected := map[string]fleet.HostSoftwareWithInstaller{
		byNSV[a1].Name + byNSV[a1].Source: {
			Name: byNSV[a1].Name, Source: byNSV[a1].Source,
			InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
				{Version: byNSV[a1].Version, Vulnerabilities: []string{vulns[0].CVE, vulns[1].CVE, vulns[2].CVE}},
			},
		},
		byNSV[b].Name + byNSV[b].Source: {
			Name: byNSV[b].Name, Source: byNSV[b].Source,
			InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
				{Version: byNSV[b].Version, Vulnerabilities: []string{vulns[3].CVE}},
			},
		},
		// c1 and c2 are the same software title because they have the same name and source
		byNSV[c1].Name + byNSV[c1].Source: {
			Name: byNSV[c1].Name, Source: byNSV[c1].Source,
			InstalledVersions: []*fleet.HostSoftwareInstalledVersion{
				{Version: byNSV[c1].Version},
				{Version: byNSV[c2].Version},
			},
		},
	}

	compareResults := func(expected map[string]fleet.HostSoftwareWithInstaller, got []*fleet.HostSoftwareWithInstaller, expectAsc bool,
		expectOmitted ...string,
	) {
		require.Len(t, got, len(expected)-len(expectOmitted))
		prev := ""
		for _, g := range got {
			e, ok := expected[g.Name+g.Source]
			require.True(t, ok, "unexpected software name:%s source:%s", g.Name, g.Source)
			require.Equal(t, e.Name, g.Name)
			require.Equal(t, e.Source, g.Source)
			if e.SoftwarePackage != nil {
				require.Equal(t, e.SoftwarePackage.SelfService, g.SoftwarePackage.SelfService)
				require.Equal(t, e.SoftwarePackage.IconURL, g.SoftwarePackage.IconURL)
				require.Equal(t, e.SoftwarePackage.AppStoreID, g.SoftwarePackage.AppStoreID)
				require.Equal(t, e.SoftwarePackage.Name, g.SoftwarePackage.Name)
				require.Equal(t, e.SoftwarePackage.Version, g.SoftwarePackage.Version)
				if e.SoftwarePackage.LastInstall != nil {
					require.Equal(t, e.SoftwarePackage.LastInstall.CommandUUID, g.SoftwarePackage.LastInstall.CommandUUID)
					require.Equal(t, e.SoftwarePackage.LastInstall.InstallUUID, g.SoftwarePackage.LastInstall.InstallUUID)
					require.NotNil(t, g.SoftwarePackage.LastInstall.InstalledAt)
				}
			}

			if e.AppStoreApp != nil {
				require.Equal(t, e.AppStoreApp.SelfService, g.AppStoreApp.SelfService)
				require.Equal(t, e.AppStoreApp.IconURL, g.AppStoreApp.IconURL)
				require.Equal(t, e.AppStoreApp.AppStoreID, g.AppStoreApp.AppStoreID)
				require.Equal(t, e.AppStoreApp.Name, g.AppStoreApp.Name)
				require.Equal(t, e.AppStoreApp.Version, g.AppStoreApp.Version)
				if e.AppStoreApp.LastInstall != nil {
					require.Equal(t, e.AppStoreApp.LastInstall.InstallUUID, g.AppStoreApp.LastInstall.InstallUUID)
					require.Equal(t, e.AppStoreApp.LastInstall.CommandUUID, g.AppStoreApp.LastInstall.CommandUUID)
					require.NotNil(t, g.AppStoreApp.LastInstall.InstalledAt)
				}
			}
			require.Len(t, g.InstalledVersions, len(e.InstalledVersions))
			if len(e.InstalledVersions) > 0 {
				byVers := make(map[string]fleet.HostSoftwareInstalledVersion, len(e.InstalledVersions))
				for _, v := range e.InstalledVersions {
					byVers[v.Version] = *v
				}
				for _, v := range g.InstalledVersions {
					ev, ok := byVers[v.Version]
					require.True(t, ok)
					require.Equal(t, ev.Version, v.Version)
					require.ElementsMatch(t, ev.InstalledPaths, v.InstalledPaths)
					require.ElementsMatch(t, ev.Vulnerabilities, v.Vulnerabilities)
				}
			}
			if prev != "" {
				if expectAsc {
					require.Greater(t, g.Name+g.Source, prev)
				} else {
					require.Less(t, g.Name+g.Source, prev)
				}
			}
			prev = g.Name + g.Source
		}
	}

	// it now returns the software with vulnerabilities and installed paths
	opts.SelfServiceOnly = false
	opts.IncludeAvailableForInstall = false
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected))}, meta)
	compareResults(expected, sw, true)

	opts.VulnerableOnly = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected) - 1)}, meta)
	compareResults(expected, sw, true, byNSV[c1].Name+byNSV[c1].Source)
	opts.VulnerableOnly = false

	// No software that is available for install
	opts.OnlyAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Empty(t, sw)
	assert.Equal(t, &fleet.PaginationMetadata{}, meta)
	opts.OnlyAvailableForInstall = false

	// Create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "mobile team"})
	require.NoError(t, err)

	// add VPP apps, one for both no team and team, and three for no-team only.
	va1, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.IOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.MacOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.IPadOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.IPadOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, &tm.ID)
	require.NoError(t, err)
	vpp1 := va1.AdamID
	va2, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_2", Platform: fleet.IOSPlatform}}, Name: "vpp2",
			BundleIdentifier: "com.app.vpp2",
		}, nil)
	require.NoError(t, err)
	va3, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_3", Platform: fleet.IOSPlatform}}, Name: "vpp3",
			BundleIdentifier: "com.app.vpp3",
		}, nil)
	require.NoError(t, err)
	va4, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_4", Platform: fleet.IOSPlatform}}, Name: "vpp4",
			BundleIdentifier: "com.app.vpp4",
		}, nil)
	require.NoError(t, err)
	vpp2, vpp3, vpp4 := va2.AdamID, va3.AdamID, va4.AdamID

	// create an installation request for vpp1 and vpp2, leaving vpp3 and vpp4 as
	// available only
	vpp1CmdUUID := createVPPAppInstallRequest(t, ds, host, vpp1, user.ID)
	vpp2CmdUUID := createVPPAppInstallRequest(t, ds, host, vpp2, user.ID)
	// make vpp1 install a success, while vpp2 has its initial request as failed
	// and a subsequent request as pending.
	createVPPAppInstallResult(t, ds, host, vpp1CmdUUID, fleet.MDMAppleStatusAcknowledged)
	createVPPAppInstallResult(t, ds, host, vpp2CmdUUID, fleet.MDMAppleStatusError)
	time.Sleep(time.Second) // ensure a different created_at timestamp
	vpp2bCmdUUID := createVPPAppInstallRequest(t, ds, host, vpp2, user.ID)
	require.NotEmpty(t, vpp2bCmdUUID)

	expected["vpp1ios_apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp1",
		Source:      "ios_apps",
		Status:      expectStatus(fleet.SoftwareInstalled),
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp1, SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{CommandUUID: vpp1CmdUUID}},
	}
	expected["vpp2ios_apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp2",
		Source:      "ios_apps",
		Status:      expectStatus(fleet.SoftwareInstallPending),
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp2, SelfService: ptr.Bool(false), LastInstall: &fleet.HostSoftwareInstall{CommandUUID: vpp2bCmdUUID}},
	}

	opts.IncludeAvailableForInstall = false
	opts.ListOptions.MatchQuery = ""
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected))}, meta)
	compareResults(expected, sw, true) // i3 is for team, i2 is available (excluded)

	expected["vpp3ios_apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp3",
		Source:      "ios_apps",
		Status:      nil,
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp3, SelfService: ptr.Bool(false)},
	}
	expected["vpp4ios_apps"] = fleet.HostSoftwareWithInstaller{
		Name:        "vpp4",
		Source:      "ios_apps",
		Status:      nil,
		AppStoreApp: &fleet.SoftwarePackageOrApp{AppStoreID: vpp4, SelfService: ptr.Bool(false)},
	}
	expectedAvailableOnly := map[string]fleet.HostSoftwareWithInstaller{}
	expectedAvailableOnly["vpp1ios_apps"] = expected["vpp1ios_apps"]
	expectedAvailableOnly["vpp2ios_apps"] = expected["vpp2ios_apps"]
	expectedAvailableOnly["vpp3ios_apps"] = expected["vpp3ios_apps"]
	expectedAvailableOnly["vpp4ios_apps"] = expected["vpp4ios_apps"]
	opts.IncludeAvailableForInstall = true
	opts.IsMDMEnrolled = true
	opts.ListOptions.PerPage = 20
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expected))}, meta)
	compareResults(expected, sw, true)

	// Available for install only
	opts.OnlyAvailableForInstall = true
	sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	assert.Equal(t, &fleet.PaginationMetadata{TotalResults: uint(len(expectedAvailableOnly))}, meta)
	compareResults(expectedAvailableOnly, sw, true)
	opts.OnlyAvailableForInstall = false
}

func testSetHostSoftwareInstallResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// create a software installer and some host install requests
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		installScript := `echo 'foo'`
		res, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`, installScript, installScript)
		if err != nil {
			return err
		}
		scriptContentID, _ := res.LastInsertId()

		uninstallScript := `echo 'bar'`
		resUninstall, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`,
			uninstallScript, uninstallScript)
		if err != nil {
			return err
		}
		uninstallScriptContentID, _ := resUninstall.LastInsertId()

		res, err = q.ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('foo', 'apps')`)
		if err != nil {
			return err
		}
		titleID, _ := res.LastInsertId()

		res, err = q.ExecContext(ctx, `
			INSERT INTO software_installers
				(title_id, filename, extension, version, install_script_content_id, uninstall_script_content_id, storage_id)
			VALUES
				(?, ?, ?, ?, ?, ?, unhex(?))`,
			titleID, "installer.pkg", "pkg", "v1.0.0", scriptContentID, uninstallScriptContentID, hex.EncodeToString([]byte("test")))
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()

		// create some install requests for the host
		for i := 0; i < 3; i++ {
			_, err = q.ExecContext(ctx, `
			INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES (?, ?, ?)`,
				fmt.Sprintf("uuid%d", i), host.ID, id)
			if err != nil {
				return err
			}
		}
		return nil
	})

	checkResults := func(want *fleet.HostSoftwareInstallResultPayload) {
		type result struct {
			HostID                    uint    `db:"host_id"`
			InstallUUID               string  `db:"execution_id"`
			PreInstallConditionOutput *string `db:"pre_install_query_output"`
			InstallScriptExitCode     *int    `db:"install_script_exit_code"`
			InstallScriptOutput       *string `db:"install_script_output"`
			PostInstallScriptExitCode *int    `db:"post_install_script_exit_code"`
			PostInstallScriptOutput   *string `db:"post_install_script_output"`
		}
		var got result
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &got,
				`SELECT
					host_id,
					execution_id,
					pre_install_query_output,
					install_script_exit_code,
					install_script_output,
					post_install_script_exit_code,
					post_install_script_output
				FROM
					host_software_installs
				WHERE execution_id = ?`, want.InstallUUID)
		})
		assert.Equal(t, want.HostID, got.HostID)
		assert.Equal(t, want.InstallUUID, got.InstallUUID)
		if want.PreInstallConditionOutput == nil {
			assert.Nil(t, got.PreInstallConditionOutput)
		} else {
			assert.NotNil(t, got.PreInstallConditionOutput)
			assert.Equal(t, *want.PreInstallConditionOutput, *got.PreInstallConditionOutput)
		}
		assert.Equal(t, want.InstallScriptExitCode, got.InstallScriptExitCode)
		if want.InstallScriptOutput == nil {
			assert.Nil(t, got.InstallScriptOutput)
		} else {
			assert.NotNil(t, got.InstallScriptOutput)
			assert.EqualValues(t, want.InstallScriptOutput, got.InstallScriptOutput)
		}
		assert.Equal(t, want.PostInstallScriptExitCode, got.PostInstallScriptExitCode)
		if want.PostInstallScriptOutput == nil {
			assert.Nil(t, got.PostInstallScriptOutput)
		} else {
			assert.NotNil(t, got.PostInstallScriptOutput)
			assert.EqualValues(t, want.InstallScriptOutput, got.InstallScriptOutput)
		}
	}

	// set a result with all fields provided
	want := &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host.ID,
		InstallUUID:               "uuid0",
		PreInstallConditionOutput: ptr.String("1"),
		InstallScriptExitCode:     ptr.Int(0),
		InstallScriptOutput:       ptr.String("ok"),
		PostInstallScriptExitCode: ptr.Int(0),
		PostInstallScriptOutput:   ptr.String("ok"),
	}
	err := ds.SetHostSoftwareInstallResult(ctx, want)
	require.NoError(t, err)
	checkResults(want)

	// set a result with only the pre-condition that failed
	want = &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host.ID,
		InstallUUID:               "uuid1",
		PreInstallConditionOutput: ptr.String(""),
	}
	err = ds.SetHostSoftwareInstallResult(ctx, want)
	require.NoError(t, err)
	checkResults(want)

	// set a result with only the install that failed
	want = &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           "uuid2",
		InstallScriptExitCode: ptr.Int(1),
		InstallScriptOutput:   ptr.String("fail"),
	}
	err = ds.SetHostSoftwareInstallResult(ctx, want)
	require.NoError(t, err)
	checkResults(want)

	// set a result for a non-existing uuid
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           "uuid-no-such",
		InstallScriptExitCode: ptr.Int(0),
		InstallScriptOutput:   ptr.String("ok"),
	})
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

func testListHostSoftwareInstallThenTransferTeam(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "user1", "user1@example.com", false)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithPlatform("darwin"))
	nanoEnroll(t, ds, host, false)
	opts := fleet.HostSoftwareTitleListOptions{
		ListOptions:                fleet.ListOptions{PerPage: 10, IncludeMetadata: true, OrderKey: "name", TestSecondaryOrderKey: "source"},
		IncludeAvailableForInstall: true,
	}

	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{host.ID})
	require.NoError(t, err)
	host.TeamID = &team1.ID

	// add a single "externally-installed" software for that host
	software := []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// create a software installer for team 1
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerTm1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "hello",
		InstallerFile:   tfr1,
		StorageID:       "storage1",
		Filename:        "file1",
		Title:           "file1",
		Version:         "1.0",
		Source:          "apps",
		TeamID:          &team1.ID,
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// install it on the host
	hostInstall1, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerTm1, false, nil)
	require.NoError(t, err)
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           hostInstall1,
		InstallScriptExitCode: ptr.Int(0),
	})
	require.NoError(t, err)

	// add a VPP app for team 1
	vppTm1, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.MacOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1",
		}, &team1.ID)
	require.NoError(t, err)

	// fail to install it on the host
	vpp1CmdUUID := createVPPAppInstallRequest(t, ds, host, vppTm1.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, host, vpp1CmdUUID, fleet.MDMAppleStatusError)

	// add the successful installer to the reported installed software
	software = []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "file1", Version: "1.0", Source: "apps"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// listing the host's software (including available for install) at this
	// point lists "a", "file1" and "vpp1" (because of the install attempt)
	sw, meta, err := ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Len(t, sw, 3)
	require.EqualValues(t, 3, meta.TotalResults)
	require.Equal(t, sw[0].Name, "a")
	require.Nil(t, sw[0].AppStoreApp)
	require.Nil(t, sw[0].SoftwarePackage)
	require.Equal(t, sw[1].Name, "file1")
	require.Nil(t, sw[1].AppStoreApp)
	require.NotNil(t, sw[1].SoftwarePackage)
	require.Equal(t, sw[2].Name, "vpp1")
	require.NotNil(t, sw[2].AppStoreApp)
	require.Nil(t, sw[2].SoftwarePackage)

	// move host to team 2
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{host.ID})
	require.NoError(t, err)
	host.TeamID = &team2.ID

	// listing the host's software (including available for install) should now
	// only list "a" and "file1" (because they are actually installed) and not
	// link them to the installer/VPP app. With and without available software
	// should result in the same rows (no available software in that new team).
	for _, b := range []bool{true, false} {
		opts.IncludeAvailableForInstall = b
		sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
		require.NoError(t, err)
		require.Len(t, sw, 2)
		require.EqualValues(t, 2, meta.TotalResults)
		require.Equal(t, sw[0].Name, "a")
		require.Nil(t, sw[0].AppStoreApp)
		require.Nil(t, sw[0].SoftwarePackage)
		require.Equal(t, sw[1].Name, "file1")
		require.Nil(t, sw[1].AppStoreApp)
		require.Nil(t, sw[1].SoftwarePackage)
	}
}

func testListHostSoftwareInstallThenDeleteInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "user1", "user1@example.com", false)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithPlatform("darwin"))
	nanoEnroll(t, ds, host, false)
	opts := fleet.HostSoftwareTitleListOptions{
		ListOptions:                fleet.ListOptions{PerPage: 10, IncludeMetadata: true, OrderKey: "name", TestSecondaryOrderKey: "source"},
		IncludeAvailableForInstall: true,
	}

	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{host.ID})
	require.NoError(t, err)
	host.TeamID = &team1.ID

	// add a single "externally-installed" software for that host
	software := []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// create a software installer for team 1
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerTm1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "hello",
		InstallerFile:   tfr1,
		StorageID:       "storage1",
		Filename:        "file1",
		Title:           "file1",
		Version:         "1.0",
		Source:          "apps",
		TeamID:          &team1.ID,
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// fail to install it on the host
	hostInstall1, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerTm1, false, nil)
	require.NoError(t, err)
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           hostInstall1,
		InstallScriptExitCode: ptr.Int(1),
	})
	require.NoError(t, err)

	// add a VPP app for team 1
	vppTm1, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: fleet.MacOSPlatform}}, Name: "vpp1",
			BundleIdentifier: "com.app.vpp1", LatestVersion: "1.0",
		}, &team1.ID)
	require.NoError(t, err)

	// install it on the host
	vpp1CmdUUID := createVPPAppInstallRequest(t, ds, host, vppTm1.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, host, vpp1CmdUUID, fleet.MDMAppleStatusAcknowledged)

	// add the successful VPP app to the reported installed software
	software = []fleet.Software{
		{Name: "a", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "vpp1", Version: "1.0", Source: "apps", BundleIdentifier: "com.app.vpp1"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// listing the host's software (including available for install) at this
	// point lists "a", "file1" and "vpp1" (because of the install attempt)
	sw, meta, err := ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Len(t, sw, 3)
	require.EqualValues(t, 3, meta.TotalResults)
	require.Equal(t, sw[0].Name, "a")
	require.Nil(t, sw[0].AppStoreApp)
	require.Nil(t, sw[0].SoftwarePackage)
	require.Equal(t, sw[1].Name, "file1")
	require.Nil(t, sw[1].AppStoreApp)
	require.NotNil(t, sw[1].SoftwarePackage)
	require.Equal(t, sw[2].Name, "vpp1")
	require.NotNil(t, sw[2].AppStoreApp)
	require.Nil(t, sw[2].SoftwarePackage)

	// delete both installers
	err = ds.DeleteSoftwareInstaller(ctx, installerTm1)
	require.NoError(t, err)
	err = ds.DeleteVPPAppFromTeam(ctx, &team1.ID, vppTm1.VPPAppID)
	require.NoError(t, err)

	// listing the host's software (including available for install) should now
	// only list "a" and "vpp1" (because they are actually installed) and not
	// link them to the installer/VPP app. With and without available software
	// should result in the same rows (no available software anymore).
	for _, b := range []bool{true, false} {
		opts.IncludeAvailableForInstall = b
		sw, meta, err = ds.ListHostSoftware(ctx, host, opts)
		require.NoError(t, err)
		require.Len(t, sw, 2)
		require.EqualValues(t, 2, meta.TotalResults)
		require.Equal(t, sw[0].Name, "a")
		require.Nil(t, sw[0].AppStoreApp)
		require.Nil(t, sw[0].SoftwarePackage)
		require.Equal(t, sw[1].Name, "vpp1")
		require.Nil(t, sw[1].AppStoreApp)
		require.Nil(t, sw[1].SoftwarePackage)
	}
}

func testListSoftwareVersionsVulnerabilityFilters(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())

	software := []fleet.Software{
		{Name: "chrome", Version: "0.0.1", Source: "apps"},
		{Name: "chrome", Version: "0.0.3", Source: "apps"},
		{Name: "safari", Version: "0.0.3", Source: "apps"},
		{Name: "safari", Version: "0.0.1", Source: "apps"},
		{Name: "firefox", Version: "0.0.3", Source: "apps"},
		{Name: "edge", Version: "0.0.3", Source: "apps"},
		{Name: "brave", Version: "0.0.3", Source: "apps"},
		{Name: "opera", Version: "0.0.3", Source: "apps"},
		{Name: "internet explorer", Version: "0.0.3", Source: "apps"},
		{Name: "netscape", Version: "0.0.3", Source: "apps"},
	}

	sw, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	var chrome001 uint
	var safari001 uint
	var firefox003 uint
	var edge003 uint
	var brave003 uint
	var opera003 uint
	var ie003 uint
	for s := range sw.Inserted {
		switch {
		case sw.Inserted[s].Name == "chrome" && sw.Inserted[s].Version == "0.0.1":
			chrome001 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "safari" && sw.Inserted[s].Version == "0.0.1":
			safari001 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "firefox" && sw.Inserted[s].Version == "0.0.3":
			firefox003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "edge" && sw.Inserted[s].Version == "0.0.3":
			edge003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "brave" && sw.Inserted[s].Version == "0.0.3":
			brave003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "opera" && sw.Inserted[s].Version == "0.0.3":
			opera003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "internet explorer" && sw.Inserted[s].Version == "0.0.3":
			ie003 = sw.Inserted[s].ID
		}
	}

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: chrome001,
		CVE:        "CVE-2024-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: safari001,
		CVE:        "CVE-2024-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: firefox003,
		CVE:        "CVE-2024-1236",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: edge003,
		CVE:        "CVE-2024-1237",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: brave003,
		CVE:        "CVE-2024-1238",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: opera003,
		CVE:        "CVE-2024-1239",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: ie003,
		CVE:        "CVE-2024-1240",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = ds.InsertCVEMeta(ctx, []fleet.CVEMeta{
		{
			// chrome
			CVE:              "CVE-2024-1234",
			CVSSScore:        ptr.Float64(7.5),
			CISAKnownExploit: ptr.Bool(true),
		},
		{
			// safari
			CVE:              "CVE-2024-1235",
			CVSSScore:        ptr.Float64(7.5),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			// firefox
			CVE:              "CVE-2024-1236",
			CVSSScore:        ptr.Float64(8.0),
			CISAKnownExploit: ptr.Bool(true),
		},
		{
			// edge
			CVE:              "CVE-2024-1237",
			CVSSScore:        ptr.Float64(8.0),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			// brave
			CVE:              "CVE-2024-1238",
			CVSSScore:        ptr.Float64(9.0),
			CISAKnownExploit: ptr.Bool(true),
		},
		// CVE-2024-1239 for opera has no CVE Meta
		{
			// internet explorer
			CVE:              "CVE-2024-1240",
			CVSSScore:        nil,
			CISAKnownExploit: nil,
		},
	})
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	type swVersion struct {
		Name    string
		Version string
	}

	tc := []struct {
		name     string
		opts     fleet.SoftwareListOptions
		expected []swVersion
		err      error
	}{
		{
			name: "vulnerable only",
			opts: fleet.SoftwareListOptions{
				ListOptions:    fleet.ListOptions{OrderKey: "name"},
				VulnerableOnly: true,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "edge",
					Version: "0.0.3",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
				{
					Name:    "internet explorer",
					Version: "0.0.3",
				},
				{
					Name:    "opera",
					Version: "0.0.3",
				},
				{
					Name:    "safari",
					Version: "0.0.1",
				},
			},
		},
		{
			name: "known exploit true",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				KnownExploit:     true,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "minimum cvss 8.0",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      8.0,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "edge",
					Version: "0.0.3",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "minimum cvss 7.9",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      7.9,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "edge",
					Version: "0.0.3",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "minimum cvss 8.0 and known exploit",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      8.0,
				KnownExploit:     true,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "minimum cvss 7.5 and known exploit",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      7.5,
				KnownExploit:     true,
			},
			expected: []swVersion{
				{
					Name:    "brave",
					Version: "0.0.3",
				},
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "maximum cvss 7.5",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MaximumCVSS:      7.5,
			},
			expected: []swVersion{
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "safari",
					Version: "0.0.1",
				},
			},
		},
		{
			name: "maximum cvss 7.6",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MaximumCVSS:      7.6,
			},
			expected: []swVersion{
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "safari",
					Version: "0.0.1",
				},
			},
		},
		{
			name: "maximum cvss 7.5 and known exploit",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MaximumCVSS:      7.5,
				KnownExploit:     true,
			},
			expected: []swVersion{
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
			},
		},
		{
			name: "minimum cvss 7.5 and maximum cvss 8.0",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      7.5,
				MaximumCVSS:      8.0,
			},
			expected: []swVersion{
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "edge",
					Version: "0.0.3",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
				{
					Name:    "safari",
					Version: "0.0.1",
				},
			},
		},
		{
			name: "minimum cvss 7.5 and maximum cvss 8.0 and known exploit",
			opts: fleet.SoftwareListOptions{
				ListOptions:      fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending},
				IncludeCVEScores: true,
				VulnerableOnly:   true,
				MinimumCVSS:      7.5,
				MaximumCVSS:      8.0,
				KnownExploit:     true,
			},
			expected: []swVersion{
				{
					Name:    "chrome",
					Version: "0.0.1",
				},
				{
					Name:    "firefox",
					Version: "0.0.3",
				},
			},
		},
		{
			name: "err if vulnerableOnly is not set with MinimumCVSS",
			opts: fleet.SoftwareListOptions{
				ListOptions: fleet.ListOptions{},
				MinimumCVSS: 7.5,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
		{
			name: "err if vulnerableOnly is not set with MaximumCVSS",
			opts: fleet.SoftwareListOptions{
				ListOptions: fleet.ListOptions{},
				MaximumCVSS: 7.5,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
		{
			name: "err if vulnerableOnly is not set with KnownExploit",
			opts: fleet.SoftwareListOptions{
				ListOptions:  fleet.ListOptions{},
				KnownExploit: true,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			sw, _, err := ds.ListSoftware(ctx, tt.opts)
			if tt.err != nil {
				require.Error(t, err)
				require.Equal(t, tt.err, err)
				return
			}
			require.Len(t, sw, len(tt.expected))
			for i, s := range sw {
				require.Equal(t, tt.expected[i].Name, s.Name)
				require.Equal(t, tt.expected[i].Version, s.Version)
			}
			count, err := ds.CountSoftware(ctx, tt.opts)
			require.NoError(t, err)
			require.Equal(t, len(tt.expected), count)
		})
	}
}

func testListHostSoftwareWithLabelScoping(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a host
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now(), test.WithPlatform("darwin"))
	nanoEnroll(t, ds, host, false)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	time.Sleep(time.Second) // ensure the labels_updated_at timestamp is before labels creation

	// create a software installer
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installer1 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		BundleIdentifier:  "bi1",
		Platform:          "darwin",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	}
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, installer1)
	require.NoError(t, err)

	// we should see installer1, since it has no label associated yet
	opts := fleet.HostSoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			PerPage:               11,
			IncludeMetadata:       true,
			OrderKey:              "name",
			TestSecondaryOrderKey: "source",
		},
		IncludeAvailableForInstall: true,
	}
	expectedInstallers := map[string]*fleet.SoftwarePackageOrApp{
		installer1.Filename: {
			Name:        installer1.Filename,
			Version:     installer1.Version,
			SelfService: ptr.Bool(false),
		},
	}

	checkSoftware := func(swList []*fleet.HostSoftwareWithInstaller, excludeNames ...string) {
		for _, got := range swList {
			want, ok := expectedInstallers[got.SoftwarePackage.Name]
			if slices.Contains(excludeNames, got.SoftwarePackage.Name) {
				require.False(t, ok)
				continue
			}
			require.True(t, ok)
			require.Equal(t, want, got.SoftwarePackage)
		}
	}

	software, _, err := ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software)

	// installer1 should be in scope since it has no labels
	scoped, err := ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	hostsInScope, err := ds.GetIncludedHostIDMapForSoftwareInstaller(ctx, installerID1)
	require.NoError(t, err)
	require.Contains(t, hostsInScope, host.ID)

	label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1" + t.Name()})
	require.NoError(t, err)

	// assign the label to the host
	require.NoError(t, ds.AddLabelsToHost(ctx, host.ID, []uint{label1.ID}))
	host.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, host)
	require.NoError(t, err)
	time.Sleep(time.Second)

	// assign the label to the software installer
	err = setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID1, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeExcludeAny,
		ByName:     map[string]fleet.LabelIdent{label1.Name: {LabelName: label1.Name, LabelID: label1.ID}},
	})
	require.NoError(t, err)

	// should be empty as the installer label is "exclude any"
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Empty(t, software)

	hostsNotInScope, err := ds.GetExcludedHostIDMapForSoftwareInstaller(ctx, installerID1)
	require.NoError(t, err)
	require.Contains(t, hostsNotInScope, host.ID)

	// installer1 should be out of scope since the label is "exclude any"
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.False(t, scoped)

	// Update the label to be "include any"
	err = setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID1, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeIncludeAny,
		ByName:     map[string]fleet.LabelIdent{label1.Name: {LabelName: label1.Name, LabelID: label1.ID}},
	})
	require.NoError(t, err)

	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software)

	// Now installer1 is in scope again: label is "include any"
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// Add an installer. No label yet.
	installer2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage2",
		Filename:          "file2",
		Title:             "file2",
		Version:           "2.0",
		Source:            "apps",
		UserID:            user1.ID,
		BundleIdentifier:  "bi2",
		Platform:          "darwin",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	}
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, installer2)
	require.NoError(t, err)

	expectedInstallers[installer2.Filename] = &fleet.SoftwarePackageOrApp{
		Name:        installer2.Filename,
		Version:     installer2.Version,
		SelfService: ptr.Bool(false),
	}

	// There's 2 installers now: installerID1 and installerID2 (because it has no labels associated)
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software)

	// Add "exclude any" labels to installer2
	label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2" + t.Name()})
	require.NoError(t, err)

	label3, err := ds.NewLabel(ctx, &fleet.Label{Name: "label3" + t.Name()})
	require.NoError(t, err)

	err = setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID2, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeExcludeAny,
		ByName: map[string]fleet.LabelIdent{
			label2.Name: {LabelName: label2.Name, LabelID: label2.ID},
			label3.Name: {LabelName: label3.Name, LabelID: label3.ID},
		},
	})
	require.NoError(t, err)

	// Now host has label1, label2
	require.NoError(t, ds.AddLabelsToHost(ctx, host.ID, []uint{label2.ID}))
	host.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, host)
	require.NoError(t, err)
	time.Sleep(time.Second)

	// List should be back to just installer1
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software, installer2.Filename)

	// installer1 is still in scope
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// installer2 is out of scope, because host has label2
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID2, host.ID)
	require.NoError(t, err)
	require.False(t, scoped)

	// Add an installer. No label yet.
	installer3 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		UserID:            user1.ID,
		BundleIdentifier:  "bi3",
		Platform:          "darwin",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	}
	installerID3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, installer3)
	require.NoError(t, err)

	time.Sleep(time.Second)
	expectedInstallers[installer3.Filename] = &fleet.SoftwarePackageOrApp{
		Name:        installer3.Filename,
		Version:     installer3.Version,
		SelfService: ptr.Bool(false),
	}

	// Add a new label and apply it to the installer. There are no hosts with this label.
	label4, err := ds.NewLabel(ctx, &fleet.Label{Name: "label4" + t.Name()})
	require.NoError(t, err)

	err = setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID3, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeExcludeAny,
		ByName:     map[string]fleet.LabelIdent{label4.Name: {LabelName: label4.Name, LabelID: label4.ID}},
	})
	require.NoError(t, err)

	// We should have [installerID1, installerID3], but the exclude any label has
	// no results for this host yet, so it's just installerID1 for now.
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	require.Len(t, software, 1)

	// installer1 is still in scope
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// installer3 is not in scope yet, because label is "exclude any" and host doesn't have results
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID3, host.ID)
	require.NoError(t, err)
	require.False(t, scoped)

	// mark as if label had been reported (but host is still not a member)
	host.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, host)
	require.NoError(t, err)
	time.Sleep(time.Second)

	// now has 2 software (installer1 and 3)
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software, installer2.Filename)

	// installer1 is still in scope
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// installer3 is in scope, because label is "exclude any" and host doesn't have the label
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID3, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// Now include hosts with label4. No host has this label, so we shouldn't see installerID3 anymore.
	err = setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID3, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeIncludeAny,
		ByName:     map[string]fleet.LabelIdent{label4.Name: {LabelName: label4.Name, LabelID: label4.ID}},
	})
	require.NoError(t, err)

	// We should have [installerID1]
	software, _, err = ds.ListHostSoftware(ctx, host, opts)
	require.NoError(t, err)
	checkSoftware(software, installer2.Filename, installer3.Filename)

	// installer1 is still in scope
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID1, host.ID)
	require.NoError(t, err)
	require.True(t, scoped)

	// installer3 is not in scope
	scoped, err = ds.IsSoftwareInstallerLabelScoped(ctx, installerID3, host.ID)
	require.NoError(t, err)
	require.False(t, scoped)
}
