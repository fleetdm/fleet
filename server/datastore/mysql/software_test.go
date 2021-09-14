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
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveHostSoftware(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	soft1 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host1.HostSoftware = soft1
	soft2 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		},
	}
	host2.HostSoftware = soft2

	err := ds.SaveHostSoftware(context.Background(), host1)
	require.NoError(t, err)
	err = ds.SaveHostSoftware(context.Background(), host2)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(context.Background(), host1)
	require.NoError(t, err)
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft1.Software, host1.HostSoftware.Software)

	err = ds.LoadHostSoftware(context.Background(), host2)
	require.NoError(t, err)
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft2.Software, host2.HostSoftware.Software)

	soft1 = fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
	}
	host1.HostSoftware = soft1
	soft2 = fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{},
	}
	host2.HostSoftware = soft2

	err = ds.SaveHostSoftware(context.Background(), host1)
	require.NoError(t, err)
	err = ds.SaveHostSoftware(context.Background(), host2)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(context.Background(), host1)
	require.NoError(t, err)
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft1.Software, host1.HostSoftware.Software)

	err = ds.LoadHostSoftware(context.Background(), host2)
	require.NoError(t, err)
	assert.False(t, host2.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft2.Software, host2.HostSoftware.Software)

	soft1 = fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
	}
	host1.HostSoftware = soft1

	err = ds.SaveHostSoftware(context.Background(), host1)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(context.Background(), host1)
	require.NoError(t, err)
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft1.Software, host1.HostSoftware.Software)
}

func TestSoftwareCPE(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	soft1 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host1.HostSoftware = soft1

	err := ds.SaveHostSoftware(context.Background(), host1)
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
	assert.Equal(t, len(host1.Software), loops)
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
	assert.Equal(t, len(host1.Software)-1, loops)
	require.NoError(t, iterator.Close())
}

func TestInsertCVEs(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host.HostSoftware = soft
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"}))
}

func TestHostSoftwareDuplicates(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	longName := strings.Repeat("a", 260)

	incoming := make(map[string]bool)
	soft2Key := softwareToUniqueString(fleet.Software{
		Name:    longName + "b",
		Version: "0.0.1",
		Source:  "chrome_extension",
	})
	incoming[soft2Key] = true

	tx, err := ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]uint), incoming))
	require.NoError(t, tx.Commit())

	incoming = make(map[string]bool)
	soft3Key := softwareToUniqueString(fleet.Software{
		Name:    longName + "c",
		Version: "0.0.1",
		Source:  "chrome_extension",
	})
	incoming[soft3Key] = true

	tx, err = ds.writer.Beginx()
	require.NoError(t, err)
	require.NoError(t, insertNewInstalledHostSoftwareDB(context.Background(), tx, host1.ID, make(map[string]uint), incoming))
	require.NoError(t, tx.Commit())
}

func TestLoadSoftwareVulnerabilities(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
			{Name: "blah", Version: "1.0", Source: "apps"},
		},
	}
	host.HostSoftware = soft
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))
	require.NoError(t, ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"}))
	require.NoError(t, ds.InsertCVEForCPE(context.Background(), "cve-321-321-321", []string{"somecpe"}))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

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

func TestAllCPEs(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
			{Name: "blah", Version: "1.0", Source: "apps"},
		},
	}
	host.HostSoftware = soft
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))

	cpes, err := ds.AllCPEs(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, cpes, []string{"somecpe", "someothercpewithoutvulns"})
}

func TestNothingChanged(t *testing.T) {
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

func TestLoadSupportsTonsOfCVEs(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
			{Name: "blah", Version: "1.0", Source: "apps"},
		},
	}
	host.HostSoftware = soft
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host))

	sort.Slice(host.Software, func(i, j int) bool { return host.Software[i].Name < host.Software[j].Name })
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host.Software[1], "someothercpewithoutvulns"))
	for i := 0; i < 1000; i++ {
		part1 := rand.Intn(1000)
		part2 := rand.Intn(1000)
		part3 := rand.Intn(1000)
		cve := fmt.Sprintf("cve-%d-%d-%d", part1, part2, part3)
		require.NoError(t, ds.InsertCVEForCPE(context.Background(), cve, []string{"somecpe"}))
	}

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

func TestListSoftware(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	soft1 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host1.HostSoftware = soft1
	soft2 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		},
	}
	host2.HostSoftware = soft2

	require.NoError(t, ds.SaveHostSoftware(context.Background(), host1))
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host2))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))

	sort.Slice(host1.Software, func(i, j int) bool { return host1.Software[i].Name < host1.Software[j].Name })
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[0], "somecpe"))
	require.NoError(t, ds.AddCPEForSoftware(context.Background(), host1.Software[1], "someothercpewithoutvulns"))
	require.NoError(t, ds.InsertCVEForCPE(context.Background(), "cve-321-432-543", []string{"somecpe"}))
	require.NoError(t, ds.InsertCVEForCPE(context.Background(), "cve-333-444-555", []string{"somecpe"}))

	foo001 := fleet.Software{
		Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "somecpe",
		Vulnerabilities: fleet.VulnerabilitiesSlice{
			{CVE: "cve-321-432-543", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-321-432-543"},
			{CVE: "cve-333-444-555", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-333-444-555"},
		},
	}
	foo002 := fleet.Software{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"}
	foo003 := fleet.Software{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "someothercpewithoutvulns"}
	bar003 := fleet.Software{Name: "bar", Version: "0.0.3", Source: "deb_packages"}

	t.Run("lists everything", func(t *testing.T) {
		software, err := ds.ListSoftware(context.Background(), nil, fleet.ListOptions{})
		require.NoError(t, err)

		require.Len(t, software, 4)
		expected := []fleet.Software{foo001, foo002, foo003, bar003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("limits the results", func(t *testing.T) {
		software, err := ds.ListSoftware(context.Background(), nil, fleet.ListOptions{PerPage: 1})
		require.NoError(t, err)

		require.Len(t, software, 1)
		expected := []fleet.Software{foo001}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("paginates", func(t *testing.T) {
		software, err := ds.ListSoftware(context.Background(), nil, fleet.ListOptions{Page: 1, PerPage: 1})
		require.NoError(t, err)

		require.Len(t, software, 1)
		expected := []fleet.Software{foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		software, err := ds.ListSoftware(context.Background(), &team1.ID, fleet.ListOptions{})
		require.NoError(t, err)

		require.Len(t, software, 2)
		expected := []fleet.Software{foo001, foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})

	t.Run("filters by team and paginates", func(t *testing.T) {
		team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1-" + t.Name()})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))

		software, err := ds.ListSoftware(context.Background(), &team1.ID, fleet.ListOptions{PerPage: 1, Page: 1, OrderKey: "id"})
		require.NoError(t, err)

		require.Len(t, software, 1)
		expected := []fleet.Software{foo003}
		test.ElementsMatchSkipID(t, software, expected)
	})
}
