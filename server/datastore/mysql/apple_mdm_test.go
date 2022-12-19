package mysql

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/godep"
	"github.com/stretchr/testify/require"
)

func TestIngestMDMAppleDevicesFromDEPSync(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        fmt.Sprintf("hostname_%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("osquery-host-id_%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("node-key_%d", i)),
			UUID:            fmt.Sprintf(fmt.Sprintf("uuid_%d", i)),
			HardwareSerial:  fmt.Sprintf("serial_%d", i),
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
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // ingested; new serial, macOS, "added" op type
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // not ingested; duplicate serial
		{SerialNumber: hosts[0].HardwareSerial, Model: "MacBook Pro", OS: "OSX", OpType: "added"}, // not ingested; existing serial
		{SerialNumber: "ijk", Model: "IPad Pro", OS: "iOS", OpType: "added"},                      // not ingested; iOS
		{SerialNumber: "tuv", Model: "Apple TV", OS: "tvOS", OpType: "added"},                     // not ingested; tvOS
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "updated"},                 // not ingested; op type "updated"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "deleted"},                 // not ingested; op type "deleted"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // ingested; new serial, macOS, "added" op type
	}
	wantSerials = append(wantSerials, "abc", "xyz")

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

func TestIngestMDMAppleDeviceFromCheckin(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestIngestMDMAppleCheckinHostAlreadyExistsInFleet", testIngestMDMAppleHostAlreadyExistsInFleet},
		{"TestIngestMDMApplCheckinAfterDEPSync", testIngestMDMAppleCheckinAfterDEPSync},
		{"TestIngestMDMApplCheckinBeforeDEPSync", testIngestMDMAppleCheckinBeforeDEPSync},
		{"TestIngestMDMAppleCheckinMultipleAuthenticateRequests", testIngestMDMAppleCheckinMultipleAuthenticateRequests},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testIngestMDMAppleHostAlreadyExistsInFleet(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ingester := fleet.NewMDMAppleHostIngester(ds, kitlog.NewNopLogger())

	testSerial := "test-serial"
	testUUID := "test-uuid"

	_, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-name",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1337"),
		NodeKey:         ptr.String("1337"),
		UUID:            testUUID,
		HardwareSerial:  testSerial,
	})
	require.NoError(t, err)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	err = ingester.Ingest(ctx, &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUUID, "MacBook Pro"))),
	})
	require.NoError(t, err)
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func testIngestMDMAppleCheckinAfterDEPSync(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ingester := fleet.NewMDMAppleHostIngester(ds, kitlog.NewNopLogger())

	testSerial := "test-serial"
	testUUID := "test-uuid"

	// simulate a host that is first ingested via DEP (e.g., the device was added via Apple Business Manager)
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	// hosts that are first ingested via DEP will have a serial number but not a UUID because UUID
	// is not available from the DEP sync endpoint
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, "", hosts[0].UUID)

	// now simulate the initial MDM checkin by that same host
	err = ingester.Ingest(ctx, &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUUID, "MacBook Pro"))),
	})
	require.NoError(t, err)
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func testIngestMDMAppleCheckinBeforeDEPSync(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ingester := fleet.NewMDMAppleHostIngester(ds, kitlog.NewNopLogger())

	testSerial := "test-serial"
	testUUID := "test-uuid"

	// ingest host on initial mdm checkin
	err := ingester.Ingest(ctx, &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUUID, "MacBook Pro"))),
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	// no effect if same host appears in DEP sync
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), n)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func testIngestMDMAppleCheckinMultipleAuthenticateRequests(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ingester := fleet.NewMDMAppleHostIngester(ds, kitlog.NewNopLogger())

	testSerial := "test-serial"
	testUUID := "test-uuid"

	err := ingester.Ingest(ctx, &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUUID, "MacBook Pro"))),
	})
	require.NoError(t, err)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	// duplicate Authenticate request has no effect
	err = ingester.Ingest(ctx, &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUUID, "MacBook Pro"))),
	})
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func xmlForTest(msgType string, serial string, udid string, model string) string {
	return fmt.Sprintf(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MessageType</key>
	<string>%s</string>
	<key>SerialNumber</key>
	<string>%s</string>
	<key>UDID</key>
	<string>%s</string>
	<key>Model</key>
	<string>%s</string>
</dict>
</plist>`, msgType, serial, udid, model)
}
