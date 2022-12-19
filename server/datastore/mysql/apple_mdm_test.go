package mysql

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/micromdm/nanodep/godep"
	"github.com/stretchr/testify/require"
)

func TestIngestAppleMDMDevicesIntoHosts(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
			HardwareSerial:  fmt.Sprintf("abc%d", i),
		})
		require.NoError(t, err)
	}

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 10)
	wantSerials := []string{}
	for _, h := range hosts {
		wantSerials = append(wantSerials, h.HardwareSerial)
	}

	// mock results incoming from depsync.Syncer
	depDevices := []godep.Device{
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},   // ingested; new serial, macOS, "added" op type
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},   // not ingested; duplicate serial
		{SerialNumber: "abc1", Model: "MacBook Pro", OS: "OSX", OpType: "added"},  // not ingested; existing serial
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "updated"}, // not ingested; op type "updated"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "deleted"}, // not ingested; op type "deleted"
		{SerialNumber: "pqr", Model: "IPad Pro", OS: "iOS", OpType: "added"},      // not ingested; iOS
		{SerialNumber: "tuv", Model: "Apple TV", OS: "tvOS", OpType: "added"},     // not ingested; tvOS
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},   // ingested; new serial, macOS, "added" op type
	}
	wantSerials = append(wantSerials, "abc", "xyz")

	// TODO: enable this part of the test if we decide to ingest from `nano_devices`
	// // new records in the `nano_devices` table also get ingested
	// for i, s := range []string{"nano1", "nano2", "abc"} {
	// 	_, err := ds.writer.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?,?,'auth')`, i+1, s)
	// 	require.NoError(t, err)
	// }
	// wantSerials = append(wantSerials, "nano1", "nano2") // "abc" already in depDevices and only ingested once

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(2), n) // 2 new hosts ("abc", "xyz")

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, len(wantSerials))
	gotSerials := []string{}
	for _, h := range hosts {
		gotSerials = append(gotSerials, h.HardwareSerial)
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
}

func TestIngestAppleMDMDeviceFromCheckin(t *testing.T) {
}
