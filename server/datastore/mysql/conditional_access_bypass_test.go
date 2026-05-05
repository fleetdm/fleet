package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessBypass(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ConditionalAccessBypassDevice", testConditionalAccessBypassDevice},
		{"ConditionalAccessBypassDeviceWithBlockingPolicy", testConditionalAccessBypassDeviceWithBlockingPolicy},
		{"ConditionalAccessConsumeBypass", testConditionalAccessConsumeBypass},
		{"ConditionalAccessClearBypasses", testConditionalAccessClearBypasses},
		{"ConditionalAccessBypassDeletedWithHost", testConditionalAccessBypassDeletedWithHost},
		{"ConditionalAccessBypassedAt", testConditionalAccessBypassedAt},
		{"ConditionalAccessBypassAllowedWithNonCAFailingCriticalPolicy", testConditionalAccessBypassAllowedWithNonCAFailingCriticalPolicy},
		{"ConditionalAccessBypassAllowedWithCAEnabledNonCriticalPolicy", testConditionalAccessBypassAllowedWithCAEnabledNonCriticalPolicy},
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

func testConditionalAccessBypassedAt(t *testing.T, ds *Datastore) {
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

	bypassedAt, err := ds.ConditionalAccessBypassedAt(ctx, host.ID)
	require.NoError(t, err)
	require.Nil(t, bypassedAt)

	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	bypassedAt, err = ds.ConditionalAccessBypassedAt(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, bypassedAt)
	require.WithinDuration(t, time.Now(), *bypassedAt, 5*time.Second)

	bypassedAtAgain, err := ds.ConditionalAccessBypassedAt(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, bypassedAtAgain)
	require.Equal(t, bypassedAt, bypassedAtAgain)

	hostWithoutBypass, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	bypassedAtOther, err := ds.ConditionalAccessBypassedAt(ctx, hostWithoutBypass.ID)
	require.NoError(t, err)
	require.Nil(t, bypassedAtOther)
}

func testConditionalAccessBypassDeviceWithBlockingPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("blocking-policy-host"),
		UUID:            "blocking-policy-uuid",
		Hostname:        "blocking.local",
		PrimaryIP:       "192.168.1.10",
		PrimaryMac:      "30-65-EC-6F-C4-70",
	})
	require.NoError(t, err)

	// Assign host to a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "blocking-policy-team"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// Create a team CA-enabled critical policy that should block bypass
	policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:                     "ca-critical-policy",
		Query:                    "select 1;",
		Critical:                 true,
		ConditionalAccessEnabled: true,
	})
	require.NoError(t, err)

	// Record a failing result for this policy on the host
	err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{policy.ID: new(false)}, time.Now(), false, nil)
	require.NoError(t, err)

	// Bypass should fail because the host has a failing CA-enabled policy
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.Error(t, err)
	var badReqErr *fleet.BadRequestError
	require.ErrorAs(t, err, &badReqErr)

	// Verify no host_conditional_access row was created
	var count int
	innerErr := ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, innerErr)
	require.Equal(t, 0, count)
}

func testConditionalAccessBypassAllowedWithNonCAFailingCriticalPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("non-ca-policy-host"),
		UUID:            "non-ca-policy-uuid",
		Hostname:        "non-ca.local",
		PrimaryIP:       "192.168.1.11",
		PrimaryMac:      "30-65-EC-6F-C4-71",
	})
	require.NoError(t, err)

	// Assign host to a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "non-ca-policy-team"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// CA policy — passing
	caPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:                     "ca-policy-passing",
		Query:                    "select 1;",
		Critical:                 true,
		ConditionalAccessEnabled: true,
	})
	require.NoError(t, err)

	// Non-CA critical policy — failing
	nonCAPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:     "non-ca-policy-failing",
		Query:    "select 1;",
		Critical: true,
	})
	require.NoError(t, err)

	err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{
		caPolicy.ID:    ptr.Bool(true),  // passing
		nonCAPolicy.ID: ptr.Bool(false), // failing
	}, time.Now(), false, nil)
	require.NoError(t, err)

	// Bypass must succeed: the only failing policy is not CA-enabled
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	// Verify a host_conditional_access row was created
	var count int
	innerErr := ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, innerErr)
	require.Equal(t, 1, count)
}

// testConditionalAccessBypassAllowedWithCAEnabledNonCriticalPolicy verifies that a CA-enabled but
// non-critical failing policy does NOT block bypass. Both critical=1 AND conditional_access_enabled=1
// are required to block.
func testConditionalAccessBypassAllowedWithCAEnabledNonCriticalPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Carol", "carol@example.com", true)

	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("ca-non-critical-host"),
		UUID:            "ca-non-critical-uuid",
		Hostname:        "ca-non-critical.local",
		PrimaryIP:       "192.168.1.12",
		PrimaryMac:      "30-65-EC-6F-C4-72",
	})
	require.NoError(t, err)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "ca-non-critical-team"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// CA-enabled but NOT critical — failing
	nonCriticalCAPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:                     "ca-enabled-non-critical",
		Query:                    "select 1;",
		Critical:                 false,
		ConditionalAccessEnabled: true,
	})
	require.NoError(t, err)

	err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{
		nonCriticalCAPolicy.ID: ptr.Bool(false), // failing
	}, time.Now(), false, nil)
	require.NoError(t, err)

	// Bypass must succeed: policy is CA-enabled but not critical
	err = ds.ConditionalAccessBypassDevice(ctx, host.ID)
	require.NoError(t, err)

	var count int
	innerErr := ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id = ?", host.ID)
	require.NoError(t, innerErr)
	require.Equal(t, 1, count)
}
