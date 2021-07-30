package mysql

import (
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

	err := ds.SaveHostSoftware(host1)
	require.NoError(t, err)
	err = ds.SaveHostSoftware(host2)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(host1)
	require.NoError(t, err)
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft1.Software, host1.HostSoftware.Software)

	err = ds.LoadHostSoftware(host2)
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

	err = ds.SaveHostSoftware(host1)
	require.NoError(t, err)
	err = ds.SaveHostSoftware(host2)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(host1)
	require.NoError(t, err)
	assert.False(t, host1.HostSoftware.Modified)
	test.ElementsMatchSkipID(t, soft1.Software, host1.HostSoftware.Software)

	err = ds.LoadHostSoftware(host2)
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

	err = ds.SaveHostSoftware(host1)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(host1)
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

	err := ds.SaveHostSoftware(host1)
	require.NoError(t, err)

	iterator, err := ds.AllSoftwareWithoutCPEIterator()
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

	err = ds.AddCPEForSoftware(fleet.Software{ID: id}, "some:cpe")
	require.NoError(t, err)

	iterator, err = ds.AllSoftwareWithoutCPEIterator()
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
}
