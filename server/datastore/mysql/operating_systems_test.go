package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOperatingSystems(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	// no os records
	list, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)

	// with os records
	seedByID := seedOperatingSystems(t, ds)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, len(seedByID))

	osByID := make(map[uint]fleet.OperatingSystem)
	for _, os := range list {
		osByID[os.ID] = os
	}
	for _, os := range osByID {
		require.Equal(t, seedByID[os.ID], os)
	}
}

func TestListOperatingSystemsForPlatform(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	// no os records
	list, err := ds.ListOperatingSystemsForPlatform(ctx, "windows")
	require.NoError(t, err)
	require.Len(t, list, 0)

	// with os records
	seedByID := seedOperatingSystems(t, ds)
	list, err = ds.ListOperatingSystemsForPlatform(ctx, "windows")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, seedByID[list[0].ID], list[0])

	// OS does not exist
	list, err = ds.ListOperatingSystemsForPlatform(ctx, "foo")
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestUpdateHostOperatingSystem(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	testHostID := uint(42)
	testOS := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "16.04.7 LTS",
		Arch:          "x86_64",
		KernelVersion: "5.10.76-linuxkit",
		Platform:      "ubuntu",
	}

	// no records
	list, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
	_, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert new os record and host operating system record
	err = ds.UpdateHostOperatingSystem(ctx, testHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, true, isSameOS(t, testOS, list[0]))
	require.Equal(t, uint(1), list[0].OSVersionID)
	storedOS, err := getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))

	// insert a host with a different architecture os
	testHostID2 := uint(43)
	testOSarm := testOS
	testOSarm.Arch = "arm64"
	err = ds.UpdateHostOperatingSystem(ctx, testHostID2, testOSarm)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// os version id is the same for both architectures
	require.Equal(t, true, isSameOS(t, testOS, list[0]))
	require.Equal(t, uint(1), list[0].OSVersionID)
	require.Equal(t, true, isSameOS(t, testOSarm, list[1]))
	require.Equal(t, uint(1), list[1].OSVersionID)

	// new version creates a new os record
	testNewVersion := testOS
	testNewVersion.Version = "22.04 LTS"
	err = ds.UpdateHostOperatingSystem(ctx, testHostID, testNewVersion)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, uint(2), list[2].OSVersionID) // new version has new os version id
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testNewVersion, *storedOS))

	// new host with existing os
	testNewHostID := uint(44)
	err = ds.UpdateHostOperatingSystem(ctx, testNewHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testNewHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))

	// no change
	err = ds.UpdateHostOperatingSystem(ctx, testNewHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testNewHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))
}

func TestUniqueOS(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	testHostIDs := make([]uint, 50)
	testOS := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "16.04.7 LTS",
		Arch:          "x86_64",
		KernelVersion: "5.10.76-linuxkit",
		Platform:      "ubuntu",
	}

	var wg sync.WaitGroup
	for i := range testHostIDs {
		wg.Add(1)
		go func(id int) {
			err := ds.UpdateHostOperatingSystem(ctx, uint(id), testOS) //nolint:gosec // dismiss G115
			assert.NoError(t, err)
			wg.Done()
		}(i)

	}
	wg.Wait()
	list, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestMaybeNewOperatingSystem(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	seedOperatingSystems(t, ds)
	list, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	osByID := make(map[uint]fleet.OperatingSystem)
	for _, os := range list {
		osByID[os.ID] = os
	}

	testOS := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "16.04.7 LTS",
		Arch:          "x86_64",
		KernelVersion: "5.10.76-linuxkit",
		Platform:      "ubuntu",
	}

	// new os, returns a newly inserted record
	result1, err := getOrGenerateOperatingSystemDB(ctx, ds.writer(ctx), testOS)
	require.NoError(t, err)
	require.True(t, isSameOS(t, testOS, *result1))
	require.NotContains(t, osByID, result1.ID)

	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Equal(t, len(osByID)+1, len(list))

	osByID = make(map[uint]fleet.OperatingSystem)
	for _, os := range list {
		osByID[os.ID] = os
	}
	require.Contains(t, osByID, result1.ID)
	require.True(t, isSameOS(t, osByID[result1.ID], testOS))

	// no change, returns the existing record
	result2, err := getOrGenerateOperatingSystemDB(ctx, ds.writer(ctx), testOS)
	require.NoError(t, err)
	require.True(t, isSameOS(t, *result1, *result2))

	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Equal(t, len(osByID), len(list))

	osByID = make(map[uint]fleet.OperatingSystem)
	for _, os := range list {
		osByID[os.ID] = os
	}
	require.Contains(t, osByID, result2.ID)
	require.True(t, isSameOS(t, osByID[result2.ID], testOS))

	// new version, returns new record
	testNewVersion := testOS
	testNewVersion.Version = "22.04 LTS"
	result3, err := getOrGenerateOperatingSystemDB(ctx, ds.writer(ctx), testNewVersion)
	require.NoError(t, err)
	require.True(t, isSameOS(t, testNewVersion, *result3))

	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Equal(t, len(osByID)+1, len(list))

	osByID = make(map[uint]fleet.OperatingSystem)
	for _, os := range list {
		osByID[os.ID] = os
	}
	require.Contains(t, osByID, result3.ID)
	require.True(t, isSameOS(t, osByID[result3.ID], testNewVersion))
	require.Contains(t, osByID, result2.ID)
	require.True(t, isSameOS(t, osByID[result2.ID], testOS))
}

func TestMaybeUpdateHostOperatingSystem(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	seedOperatingSystems(t, ds)
	osList, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)

	testHostID := uint(42)

	// no record exists for test host
	_, err = getIDHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert test host and os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[0].ID)
	require.NoError(t, err)
	osID, err := getIDHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[0].ID, osID)

	// update test host with new os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[1].ID)
	require.NoError(t, err)
	osID, err = getIDHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1].ID, osID)

	// no change
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[1].ID)
	require.NoError(t, err)
	osID, err = getIDHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1].ID, osID)
}

func TestGetHostOperatingSystem(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	seedOperatingSystems(t, ds)
	osList, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)

	testHostID := uint(42)

	// no record exists for test host
	_, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = ds.GetHostOperatingSystem(ctx, testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert test host and os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[0].ID)
	require.NoError(t, err)
	os, err := getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[0], *os)

	os, err = ds.GetHostOperatingSystem(ctx, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[0], *os)

	// update test host with new os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[1].ID)
	require.NoError(t, err)
	os, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)

	os, err = ds.GetHostOperatingSystem(ctx, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)

	// no change
	err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID, osList[1].ID)
	require.NoError(t, err)
	os, err = getHostOperatingSystemDB(ctx, ds.writer(ctx), testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)

	os, err = ds.GetHostOperatingSystem(ctx, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)
}

func TestCleanupHostOperatingSystems(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	seedOperatingSystems(t, ds)
	testOSs, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)

	testHosts := make([]*fleet.Host, 10)
	osByHostID := make(map[uint]fleet.OperatingSystem)

	for i := range testHosts {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
		testHosts[i] = h

		// insert host operating system record so initially each os is seeded with two hosts
		hostOS := testOSs[i%len(testOSs)]
		err = upsertHostOperatingSystemDB(ctx, ds.writer(ctx), h.ID, hostOS.ID)
		require.NoError(t, err)
		osByHostID[h.ID] = hostOS
	}

	assertDeletedHostOS := func(expectDeletedIDs []uint) {
		for _, h := range testHosts {
			os, err := getHostOperatingSystemDB(ctx, ds.writer(ctx), h.ID)
			if errors.Is(err, sql.ErrNoRows) {
				require.Contains(t, expectDeletedIDs, h.ID)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, os)
			require.Equal(t, osByHostID[h.ID], *os)
		}
	}
	assertDeletedOS := func(expectDeletedIDs []uint) {
		list, err := ds.ListOperatingSystems(ctx)
		require.NoError(t, err)

		byID := make(map[uint]fleet.OperatingSystem)
		for _, os := range list {
			byID[os.ID] = os
		}

		for _, testOS := range testOSs {
			os, ok := byID[testOS.ID]
			if !ok {
				require.Contains(t, expectDeletedIDs, testOS.ID)
				return
			}
			require.Equal(t, testOS, os)
		}
	}

	// initial state
	assertDeletedHostOS([]uint{})
	assertDeletedOS([]uint{})

	// nothing to clean up
	require.NoError(t, ds.CleanupHostOperatingSystems(ctx))
	assertDeletedHostOS([]uint{})
	assertDeletedOS([]uint{})

	// delete some hosts
	var deletedHostIDs []uint
	require.NoError(t, ds.DeleteHost(ctx, testHosts[0].ID))
	require.NoError(t, ds.DeleteHost(ctx, testHosts[1].ID))
	deletedHostIDs = append(deletedHostIDs, testHosts[0].ID, testHosts[1].ID)

	require.NoError(t, ds.CleanupHostOperatingSystems(ctx))

	// clean up removes host_operating_system record for deleted hosts
	assertDeletedHostOS(deletedHostIDs)
	// no deleted operating_system records because at least one host
	// with each operating system still exists
	assertDeletedOS([]uint{})

	// delete remaining host for seedOSList[0]
	require.NoError(t, ds.DeleteHost(ctx, testHosts[5].ID))
	deletedHostIDs = append(deletedHostIDs, testHosts[5].ID)

	require.NoError(t, ds.CleanupHostOperatingSystems(ctx))

	// clean up removes host_operating_system record for deleted hosts
	assertDeletedHostOS(deletedHostIDs)
	// operating_system record for seedOSList[0] is deleted because
	// no remaining hosts have that operating system
	assertDeletedOS([]uint{testOSs[0].ID})
}

func seedOperatingSystems(t *testing.T, ds *Datastore) map[uint]fleet.OperatingSystem {
	osSeeds := []fleet.OperatingSystem{
		{
			Name:           "Microsoft Windows 11 Enterprise Evaluation",
			Version:        "10.0.22000.795",
			Arch:           "64-bit",
			KernelVersion:  "10.0.22000.795",
			Platform:       "windows",
			DisplayVersion: "21H2",
			OSVersionID:    1,
		},
		{
			Name:          "macOS",
			Version:       "12.3.1",
			Arch:          "x86_64",
			KernelVersion: "21.4.0",
			Platform:      "darwin",
			OSVersionID:   2,
		},
		{
			Name:          "Ubuntu",
			Version:       "20.04.2 LTS",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
			Platform:      "ubuntu",
			OSVersionID:   3,
		},
		{
			Name:          "Debian GNU/Linux",
			Version:       "10.0.0",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
			Platform:      "debian",
			OSVersionID:   4,
		},
		{
			Name:          "CentOS Linux",
			Version:       "8.3.2011",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
			Platform:      "rhel",
			OSVersionID:   5,
		},
	}
	storedById := make(map[uint]fleet.OperatingSystem)
	for _, os := range osSeeds {
		stored, err := newOperatingSystemDB(context.Background(), ds.writer(context.Background()), os)
		require.NoError(t, err)
		require.True(t, isSameOS(t, os, *stored))
		storedById[stored.ID] = *stored
	}
	return storedById
}

func isSameOS(t *testing.T, os1 fleet.OperatingSystem, os2 fleet.OperatingSystem) bool {
	return assert.ElementsMatch(t, []string{os1.Name, os1.Version, os1.Arch, os1.KernelVersion, os1.Platform}, []string{os2.Name, os2.Version, os2.Arch, os2.KernelVersion, os2.Platform})
}
