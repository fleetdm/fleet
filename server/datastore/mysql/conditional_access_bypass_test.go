package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessBypass(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ConditionalAccessBypassDevice", testConditionalAccessBypassDevice},
		{"ConditionalAccessConsumeBypass", testConditionalAccessConsumeBypass},
		{"ConditionalAccessClearBypasses", testConditionalAccessClearBypasses},
		{"ConditionalAccessBypassDeletedWithHost", testConditionalAccessBypassDeletedWithHost},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testConditionalAccessBypassDevice(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	// Insert a bypass record
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	// Verify the record exists
	var count int
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Call again to test ON DUPLICATE KEY UPDATE behavior
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	// Verify still only one record exists
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func testConditionalAccessConsumeBypass(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	// Consume when no bypass exists - should return nil without error
	bypassedAt, err := ds.ConditionalAccessConsumeBypass(ctx, host.ID)
	require.NoError(t, err)
	require.Nil(t, bypassedAt)

	// Create a bypass record
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	// Consume the bypass
	bypassedAt, err = ds.ConditionalAccessConsumeBypass(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, bypassedAt)

	// Verify the record was deleted
	var count int
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Try to consume again - should return nil without error
	bypassedAt, err = ds.ConditionalAccessConsumeBypass(ctx, host.ID)
	require.NoError(t, err)
	require.Nil(t, bypassedAt)
}

func testConditionalAccessClearBypasses(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Clear when no bypasses exist - should succeed without error
	err := ds.ConditionalAccessClearBypasses(ctx)
	require.NoError(t, err)

	// Create multiple hosts with bypass records
	host1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo1.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	host2, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	host3, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		Hostname:        "foo3.local",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
	})
	require.NoError(t, err)

	// Create bypass records for all hosts
	err = ds.ConditionalAccessBypassDevice(ctx, host1.ID)
	require.NoError(t, err)
	err = ds.ConditionalAccessBypassDevice(ctx, host2.ID)
	require.NoError(t, err)
	err = ds.ConditionalAccessBypassDevice(ctx, host3.ID)
	require.NoError(t, err)

	// Verify all records exist
	var count int
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access")
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Clear all bypasses
	err = ds.ConditionalAccessClearBypasses(ctx)
	require.NoError(t, err)

	// Verify all records were deleted
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testConditionalAccessBypassDeletedWithHost(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a host
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	// Create a bypass record for the host
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	// Verify the bypass record exists
	var count int
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Delete the host
	err = ds.DeleteHost(ctx, host.ID)
	require.NoError(t, err)

	// Verify the bypass record was also deleted
	err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count, "bypass record should be deleted when host is deleted")
}
