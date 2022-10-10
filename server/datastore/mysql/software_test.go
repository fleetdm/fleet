package mysql

import (
	"context"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"HostsBySoftwareIDs", testHostsBySoftwareIDs},
		{"UpdateHostSoftware", testUpdateHostSoftware},
		{"ListSoftwareByHostIDShort", testListSoftwareByHostIDShort},
		{"ListSoftwareVulnerabilities", testListSoftwareVulnerabilities},
		{"InsertVulnerabilities", testInsertVulnerabilities},
		{"ListCVEs", testListCVEs},
		{"ListSoftwareForVulnDetection", testListSoftwareForVulnDetection},
		{"SoftwareByID", testSoftwareByID},
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

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	soft1ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID, false)
	require.NoError(t, err)
	require.NotNil(t, soft1ByID)
	assert.Equal(t, host1.HostSoftware.Software[0], *soft1ByID)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}
	software2 = []fleet.Software{}

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: "com.zoo"}, // "empty" -> "non-empty"
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.other"}, // "non-empty" -> "non-empty"
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: ""},               // non-empty -> empty
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)
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

	err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)

	err = ds.UpdateHostSoftware(context.Background(), host1.ID, software2)
	require.NoError(t, err)

	iterator, err := ds.AllSoftwareWithoutCPEIterator(context.Background(), oval.SupportedSoftwareSources)
	defer iterator.Close()
	require.NoError(t, err)

	loops := 0
	id := uint(0)
	for iterator.Next() {
		software, err := iterator.Value()
		require.NoError(t, err)
		require.NoError(t, iterator.Err())

		require.NotEmpty(t, software.ID)
		id = software.ID

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

	err = ds.AddCPEForSoftware(context.Background(), fleet.Software{ID: id}, "some:cpe")
	require.NoError(t, err)

	iterator, err = ds.AllSoftwareWithoutCPEIterator(context.Background(), oval.SupportedSoftwareSources)
	defer iterator.Close()
	require.NoError(t, err)

	loops = 0
	for iterator.Next() {
		software, err := iterator.Value()
		require.NoError(t, err)
		require.NoError(t, iterator.Err())

		require.NotEmpty(t, software.ID)
		require.NotEqual(t, id, software.ID)

		require.NotEmpty(t, software.Name)
		require.NotEmpty(t, software.Version)
		require.NotEmpty(t, software.Source)

		if loops > 1 {
			t.Error("Looping through more software than we have")
		}
		loops++
	}
	assert.Equal(t, len(software1)-1, loops)
	require.NoError(t, iterator.Close())
}

func testSoftwareHostDuplicates(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	longName := strings.Repeat("a", 260)

	incoming := make(map[string]fleet.Software)
	sw := fleet.Software{
		Name:    longName + "b",
		Version: "0.0.1",
		Source:  "chrome_extension",
	}
	soft2Key := softwareToUniqueString(sw)
	incoming[soft2Key] = sw

	tx, err := ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]fleet.Software), incoming))
	require.NoError(t, tx.Commit())

	incoming = make(map[string]fleet.Software)
	sw = fleet.Software{
		Name:    longName + "c",
		Version: "0.0.1",
		Source:  "chrome_extension",
	}
	soft3Key := softwareToUniqueString(sw)
	incoming[soft3Key] = sw

	tx, err = ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]fleet.Software), incoming))
	require.NoError(t, tx.Commit())
}

func testSoftwareLoadVulnerabilities(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2022-0001"},
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2022-0002"},
	}
	_, err := ds.InsertVulnerabilities(context.Background(), vulns, fleet.NVDSource)
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	softByID, err := ds.SoftwareByID(context.Background(), host.HostSoftware.Software[0].ID, false)
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
	ds.UpdateHost(ctx, debian)

	ubuntu := test.NewHost(t, ds, "host4", "", "host4key", "host4uuid", time.Now())
	ubuntu.Platform = "ubuntu"
	ds.UpdateHost(ctx, ubuntu)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "biz", Version: "0.0.1", Source: "deb_packages"},
		{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
	}
	require.NoError(t, ds.UpdateHostSoftware(ctx, debian.ID, software[:2]))
	require.NoError(t, ds.LoadHostSoftware(ctx, debian, false))

	require.NoError(t, ds.UpdateHostSoftware(ctx, ubuntu.ID, software[2:]))
	require.NoError(t, ds.LoadHostSoftware(ctx, ubuntu, false))

	require.NoError(t, ds.AddCPEForSoftware(ctx, debian.Software[0], "cpe1"))
	require.NoError(t, ds.AddCPEForSoftware(ctx, debian.Software[1], "cpe2"))

	require.NoError(t, ds.AddCPEForSoftware(ctx, ubuntu.Software[0], "cpe3"))
	require.NoError(t, ds.AddCPEForSoftware(ctx, ubuntu.Software[1], "cpe4"))

	cpes, err := ds.ListSoftwareCPEs(ctx)
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
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

	sort.Slice(host.Software, func(i, j int) bool { return host.Software[i].Name < host.Software[j].Name })

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))

	_, err := addCPEForSoftwareDB(context.Background(), ds.writer, host.Software[0], "somecpe")
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
	_, err = ds.writer.ExecContext(context.Background(), query, args...)
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

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host3.ID, software3))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host3, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[1], "someothercpewithoutvulns"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host3.Software[0], "somecpe2"))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host3, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})

	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: host1.Software[0].ID, CVE: "CVE-2022-0001"},
		{SoftwareID: host1.Software[0].ID, CVE: "CVE-2022-0002"},
		{SoftwareID: host3.Software[0].ID, CVE: "CVE-2022-0003"},
	}

	_, err := ds.InsertVulnerabilities(context.Background(), vulns, fleet.NVDSource)
	require.NoError(t, err)

	cveMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2022-0001",
			CVSSScore:        ptr.Float64(2.0),
			EPSSProbability:  ptr.Float64(0.01),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			CVE:              "CVE-2022-0002",
			CVSSScore:        ptr.Float64(1.0),
			EPSSProbability:  ptr.Float64(0.99),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			CVE:              "CVE-2022-0003",
			CVSSScore:        ptr.Float64(3.0),
			EPSSProbability:  ptr.Float64(0.98),
			CISAKnownExploit: ptr.Bool(true),
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
				CVE:              "CVE-2022-0001",
				DetailsLink:      "https://nvd.nist.gov/vuln/detail/CVE-2022-0001",
				CVSSScore:        ptr.Float64Ptr(2.0),
				EPSSProbability:  ptr.Float64Ptr(0.01),
				CISAKnownExploit: ptr.BoolPtr(false),
			},
			{
				CVE:              "CVE-2022-0002",
				DetailsLink:      "https://nvd.nist.gov/vuln/detail/CVE-2022-0002",
				CVSSScore:        ptr.Float64Ptr(1.0),
				EPSSProbability:  ptr.Float64Ptr(0.99),
				CISAKnownExploit: ptr.BoolPtr(false),
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
				CVE:              "CVE-2022-0003",
				DetailsLink:      "https://nvd.nist.gov/vuln/detail/CVE-2022-0003",
				CVSSScore:        ptr.Float64Ptr(3.0),
				EPSSProbability:  ptr.Float64Ptr(0.98),
				CISAKnownExploit: ptr.BoolPtr(true),
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
				Page:     1,
				PerPage:  1,
				OrderKey: "version",
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
				PerPage:  1,
				Page:     1,
				OrderKey: "id",
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

		opts.MatchQuery = "CVE-2022-0002"
		software = listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// partial CVE
		opts.MatchQuery = "0002"
		software = listSoftwareCheckCount(t, ds, 1, 1, opts, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// unknown CVE
		opts.MatchQuery = "CVE-2022-0000"
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
		opts.MatchQuery = "0.0.3"
		software = listSoftwareCheckCount(t, ds, 2, 2, opts, true)
		expected = []fleet.Software{foo003, bar003}
		test.ElementsMatchSkipID(t, software, expected)

		// query by version (case insensitive)
		opts.MatchQuery = "V0.0.2"
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
	software, err := ds.ListSoftware(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, software, expectedListCount)
	count, err := ds.CountSoftware(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, expectedFullCount, count)
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
		err := ds.writer.Get(&tableCount, "SELECT COUNT(*) FROM software_host_counts")
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

	require.NoError(t, ds.UpdateHostSoftware(ctx, host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(ctx, host2.ID, software2))

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
	require.NoError(t, ds.UpdateHostSoftware(ctx, host2.ID, software2))
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
	_, err := ds.writer.ExecContext(ctx, `INSERT INTO software (name, version, source) VALUES ('baz', '0.0.1', 'testing')`)
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
	require.NoError(t, ds.UpdateHostSoftware(ctx, host3.ID, software3))
	require.NoError(t, ds.UpdateHostSoftware(ctx, host4.ID, software4))

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
	require.NoError(t, ds.UpdateHostSoftware(ctx, host4.ID, software4))
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
	require.NoError(t, ds.UpdateHostSoftware(ctx, host4.ID, software4))
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

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[0], "cpe_foo_chrome_3"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[1], "cpe_foo_rpm"))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[0], "cpe_bar_rpm"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[1], "cpe_foo_chrome_2"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[2], "cpe_foo_chrome_3"))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))
	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	chrome3 := host2.Software[2]
	n, err := ds.InsertVulnerabilities(context.Background(), []fleet.SoftwareVulnerability{
		{
			SoftwareID: chrome3.ID,
			CVE:        "CVE-2022-0001",
		},
	}, fleet.NVDSource)

	require.NoError(t, err)
	require.Equal(t, 1, int(n))

	barRpm := host2.Software[0]
	n, err = ds.InsertVulnerabilities(context.Background(),
		[]fleet.SoftwareVulnerability{
			{
				SoftwareID: barRpm.ID,
				CVE:        "CVE-2022-0002",
			},
			{
				SoftwareID: barRpm.ID,
				CVE:        "CVE-2022-0003",
			},
		}, fleet.NVDSource)

	require.NoError(t, err)
	require.Equal(t, 2, int(n))

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
	require.ElementsMatch(t, hosts, []*fleet.HostShort{
		{
			ID:          1,
			Hostname:    "host1",
			DisplayName: "computer1",
		}, {
			ID:          2,
			Hostname:    "host2",
			DisplayName: "host2",
		},
	})

	// CVE of bar.rpm 0.0.3, only host 2 has it
	hosts, err = ds.HostsByCVE(ctx, "CVE-2022-0002")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hosts[0].Hostname, "host2")
}

func testHostsBySoftwareIDs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hosts, err := ds.HostsBySoftwareIDs(ctx, []uint{0})
	require.NoError(t, err)
	require.Len(t, hosts, 0)

	insertVulnSoftwareForTest(t, ds)

	allSoftware, err := ds.ListSoftware(ctx, fleet.SoftwareListOptions{})
	require.NoError(t, err)

	var chrome3 fleet.Software
	var barRpm fleet.Software

	for _, s := range allSoftware {
		if s.GenerateCPE == "cpe_foo_chrome_3" {
			chrome3 = s
		}

		if s.GenerateCPE == "cpe_bar_rpm" {
			barRpm = s
		}
	}

	require.NotZero(t, chrome3.ID)
	require.NotZero(t, barRpm.ID)

	hosts, err = ds.HostsBySoftwareIDs(ctx, []uint{chrome3.ID})
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.ElementsMatch(t, hosts, []*fleet.HostShort{
		{
			ID:          1,
			Hostname:    "host1",
			DisplayName: "computer1",
		}, {
			ID:          2,
			Hostname:    "host2",
			DisplayName: "host2",
		}})

	hosts, err = ds.HostsBySoftwareIDs(ctx, []uint{barRpm.ID})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hosts[0].Hostname, "host2")

	// Duplicates should not be returned if cpes are found on the same host ie host2 should only appear once
	hosts, err = ds.HostsBySoftwareIDs(ctx, []uint{chrome3.ID, barRpm.ID})
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.Equal(t, hosts[0].Hostname, "host1")
	require.Equal(t, hosts[1].Hostname, "host2")
}

func testUpdateHostSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)

	// sort software slice by last opened at timestamp
	genSortFn := func(sl []fleet.Software) func(l, r int) bool {
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
	err := ds.UpdateHostSoftware(ctx, host.ID, sw)
	require.NoError(t, err)
	validateSoftware(tup{name: "foo"}, tup{"bar", lastYear}, tup{"baz", now})

	// make changes: remove foo, add qux, no new timestamp on bar, small ts change on baz
	nowish := now.Add(3 * time.Second)
	sw = []fleet.Software{
		{Name: "bar", Version: "0.0.2", Source: "test", GenerateCPE: "cpe_bar"},
		{Name: "baz", Version: "0.0.3", Source: "test", GenerateCPE: "cpe_baz", LastOpenedAt: &nowish},
		{Name: "qux", Version: "0.0.4", Source: "test", GenerateCPE: "cpe_qux"},
	}
	err = ds.UpdateHostSoftware(ctx, host.ID, sw)
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
	err = ds.UpdateHostSoftware(ctx, host.ID, sw)
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

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
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

func testListSoftwareVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	require.NoError(t, ds.UpdateHostSoftware(ctx, host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[0], "foo_cpe"))
	require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[1], "bar_cpe"))
	require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[2], "blah_cpe"))
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	cveMap := map[int]string{
		0: "cve-123",
		1: "cve-456",
	}

	var vulns []fleet.SoftwareVulnerability
	for i, s := range host.Software {
		cve, ok := cveMap[i]
		if ok {
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        cve,
			})
		}

	}
	n, err := ds.InsertVulnerabilities(ctx, vulns, fleet.NVDSource)
	require.NoError(t, err)
	require.Equal(t, int(n), 2)

	expectedCVEs := []string{"cve-123", "cve-456"}

	actualCVEs := make([]string, 0)
	result, err := ds.ListSoftwareVulnerabilities(ctx, []uint{host.ID})
	for _, r := range result[host.ID] {
		actualCVEs = append(actualCVEs, r.CVE)
	}

	require.NoError(t, err)
	require.ElementsMatch(t, expectedCVEs, actualCVEs)

	for _, r := range result[host.ID] {
		require.NotEqual(t, r.SoftwareID, 0)
	}
}

func testInsertVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("no vulnerabilities to insert", func(t *testing.T) {
		r, err := ds.InsertVulnerabilities(ctx, nil, fleet.UbuntuOVALSource)
		require.Zero(t, r)
		require.NoError(t, err)
	})

	t.Run("duplicated vulnerabilities", func(t *testing.T) {
		host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
		software := fleet.Software{
			Name: "foo", Version: "0.0.1", Source: "chrome_extensions",
		}

		require.NoError(t, ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{software}))
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[0], "foo_cpe_1"))

		var vulns []fleet.SoftwareVulnerability
		for _, s := range host.Software {
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID, CVE: "cve-1",
			})
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID, CVE: "cve-1",
			})
		}

		n, err := ds.InsertVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Equal(t, 1, int(n))

		storedVulns, err := ds.ListSoftwareVulnerabilities(ctx, []uint{host.ID})
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

		require.NoError(t, ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{software}))
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[0], "foo_cpe_2"))

		var vulns []fleet.SoftwareVulnerability
		for _, s := range host.Software {
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        "cve-2",
			})
		}

		n, err := ds.InsertVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Equal(t, 1, int(n))

		n, err = ds.InsertVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Equal(t, 0, int(n))

		storedVulns, err := ds.ListSoftwareVulnerabilities(ctx, []uint{host.ID})
		require.NoError(t, err)

		occurrence := make(map[string]int)
		for _, v := range storedVulns[host.ID] {
			occurrence[v.CVE] = occurrence[v.CVE] + 1
		}
		require.Equal(t, 1, occurrence["cve-1"])
		require.Equal(t, 1, occurrence["cve-2"])
	})
}

func testListCVEs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	now := time.Now().UTC()
	threeDaysAgo := now.Add(-3 * 24 * time.Hour)
	twoWeeksAgo := now.Add(-14 * 24 * time.Hour)
	twoMonthsAgo := now.Add(-60 * 24 * time.Hour)

	testCases := []fleet.CVEMeta{
		{CVE: "cve-1", Published: &threeDaysAgo},
		{CVE: "cve-2", Published: &twoWeeksAgo},
		{CVE: "cve-3", Published: &twoMonthsAgo},
		{CVE: "cve-4"},
	}

	err := ds.InsertCVEMeta(ctx, testCases)
	require.NoError(t, err)

	result, err := ds.ListCVEs(ctx, 30*24*time.Hour)
	require.NoError(t, err)

	expected := []string{"cve-1", "cve-2"}
	var actual []string
	for _, r := range result {
		actual = append(actual, r.CVE)
	}

	require.ElementsMatch(t, expected, actual)
}

func testListSoftwareForVulnDetection(t *testing.T, ds *Datastore) {
	t.Run("returns software without CPE entries", func(t *testing.T) {
		ctx := context.Background()

		host := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
		host.Platform = "debian"
		ds.UpdateHost(ctx, host)

		software := []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
			{Name: "biz", Version: "0.0.1", Source: "deb_packages"},
			{Name: "baz", Version: "0.0.3", Source: "deb_packages"},
		}
		require.NoError(t, ds.UpdateHostSoftware(ctx, host.ID, software))
		require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
		require.NoError(t, ds.AddCPEForSoftware(ctx, host.Software[0], "cpe1"))
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

func testSoftwareByID(t *testing.T, ds *Datastore) {
	t.Run("software installed in multiple hosts does not have duplicated vulnerabilities", func(t *testing.T) {
		ctx := context.Background()

		hostA := test.NewHost(t, ds, "hostA", "", "hostAkey", "hostAuuid", time.Now())
		hostA.Platform = "ubuntu"
		ds.UpdateHost(ctx, hostA)

		hostB := test.NewHost(t, ds, "hostB", "", "hostBkey", "hostBuuid", time.Now())
		hostB.Platform = "ubuntu"
		ds.UpdateHost(ctx, hostB)

		software := []fleet.Software{
			{Name: "foo_123", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar_123", Version: "0.0.3", Source: "apps"},
			{Name: "biz_123", Version: "0.0.1", Source: "deb_packages"},
			{Name: "baz_123", Version: "0.0.3", Source: "deb_packages"},
		}

		require.NoError(t, ds.UpdateHostSoftware(ctx, hostA.ID, software))
		require.NoError(t, ds.UpdateHostSoftware(ctx, hostB.ID, software))

		require.NoError(t, ds.LoadHostSoftware(ctx, hostA, false))
		require.NoError(t, ds.LoadHostSoftware(ctx, hostB, false))

		// Add one vulnerability to each software
		var vulns []fleet.SoftwareVulnerability
		for i, s := range hostA.Software {
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID: s.ID,
				CVE:        fmt.Sprintf("cve-%d", i),
			})
		}
		n, err := ds.InsertVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Equal(t, 4, int(n))

		for _, s := range hostA.Software {
			result, err := ds.SoftwareByID(ctx, s.ID, true)
			require.NoError(t, err)
			require.Len(t, result.Vulnerabilities, 1)
		}
	})
}
