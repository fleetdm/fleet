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
	"github.com/jmoiron/sqlx"
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
		{"InsertCVEs", testSoftwareInsertCVEs},
		{"HostDuplicates", testSoftwareHostDuplicates},
		{"LoadVulnerabilities", testSoftwareLoadVulnerabilities},
		{"AllCPEs", testSoftwareAllCPEs},
		{"NothingChanged", testSoftwareNothingChanged},
		{"LoadSupportsTonsOfCVEs", testSoftwareLoadSupportsTonsOfCVEs},
		{"List", testSoftwareList},
		{"CalculateHostsPerSoftware", testSoftwareCalculateHostsPerSoftware},
		{"ListVulnerableSoftwareBySource", testListVulnerableSoftwareBySource},
		{"DeleteVulnerabilitiesByCPECVE", testDeleteVulnerabilitiesByCPECVE},
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

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	soft1ByID, err := ds.SoftwareByID(context.Background(), host1.HostSoftware.Software[0].ID)
	require.NoError(t, err)
	require.NotNil(t, soft1ByID)
	assert.Equal(t, host1.HostSoftware.Software[0], *soft1ByID)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}
	software2 = []fleet.Software{}

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software1 = []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "towel", Version: "42.0.0", Source: "apps"},
	}

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software1, host1.HostSoftware.Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: "com.zoo"}, // "empty" -> "non-empty"
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)

	software2 = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.other"}, // "non-empty" -> "non-empty"
		{Name: "zoo", Version: "0.0.5", Source: "deb_packages", BundleIdentifier: ""},               // non-empty -> empty
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipIDAndHostCount(t, software2, host2.HostSoftware.Software)
}

func testSoftwareCPE(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}

	err := ds.UpdateHostSoftware(context.Background(), host1.ID, software1)
	require.NoError(t, err)

	iterator, err := ds.AllSoftwareWithoutCPEIterator(context.Background())
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

		if loops > 2 {
			t.Error("Looping through more software than we have")
		}
		loops++
	}
	assert.Equal(t, len(software1), loops)
	require.NoError(t, iterator.Close())

	err = ds.AddCPEForSoftware(context.Background(), fleet.Software{ID: id}, "some:cpe")
	require.NoError(t, err)

	iterator, err = ds.AllSoftwareWithoutCPEIterator(context.Background())
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

func testSoftwareInsertCVEs(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "deb_packages", Release: "1"},
		{Name: "foo", Version: "0.0.1", Source: "deb_packages", Release: "2"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "somecpe"))
	count, err := ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"})
	require.NoError(t, err)
	// inserts one per release
	assert.Equal(t, int64(2), count)

	// run again for the same CPE, should not create any new row
	count, err = ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func testSoftwareHostDuplicates(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	longName := strings.Repeat("a", 260)

	incoming := make(map[string]struct{})
	soft2Key := softwareToUniqueString(fleet.Software{
		Name:    longName + "b",
		Version: "0.0.1",
		Source:  "chrome_extension",
	})
	incoming[soft2Key] = struct{}{}

	tx, err := ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]uint), incoming))
	require.NoError(t, tx.Commit())

	incoming = make(map[string]struct{})
	soft3Key := softwareToUniqueString(fleet.Software{
		Name:    longName + "c",
		Version: "0.0.1",
		Source:  "chrome_extension",
	})
	incoming[soft3Key] = struct{}{}

	tx, err = ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]uint), incoming))
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
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))
	_, err := ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"})
	require.NoError(t, err)
	_, err = ds.InsertCVEForCPE(context.Background(), "cve-321-321-321", []string{"somecpe"})
	require.NoError(t, err)

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	softByID, err := ds.SoftwareByID(context.Background(), host.HostSoftware.Software[0].ID)
	require.NoError(t, err)
	require.NotNil(t, softByID)
	require.Len(t, softByID.Vulnerabilities, 2)

	assert.Equal(t, "somecpe", host.Software[0].GenerateCPE)
	require.Len(t, host.Software[0].Vulnerabilities, 2)
	assert.Equal(t, "cve-123-123-132", host.Software[0].Vulnerabilities[0].CVE)
	assert.Equal(t,
		"https://nvd.nist.gov/vuln/detail/cve-123-123-132", host.Software[0].Vulnerabilities[0].DetailsLink)
	assert.Equal(t, "cve-321-321-321", host.Software[0].Vulnerabilities[1].CVE)
	assert.Equal(t,
		"https://nvd.nist.gov/vuln/detail/cve-321-321-321", host.Software[0].Vulnerabilities[1].DetailsLink)

	assert.Equal(t, "someothercpewithoutvulns", host.Software[1].GenerateCPE)
	require.Len(t, host.Software[1].Vulnerabilities, 0)
}

func testSoftwareAllCPEs(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))

	cpes, err := ds.AllCPEs(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, cpes, []string{"somecpe", "someothercpewithoutvulns"})
}

func testSoftwareNothingChanged(t *testing.T, ds *Datastore) {
	assert.False(t, nothingChanged(nil, []fleet.Software{{}}))
	assert.True(t, nothingChanged(nil, nil))
	assert.True(t, nothingChanged(
		[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
		[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
	))
	assert.False(t, nothingChanged(
		[]fleet.Software{{Name: "A", Version: "1.1", Source: "ASD"}},
		[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
	))
	assert.False(t, nothingChanged(
		[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}},
		[]fleet.Software{{Name: "A", Version: "1.0", Source: "ASD"}, {Name: "B", Version: "1.0", Source: "ASD"}},
	))
}

func testSoftwareLoadSupportsTonsOfCVEs(t *testing.T, ds *Datastore) {
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "blah", Version: "1.0", Source: "apps"},
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	sort.Slice(host.Software, func(i, j int) bool { return host.Software[i].Name < host.Software[j].Name })

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))

	require.NoError(t, ds.withTx(context.Background(), func(tx sqlx.ExtContext) error {
		somecpeID, err := addCPEForSoftwareDB(context.Background(), tx, host.Software[0], "somecpe")
		if err != nil {
			return err
		}
		sql := `INSERT IGNORE INTO software_cve (cpe_id, cve) VALUES (?, ?)`
		for i := 0; i < 1000; i++ {
			part1 := rand.Intn(1000)
			part2 := rand.Intn(1000)
			part3 := rand.Intn(1000)
			cve := fmt.Sprintf("cve-%d-%d-%d", part1, part2, part3)
			if _, err := tx.ExecContext(context.Background(), sql, somecpeID, cve); err != nil {
				return err
			}
		}
		return nil
	}))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	for _, software := range host.Software {
		switch software.Name {
		case "bar":
			assert.Equal(t, "somecpe", software.GenerateCPE)
			require.Len(t, software.Vulnerabilities, 1000)
			assert.True(t, strings.HasPrefix(software.Vulnerabilities[0].CVE, "cve-"))
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
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))

	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[1], "someothercpewithoutvulns"))
	_, err := ds.InsertCVEForCPE(context.Background(), "cve-321-432-543", []string{"somecpe"})
	require.NoError(t, err)
	_, err = ds.InsertCVEForCPE(context.Background(), "cve-333-444-555", []string{"somecpe"})
	require.NoError(t, err)

	foo001 := fleet.Software{
		Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "somecpe",
		Vulnerabilities: fleet.VulnerabilitiesSlice{
			{CVE: "cve-321-432-543", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-321-432-543"},
			{CVE: "cve-333-444-555", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-333-444-555"},
		},
	}
	foo002 := fleet.Software{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"}
	foo003 := fleet.Software{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "someothercpewithoutvulns"}
	bar003 := fleet.Software{Name: "bar", Version: "0.0.3", Source: "deb_packages"}

	t.Run("lists everything", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 4, 4, fleet.SoftwareListOptions{}, true)
		expected := []fleet.Software{bar003, foo001, foo003, foo002}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("limits the results", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 1, 4, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{PerPage: 1, OrderKey: "version"}}, true)
		expected := []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("paginates", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 1, 4, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{Page: 1, PerPage: 1, OrderKey: "version"}}, true)
		require.Len(t, software, 1)
		var expected []fleet.Software
		// Both foo003 and bar003 have the same version, thus we check which one the database picked
		// for the second page.
		if software[0].Name == "foo" {
			expected = []fleet.Software{foo003}
		} else {
			expected = []fleet.Software{bar003}
		}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		software := listSoftwareCheckCount(t, ds, 2, 2, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{OrderKey: "version"}, TeamID: &team1.ID}, true)
		expected := []fleet.Software{foo001, foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team and paginates", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1-" + t.Name()})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		software := listSoftwareCheckCount(t, ds, 1, 2, fleet.SoftwareListOptions{
			ListOptions: fleet.ListOptions{
				PerPage:  1,
				Page:     1,
				OrderKey: "id",
			},
			TeamID: &team1.ID,
		}, true)
		expected := []fleet.Software{foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters vulnerable software", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{VulnerableOnly: true}, true)
		expected := []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters specific cves", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "cve-321-432-543"}}, true)
		expected := []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		software = listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "cve-333-444-555"}}, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// partial cve
		software = listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "333-444"}}, true)
		expected = []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)

		// unknown CVE
		listSoftwareCheckCount(t, ds, 0, 0, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "cve-000-000-000"}}, true)
	})

	t.Run("filters by query", func(t *testing.T) {
		// query by name (case insensitive)
		software := listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "baR"}}, true)
		expected := []fleet.Software{bar003}
		test.ElementsMatchSkipID(t, software, expected)
		// query by version
		software = listSoftwareCheckCount(t, ds, 2, 2, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "0.0.3"}}, true)
		expected = []fleet.Software{foo003, bar003}
		test.ElementsMatchSkipID(t, software, expected)
		// query by version (case insensitive)
		software = listSoftwareCheckCount(t, ds, 1, 1, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{MatchQuery: "V0.0.2"}}, true)
		expected = []fleet.Software{foo002}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("can order by name and id", func(t *testing.T) {
		software := listSoftwareCheckCount(t, ds, 4, 4, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{OrderKey: "name,id", OrderDirection: fleet.OrderAscending}}, false)
		assert.Equal(t, bar003.Name, software[0].Name)
		assert.Equal(t, bar003.Version, software[0].Version)
		assert.Equal(t, bar003.Source, software[0].Source)

		// it's ordered by id, descending
		assert.Greater(t, software[2].ID, software[1].ID)
		assert.Greater(t, software[3].ID, software[2].ID)
	})

	t.Run("hosts count", func(t *testing.T) {
		defer TruncateTables(t, ds, "software_host_counts")
		listSoftwareCheckCount(t, ds, 0, 0, fleet.SoftwareListOptions{WithHostCounts: true}, false)

		// create the counts for those software and re-run
		require.NoError(t, ds.CalculateHostsPerSoftware(context.Background(), time.Now()))
		software := listSoftwareCheckCount(t, ds, 4, 4, fleet.SoftwareListOptions{ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}, WithHostCounts: true}, false)
		// ordered by counts descending, so foo003 is first
		assert.Equal(t, foo003.Name, software[0].Name)
		assert.Equal(t, 2, software[0].HostsCount)
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

func testSoftwareCalculateHostsPerSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	cmpNameVersionCount := func(want, got []fleet.Software) {
		cmp := make([]fleet.Software, len(got))
		for i, sw := range got {
			cmp[i] = fleet.Software{Name: sw.Name, Version: sw.Version, HostsCount: sw.HostsCount}
		}
		require.ElementsMatch(t, want, cmp)
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

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	err := ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)

	globalOpts := fleet.SoftwareListOptions{WithHostCounts: true, ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	globalCounts := listSoftwareCheckCount(t, ds, 4, 4, globalOpts, false)

	want := []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
		{Name: "bar", Version: "0.0.3", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	// update host2, remove "bar" software
	software2 = []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))

	err = ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)

	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	// create a software entry without any host and any counts
	_, err = ds.writer.ExecContext(ctx, `INSERT INTO software (name, version, source) VALUES ('baz', '0.0.1', 'testing')`)
	require.NoError(t, err)

	// listing without the counts gets that new software entry
	allSw := listSoftwareCheckCount(t, ds, 4, 4, fleet.SoftwareListOptions{}, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 0},
		{Name: "foo", Version: "0.0.1", HostsCount: 0},
		{Name: "foo", Version: "v0.0.2", HostsCount: 0},
		{Name: "baz", Version: "0.0.1", HostsCount: 0},
	}
	cmpNameVersionCount(want, allSw)

	// after a call to Calculate, the unused software entry is removed
	err = ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)

	allSw = listSoftwareCheckCount(t, ds, 3, 3, fleet.SoftwareListOptions{}, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 0},
		{Name: "foo", Version: "0.0.1", HostsCount: 0},
		{Name: "foo", Version: "v0.0.2", HostsCount: 0},
	}
	cmpNameVersionCount(want, allSw)

	// create 2 teams and assign a new host to each
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host3.ID}))
	host4 := test.NewHost(t, ds, "host4", "", "host4key", "host4uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{host4.ID}))

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
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host3.ID, software3))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host4.ID, software4))

	// at this point, there's no counts per team, only global counts
	globalCounts = listSoftwareCheckCount(t, ds, 3, 3, globalOpts, false)
	want = []fleet.Software{
		{Name: "foo", Version: "0.0.3", HostsCount: 2},
		{Name: "foo", Version: "0.0.1", HostsCount: 1},
		{Name: "foo", Version: "v0.0.2", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Opts := fleet.SoftwareListOptions{WithHostCounts: true, TeamID: ptr.Uint(team1.ID), ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	team1Counts := listSoftwareCheckCount(t, ds, 0, 0, team1Opts, false)
	want = []fleet.Software{}
	cmpNameVersionCount(want, team1Counts)

	// after a call to Calculate, the global counts are updated and the team counts appear
	err = ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)

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
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host4.ID, software4))

	err = ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)

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
}

func dumpSoftwareHostCounts(t *testing.T, ds *Datastore) {
	type row struct {
		SoftwareID uint      `db:"software_id"`
		HostsCount int       `db:"hosts_count"`
		TeamID     *uint     `db:"team_id"`
		UpdatedAt  time.Time `db:"updated_at"`
	}
	var rows []row
	err := ds.writer.Select(&rows, "select software_id, hosts_count, team_id, updated_at from software_host_counts order by software_id")
	require.NoError(t, err)
	for _, r := range rows {
		t.Logf("software: %03d\thosts: %03d\tteam: %v\tts: %s", r.SoftwareID, r.HostsCount, r.TeamID, r.UpdatedAt.Format(time.RFC3339))
	}

	var soft []*fleet.Software
	err = ds.writer.Select(&soft, "select * from software order by id")
	require.NoError(t, err)
	for _, sw := range soft {
		t.Logf("id: %03d\tname: %s\tversion: %s\tbundle: %s\tsource: %s", sw.ID, sw.Name, sw.Version, sw.BundleIdentifier, sw.Source)
	}

	type hostSw struct {
		HostID     uint `db:"host_id"`
		SoftwareID uint `db:"software_id"`
	}
	var hostSws []*hostSw
	err = ds.writer.Select(&hostSws, "select * from host_software order by host_id")
	require.NoError(t, err)
	for _, hsw := range hostSws {
		t.Logf("host: %03d\tsoftware: %03d", hsw.HostID, hsw.SoftwareID)
	}

	var hosts []*fleet.Host
	err = ds.writer.Select(&hosts, "select id, hostname, team_id from hosts order by id")
	require.NoError(t, err)
	for _, h := range hosts {
		t.Logf("host: %03d\tname: %s\tteam: %v", h.ID, h.Hostname, h.TeamID)
	}
}

func insertVulnSoftwareForTest(t *testing.T, ds *Datastore) {
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{
			Name: "foo.rpm", Version: "0.0.1", Source: "rpm_packages", GenerateCPE: "cpe_foo_rpm",
		},
		{
			Name: "foo.chrome", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "cpe_foo_chrome",
		},
	}
	software2 := []fleet.Software{
		{
			Name: "foo.chrome", Version: "v0.0.2", Source: "chrome_extensions", GenerateCPE: "cpe_foo_chrome2",
		},
		{
			Name: "foo.chrome", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "cpe_foo_chrome_3",
			Vulnerabilities: fleet.VulnerabilitiesSlice{
				{CVE: "cve-123-456-789", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-123-456-789"},
			},
		},
		{
			Name: "bar.rpm", Version: "0.0.3", Source: "rpm_packages", GenerateCPE: "cpe_bar_rpm",
			Vulnerabilities: fleet.VulnerabilitiesSlice{
				{CVE: "cve-321-432-543", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-321-432-543"},
				{CVE: "cve-333-444-555", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-333-444-555"},
			},
		},
	}

	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software1))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))

	sort.Slice(host1.Software, func(i, j int) bool {
		return host1.Software[i].Name+host1.Software[i].Version < host1.Software[j].Name+host1.Software[j].Version
	})
	sort.Slice(host2.Software, func(i, j int) bool {
		return host2.Software[i].Name+host2.Software[i].Version < host2.Software[j].Name+host2.Software[j].Version
	})

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[0], "cpe_foo_chrome"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[1], "cpe_foo_rpm"))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[0], "cpe_bar_rpm"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[1], "cpe_foo_chrome_3"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host2.Software[2], "cpe_foo_chrome_2"))

	_, err := ds.InsertCVEForCPE(context.Background(), "cve-123-456-789", []string{"cpe_foo_chrome_3"})
	require.NoError(t, err)

	_, err = ds.InsertCVEForCPE(context.Background(), "cve-321-432-543", []string{"cpe_bar_rpm"})
	require.NoError(t, err)
	_, err = ds.InsertCVEForCPE(context.Background(), "cve-333-444-555", []string{"cpe_bar_rpm"})
	require.NoError(t, err)
}

func testListVulnerableSoftwareBySource(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	insertVulnSoftwareForTest(t, ds)

	vulnerable, err := ds.ListVulnerableSoftwareBySource(ctx, "apps")
	require.NoError(t, err)
	require.Empty(t, vulnerable)

	vulnerable, err = ds.ListVulnerableSoftwareBySource(ctx, "rpm_packages")
	require.NoError(t, err)
	require.Len(t, vulnerable, 1)
	require.Equal(t, vulnerable[0].Name, "bar.rpm")
	require.Len(t, vulnerable[0].Vulnerabilities, 2)
	sort.Slice(vulnerable[0].Vulnerabilities, func(i, j int) bool {
		return vulnerable[0].Vulnerabilities[i].CVE < vulnerable[0].Vulnerabilities[j].CVE
	})
	require.Equal(t, "cve-321-432-543", vulnerable[0].Vulnerabilities[0].CVE)
	require.Equal(t, "cve-333-444-555", vulnerable[0].Vulnerabilities[1].CVE)
}

func testDeleteVulnerabilitiesByCPECVE(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	err := ds.DeleteVulnerabilitiesByCPECVE(ctx, nil)
	require.NoError(t, err)

	insertVulnSoftwareForTest(t, ds)

	err = ds.DeleteVulnerabilitiesByCPECVE(ctx, []fleet.SoftwareVulnerability{
		{
			CPEID: 999, // unknown CPE
			CVE:   "cve-333-444-555",
		},
	})
	require.NoError(t, err)

	software, err := ds.ListVulnerableSoftwareBySource(ctx, "rpm_packages")
	require.NoError(t, err)

	require.Len(t, software, 1)
	barRPM := software[0]
	require.Len(t, barRPM.Vulnerabilities, 2)

	err = ds.DeleteVulnerabilitiesByCPECVE(ctx, []fleet.SoftwareVulnerability{
		{
			CPEID: barRPM.CPEID,
			CVE:   "unknown-cve",
		},
	})
	require.NoError(t, err)

	err = ds.DeleteVulnerabilitiesByCPECVE(ctx, []fleet.SoftwareVulnerability{
		{
			CPEID: barRPM.CPEID,
			CVE:   "cve-333-444-555",
		},
	})
	require.NoError(t, err)

	software, err = ds.ListVulnerableSoftwareBySource(ctx, "rpm_packages")
	require.NoError(t, err)
	require.Len(t, software, 1)
	barRPM = software[0]
	require.Len(t, barRPM.Vulnerabilities, 1)

	err = ds.DeleteVulnerabilitiesByCPECVE(ctx, []fleet.SoftwareVulnerability{
		{
			CPEID: barRPM.CPEID,
			CVE:   "cve-321-432-543",
		},
	})
	require.NoError(t, err)

	software, err = ds.ListVulnerableSoftwareBySource(ctx, "rpm_packages")
	require.NoError(t, err)
	require.Len(t, software, 0)

	software, err = ds.ListVulnerableSoftwareBySource(ctx, "chrome_extensions")
	require.NoError(t, err)
	require.Len(t, software, 1)
}
