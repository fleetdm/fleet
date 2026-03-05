package mysql

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsEnforcement(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"BatchSetWindowsEnforcementProfiles", testBatchSetWindowsEnforcementProfiles},
		{"ListWindowsEnforcementProfiles", testListWindowsEnforcementProfiles},
		{"GetDeleteWindowsEnforcementProfile", testGetDeleteWindowsEnforcementProfile},
		{"BulkUpsertHostWindowsEnforcement", testBulkUpsertHostWindowsEnforcement},
		{"GetHostWindowsEnforcement", testGetHostWindowsEnforcement},
		{"GetHostWindowsEnforcementHash", testGetHostWindowsEnforcementHash},
		{"ListWindowsEnforcementToInstallRemove", testListWindowsEnforcementToInstallRemove},
		{"BulkSetPendingWindowsEnforcementForHosts", testBulkSetPendingWindowsEnforcementForHosts},
		{"GetPendingWindowsEnforcementForHost", testGetPendingWindowsEnforcementForHost},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testBatchSetWindowsEnforcementProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Apply empty set for no-team - should not error
	err := ds.BatchSetWindowsEnforcementProfiles(ctx, nil, nil)
	require.NoError(t, err)

	// Apply a set of profiles for no-team
	profiles := []*fleet.WindowsEnforcementProfile{
		{Name: "registry-cis", RawPolicy: []byte(`{"registry":[{"path":"HKLM\\Test","name":"TestVal","type":"dword","value":1}]}`)},
		{Name: "audit-cis", RawPolicy: []byte(`{"audit_policy":[{"subcategory":"Logon","include":"Success"}]}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, profiles)
	require.NoError(t, err)

	// Verify profiles were created
	got, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)

	for _, p := range got {
		require.True(t, strings.HasPrefix(p.ProfileUUID, fleet.WindowsEnforcementUUIDPrefix),
			"profile UUID should start with %q, got %q", fleet.WindowsEnforcementUUIDPrefix, p.ProfileUUID)
		require.NotEmpty(t, p.Checksum)
	}

	// Apply same profiles - should be idempotent (no changes)
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, profiles)
	require.NoError(t, err)

	got2, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got2, 2)

	// UUIDs should be the same as before (not regenerated)
	uuids1 := make(map[string]string)
	for _, p := range got {
		uuids1[p.Name] = p.ProfileUUID
	}
	for _, p := range got2 {
		require.Equal(t, uuids1[p.Name], p.ProfileUUID)
	}

	// Update one profile's content
	profiles[0].RawPolicy = []byte(`{"registry":[{"path":"HKLM\\Test","name":"TestVal","type":"dword","value":2}]}`)
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, profiles)
	require.NoError(t, err)

	got3, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got3, 2)

	// Remove one profile
	profiles = profiles[:1]
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, profiles)
	require.NoError(t, err)

	got4, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got4, 1)
	require.Equal(t, "registry-cis", got4[0].Name)

	// Apply profiles for a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test-team"})
	require.NoError(t, err)

	teamProfiles := []*fleet.WindowsEnforcementProfile{
		{Name: "team-registry", RawPolicy: []byte(`{"registry":[]}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, &team.ID, teamProfiles)
	require.NoError(t, err)

	gotTeam, err := ds.ListWindowsEnforcementProfiles(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, gotTeam, 1)

	// Verify no-team profiles are unaffected
	gotNoTeam, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, gotNoTeam, 1)

	// Clear all team profiles
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, &team.ID, nil)
	require.NoError(t, err)

	gotTeam, err = ds.ListWindowsEnforcementProfiles(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, gotTeam, 0)
}

func testListWindowsEnforcementProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Empty list
	profiles, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, profiles)

	// Add profiles and verify order
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "zz-last", RawPolicy: []byte(`{}`)},
		{Name: "aa-first", RawPolicy: []byte(`{}`)},
		{Name: "mm-middle", RawPolicy: []byte(`{}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	profiles, err = ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 3)
	require.Equal(t, "aa-first", profiles[0].Name)
	require.Equal(t, "mm-middle", profiles[1].Name)
	require.Equal(t, "zz-last", profiles[2].Name)
}

func testGetDeleteWindowsEnforcementProfile(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a profile
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "test-profile", RawPolicy: []byte(`{"registry":[]}`)},
	}
	err := ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	profiles, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	uuid := profiles[0].ProfileUUID

	// Get by UUID
	got, err := ds.GetWindowsEnforcementProfile(ctx, uuid)
	require.NoError(t, err)
	require.Equal(t, "test-profile", got.Name)
	require.Equal(t, uuid, got.ProfileUUID)

	// Delete by UUID
	err = ds.DeleteWindowsEnforcementProfile(ctx, uuid)
	require.NoError(t, err)

	// Verify deletion
	profiles, err = ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, profiles)
}

func testBulkUpsertHostWindowsEnforcement(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Empty payload should not error
	err := ds.BulkUpsertHostWindowsEnforcement(ctx, nil)
	require.NoError(t, err)

	// Create host and enforcement profile
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("test-host-osquery"),
		NodeKey:         ptr.String("test-host-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "test-profile", RawPolicy: []byte(`{}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	profiles, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	pending := fleet.MDMDeliveryPending
	payload := []*fleet.HostWindowsEnforcement{
		{
			HostUUID:      host.UUID,
			ProfileUUID:   profiles[0].ProfileUUID,
			Name:          "test-profile",
			Status:        &pending,
			OperationType: fleet.MDMOperationTypeInstall,
		},
	}
	err = ds.BulkUpsertHostWindowsEnforcement(ctx, payload)
	require.NoError(t, err)

	// Update status
	verified := fleet.MDMDeliveryVerified
	payload[0].Status = &verified
	err = ds.BulkUpsertHostWindowsEnforcement(ctx, payload)
	require.NoError(t, err)
}

func testGetHostWindowsEnforcement(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create host and enforcement profile
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-2",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("test-host-2-osquery"),
		NodeKey:         ptr.String("test-host-2-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// Empty result for host with no enforcement
	results, err := ds.GetHostWindowsEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, results)

	// Create profile and upsert enforcement status
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "test-get-profile", RawPolicy: []byte(`{}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	profiles, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	pending := fleet.MDMDeliveryPending
	payload := []*fleet.HostWindowsEnforcement{
		{
			HostUUID:      host.UUID,
			ProfileUUID:   profiles[0].ProfileUUID,
			Name:          "test-get-profile",
			Status:        &pending,
			OperationType: fleet.MDMOperationTypeInstall,
		},
	}
	err = ds.BulkUpsertHostWindowsEnforcement(ctx, payload)
	require.NoError(t, err)

	results, err = ds.GetHostWindowsEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, host.UUID, results[0].HostUUID)
	assert.Equal(t, profiles[0].ProfileUUID, results[0].ProfileUUID)
	assert.Equal(t, "test-get-profile", results[0].Name)
	require.NotNil(t, results[0].Status)
	assert.Equal(t, fleet.MDMDeliveryPending, *results[0].Status)
}

func testGetHostWindowsEnforcementHash(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a Windows host with no team
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "hash-test-host",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("hash-test-osquery"),
		NodeKey:         ptr.String("hash-test-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// No enforcement profiles - hash should be empty
	hash, err := ds.GetHostWindowsEnforcementHash(ctx, host.UUID)
	require.NoError(t, err)
	assert.Empty(t, hash)

	// Add enforcement profiles for no-team
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "hash-test-profile", RawPolicy: []byte(`{}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	// Now hash should be non-empty
	hash, err = ds.GetHostWindowsEnforcementHash(ctx, host.UUID)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Same profiles should give same hash
	hash2, err := ds.GetHostWindowsEnforcementHash(ctx, host.UUID)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Adding another profile should change the hash
	batch = append(batch, &fleet.WindowsEnforcementProfile{
		Name: "hash-test-profile-2", RawPolicy: []byte(`{"registry":[]}`),
	})
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	hash3, err := ds.GetHostWindowsEnforcementHash(ctx, host.UUID)
	require.NoError(t, err)
	assert.NotEmpty(t, hash3)
	assert.NotEqual(t, hash, hash3)
}

func testListWindowsEnforcementToInstallRemove(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a Windows host
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "install-remove-host",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("install-remove-osquery"),
		NodeKey:         ptr.String("install-remove-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// No profiles - empty results
	toInstall, err := ds.ListWindowsEnforcementToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, toInstall)

	toRemove, err := ds.ListWindowsEnforcementToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// Add enforcement profile for no-team (host has no team)
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "install-test", RawPolicy: []byte(`{"registry":[]}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	profiles, err := ds.ListWindowsEnforcementProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	// Now there should be profiles to install (desired but not current)
	toInstall, err = ds.ListWindowsEnforcementToInstall(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, toInstall)

	// Find our host in the results
	var foundHost bool
	for _, p := range toInstall {
		if p.HostUUID == host.UUID {
			foundHost = true
			assert.Equal(t, profiles[0].ProfileUUID, p.ProfileUUID)
			break
		}
	}
	assert.True(t, foundHost, "host should be in install list")

	// Nothing to remove yet
	toRemove, err = ds.ListWindowsEnforcementToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// Now upsert the enforcement status (simulating that it was installed)
	verified := fleet.MDMDeliveryVerified
	payload := []*fleet.HostWindowsEnforcement{
		{
			HostUUID:      host.UUID,
			ProfileUUID:   profiles[0].ProfileUUID,
			Status:        &verified,
			OperationType: fleet.MDMOperationTypeInstall,
		},
	}
	err = ds.BulkUpsertHostWindowsEnforcement(ctx, payload)
	require.NoError(t, err)

	// Remove the enforcement profile from desired state
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, nil)
	require.NoError(t, err)

	// Now there should be profiles to remove (current but not desired)
	toRemove, err = ds.ListWindowsEnforcementToRemove(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, toRemove)

	foundHost = false
	for _, p := range toRemove {
		if p.HostUUID == host.UUID {
			foundHost = true
			break
		}
	}
	assert.True(t, foundHost, "host should be in remove list")
}

func testBulkSetPendingWindowsEnforcementForHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Empty host IDs - should not error
	err := ds.BulkSetPendingWindowsEnforcementForHosts(ctx, nil)
	require.NoError(t, err)

	err = ds.BulkSetPendingWindowsEnforcementForHosts(ctx, []uint{})
	require.NoError(t, err)

	// Create a Windows host and enforcement profile
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "bulk-pending-host",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("bulk-pending-osquery"),
		NodeKey:         ptr.String("bulk-pending-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "bulk-pending-profile", RawPolicy: []byte(`{"registry":[]}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	// Set pending for this host
	err = ds.BulkSetPendingWindowsEnforcementForHosts(ctx, []uint{host.ID})
	require.NoError(t, err)

	// Verify the host now has pending enforcement
	results, err := ds.GetHostWindowsEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, results[0].Status)
	assert.Equal(t, fleet.MDMDeliveryPending, *results[0].Status)
	assert.Equal(t, fleet.MDMOperationTypeInstall, results[0].OperationType)
}

func testGetPendingWindowsEnforcementForHost(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a Windows host
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "pending-enforcement-host",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("pending-enforcement-osquery"),
		NodeKey:         ptr.String("pending-enforcement-node"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// No enforcement profiles - empty result
	policies, err := ds.GetPendingWindowsEnforcementForHost(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, policies)

	// Add enforcement profiles
	batch := []*fleet.WindowsEnforcementProfile{
		{Name: "cis-registry", RawPolicy: []byte(`{"registry":[{"path":"HKLM\\Test","name":"v","type":"dword","value":1}]}`)},
		{Name: "cis-audit", RawPolicy: []byte(`{"audit_policy":[]}`)},
	}
	err = ds.BatchSetWindowsEnforcementProfiles(ctx, nil, batch)
	require.NoError(t, err)

	// Now the host should have pending policies
	policies, err = ds.GetPendingWindowsEnforcementForHost(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	// Should be ordered by name
	assert.Equal(t, "cis-audit", policies[0].Name)
	assert.Equal(t, "cis-registry", policies[1].Name)

	// Each should have profile UUID and raw_policy
	for _, p := range policies {
		assert.NotEmpty(t, p.ProfileUUID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.RawPolicy)
	}
}
