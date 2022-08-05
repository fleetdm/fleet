package mysql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		require.Equal(t, os, seedByID[os.ID])
	}
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
	}

	// no records
	list, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
	_, err = getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert new os record and host operating system record
	err = ds.UpdateHostOperatingSystem(ctx, testHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, true, isSameOS(t, testOS, list[0]))
	storedOS, err := getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))

	// new version creates a new os record
	testNewVersion := testOS
	testNewVersion.Version = "22.04 LTS"
	err = ds.UpdateHostOperatingSystem(ctx, testHostID, testNewVersion)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testNewVersion, *storedOS))

	// new host with existing os
	testNewHostID := uint(43)
	err = ds.UpdateHostOperatingSystem(ctx, testNewHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer, testNewHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))

	// no change
	err = ds.UpdateHostOperatingSystem(ctx, testNewHostID, testOS)
	require.NoError(t, err)
	list, err = ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	storedOS, err = getHostOperatingSystemDB(ctx, ds.writer, testNewHostID)
	require.NoError(t, err)
	require.Equal(t, true, isSameOS(t, testOS, *storedOS))
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
	}

	// new os, returns a newly inserted record
	result1, err := getOrGenerateOperatingSystemDB(ctx, ds.writer, testOS)
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
	result2, err := getOrGenerateOperatingSystemDB(ctx, ds.writer, testOS)
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
	result3, err := getOrGenerateOperatingSystemDB(ctx, ds.writer, testNewVersion)
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
	_, err = getIDHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert test host and os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[0].ID)
	require.NoError(t, err)
	osID, err := getIDHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[0].ID, osID)

	// update test host with new os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[1].ID)
	require.NoError(t, err)
	osID, err = getIDHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1].ID, osID)

	// no change
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[1].ID)
	require.NoError(t, err)
	osID, err = getIDHostOperatingSystemDB(ctx, ds.writer, testHostID)
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
	_, err = getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// insert test host and os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[0].ID)
	require.NoError(t, err)
	os, err := getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[0], *os)

	// update test host with new os id
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[1].ID)
	require.NoError(t, err)
	os, err = getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)

	// no change
	err = upsertHostOperatingSystemDB(ctx, ds.writer, testHostID, osList[1].ID)
	require.NoError(t, err)
	os, err = getHostOperatingSystemDB(ctx, ds.writer, testHostID)
	require.NoError(t, err)
	require.Equal(t, osList[1], *os)
}

func seedOperatingSystems(t *testing.T, ds *Datastore) map[uint]fleet.OperatingSystem {
	osSeeds := []fleet.OperatingSystem{
		{
			Name:          "Microsoft Windows 11 Enterprise Evaluation",
			Version:       "21H2",
			Arch:          "64-bit",
			KernelVersion: "10.0.22000.795",
		},
		{
			Name:          "macOS",
			Version:       "12.3.1",
			Arch:          "x86_64",
			KernelVersion: "21.4.0",
		},
		{
			Name:          "Ubuntu",
			Version:       "20.04.2 LTS",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
		},
		{
			Name:          "Debian GNU/Linux",
			Version:       "10.0.0",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
		},
		{
			Name:          "CentOS Linux",
			Version:       "8.3.2011",
			Arch:          "x86_64",
			KernelVersion: "5.10.76-linuxkit",
		},
	}
	storedById := make(map[uint]fleet.OperatingSystem)
	for _, os := range osSeeds {
		stored, err := newOperatingSystemDB(context.Background(), ds.writer, os)
		require.NoError(t, err)
		require.True(t, isSameOS(t, os, *stored))
		storedById[stored.ID] = *stored
	}
	return storedById
}

func isSameOS(t *testing.T, os1 fleet.OperatingSystem, os2 fleet.OperatingSystem) bool {
	return assert.ElementsMatch(t, []string{os1.Name, os1.Version, os1.Arch, os1.KernelVersion}, []string{os2.Name, os2.Version, os2.Arch, os2.KernelVersion})
}
