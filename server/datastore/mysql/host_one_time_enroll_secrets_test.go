package mysql

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHostOneTimeEnrollSecrets(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Mint", testOneTimeEnrollSecretMint},
		{"EnrollBothPlanes", testOneTimeEnrollSecretEnrollBothPlanes},
		{"EnrollRules", testOneTimeEnrollSecretEnrollRules},
		{"RejectSharedSecretForMDMHosts", testOneTimeEnrollSecretRejectShared},
		{"ResetTurnOffAndDelete", testOneTimeEnrollSecretResetAndDelete},
		{"Cleanup", testOneTimeEnrollSecretCleanup},
		{"FleetdProfileByTeamAndIdentifier", testFleetdProfileByTeamAndIdentifier},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

const oneTimeSecretProfile = `<dict><key>EnrollSecret</key><string>$FLEET_HOST_SECRET_ENROLL_SECRET</string></dict>` // nolint:gosec // Not hardcoded credentials

func newOneTimeSecretTestHost(t *testing.T, ds *Datastore, platform string, teamID *uint) *fleet.Host {
	id := strings.ToUpper(uuid.NewString())
	h, err := ds.NewHost(t.Context(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(id),
		NodeKey:         new("nk-" + id),
		UUID:            id,
		Hostname:        "host-" + id[:8],
		HardwareSerial:  "SERIAL-" + id[:8],
		Platform:        platform,
		TeamID:          teamID,
	})
	require.NoError(t, err)
	return h
}

// mintOneTimeSecret delivers the fleetd profile placeholder for the host, the
// way the nanomdm service does when the device fetches the command, and
// returns the stored row for the secret that was substituted.
func mintOneTimeSecret(t *testing.T, ds *Datastore, hostUUID string) *fleet.HostOneTimeEnrollSecret {
	expanded, err := ds.ExpandHostSecrets(t.Context(), oneTimeSecretProfile, hostUUID)
	require.NoError(t, err)
	start := strings.Index(expanded, "<string>") + len("<string>")
	end := strings.Index(expanded, "</string>")
	secret := expanded[start:end]
	require.NotContains(t, secret, fleet.HostSecretPrefix)
	require.NotEmpty(t, secret)

	row, err := ds.GetHostOneTimeEnrollSecret(t.Context(), secret)
	require.NoError(t, err)
	require.Equal(t, secret, row.Secret)
	return row
}

func orbitEnrollOpts(h *fleet.Host, teamID *uint, extra ...fleet.DatastoreEnrollOrbitOption) []fleet.DatastoreEnrollOrbitOption {
	return append([]fleet.DatastoreEnrollOrbitOption{
		fleet.WithEnrollOrbitMDMEnabled(true),
		fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
			HardwareUUID:   h.UUID,
			HardwareSerial: h.HardwareSerial,
			Platform:       h.Platform,
		}),
		fleet.WithEnrollOrbitNodeKey(uuid.NewString()),
		fleet.WithEnrollOrbitTeamID(teamID),
	}, extra...)
}

func osqueryEnrollOpts(h *fleet.Host, teamID *uint, extra ...fleet.DatastoreEnrollOsqueryOption) []fleet.DatastoreEnrollOsqueryOption {
	return append([]fleet.DatastoreEnrollOsqueryOption{
		fleet.WithEnrollOsqueryMDMEnabled(true),
		fleet.WithEnrollOsqueryHostID(h.UUID),
		fleet.WithEnrollOsqueryHardwareUUID(h.UUID),
		fleet.WithEnrollOsqueryHardwareSerial(h.HardwareSerial),
		fleet.WithEnrollOsqueryNodeKey(uuid.NewString()),
		fleet.WithEnrollOsqueryTeamID(teamID),
	}, extra...)
}

func requireEnrollmentRejected(t *testing.T, err error, reason string, hostID *uint) {
	var rejected *fleet.EnrollmentRejectedError
	require.ErrorAs(t, err, &rejected)
	require.NotNil(t, rejected)
	require.Equal(t, reason, rejected.Reason)
	require.Equal(t, hostID, rejected.HostID)
}

func testOneTimeEnrollSecretMint(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	h := newOneTimeSecretTestHost(t, ds, "darwin", &team.ID)

	row := mintOneTimeSecret(t, ds, h.UUID)
	require.Equal(t, h.ID, *row.HostID)
	require.Equal(t, team.ID, *row.TeamID)
	require.Equal(t, "darwin", row.Platform)
	require.Equal(t, h.UUID, row.HardwareUUID)
	require.Equal(t, h.HardwareSerial, row.HardwareSerial)
	require.Nil(t, row.ConsumedAt)
	require.Nil(t, row.OrbitUsedAt)
	require.Nil(t, row.OsqueryUsedAt)

	// re-delivery hands out the same unconsumed secret
	again := mintOneTimeSecret(t, ds, h.UUID)
	require.Equal(t, row.ID, again.ID)
	require.Equal(t, row.Secret, again.Secret)

	// lookups are exact and case-sensitive
	if flipped := strings.ToUpper(row.Secret); flipped != row.Secret {
		_, err = ds.GetHostOneTimeEnrollSecret(ctx, flipped)
		require.True(t, fleet.IsNotFound(err))
	}
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, "not-a-secret")
	require.True(t, fleet.IsNotFound(err))
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, "")
	require.True(t, fleet.IsNotFound(err))

	// only device-channel enrollments mint
	_, err = ds.ExpandHostSecrets(ctx, oneTimeSecretProfile, h.UUID+":USER-1")
	require.Error(t, err)

	// unknown enrollment
	_, err = ds.ExpandHostSecrets(ctx, oneTimeSecretProfile, uuid.NewString())
	require.Error(t, err)

	// concurrent first deliveries (a NotNow re-serve racing a refetch) must
	// converge on a single unconsumed secret
	racer := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	const racers = 8
	results := make(chan string, racers)
	var wg sync.WaitGroup
	for range racers {
		wg.Go(func() {
			expanded, err := ds.ExpandHostSecrets(ctx, oneTimeSecretProfile, racer.UUID)
			if err != nil {
				results <- "error: " + err.Error()
				return
			}
			start := strings.Index(expanded, "<string>") + len("<string>")
			results <- expanded[start:strings.Index(expanded, "</string>")]
		})
	}
	wg.Wait()
	close(results)
	distinct := map[string]struct{}{}
	for r := range results {
		require.NotContains(t, r, "error:")
		distinct[r] = struct{}{}
	}
	require.Len(t, distinct, 1)
	var unconsumed int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &unconsumed,
		`SELECT COUNT(*) FROM host_one_time_enroll_secrets WHERE host_id = ? AND consumed_at IS NULL`, racer.ID))
	require.Equal(t, 1, unconsumed)

	// duplicate UUIDs: bind to the row orbit will match (osquery_host_id equal
	// to the uuid), even when it is not the lowest id
	lower := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	higher := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	require.Less(t, lower.ID, higher.ID)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET uuid = ? WHERE id = ?`, higher.UUID, lower.ID)
	require.NoError(t, err)
	row = mintOneTimeSecret(t, ds, higher.UUID)
	require.Equal(t, higher.ID, *row.HostID)
	require.Equal(t, higher.HardwareSerial, row.HardwareSerial)

	// duplicate UUIDs with no osquery_host_id match: lowest id, like the
	// matcher's serial branch
	shared := strings.ToUpper(uuid.NewString())
	first := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	second := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET uuid = ? WHERE id IN (?, ?)`, shared, first.ID, second.ID)
	require.NoError(t, err)
	row = mintOneTimeSecret(t, ds, shared)
	require.Equal(t, first.ID, *row.HostID)
	require.Equal(t, shared, row.HardwareUUID)
}

func testOneTimeEnrollSecretEnrollBothPlanes(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	row := mintOneTimeSecret(t, ds, h.UUID)

	// orbit uses it first
	orbitOpts := orbitEnrollOpts(h, row.TeamID, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))
	enrolled, err := ds.EnrollOrbit(ctx, orbitOpts...)
	require.NoError(t, err)
	require.Equal(t, h.ID, enrolled.ID)
	afterOrbit, err := ds.Host(ctx, h.ID)
	require.NoError(t, err)
	orbitNodeKey := *afterOrbit.OrbitNodeKey

	row, err = ds.GetHostOneTimeEnrollSecret(ctx, row.Secret)
	require.NoError(t, err)
	require.NotNil(t, row.ConsumedAt)
	require.NotNil(t, row.OrbitUsedAt)
	require.Nil(t, row.OsqueryUsedAt)

	// a second orbit use is rejected and does not rotate the node key
	_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, row.TeamID, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
	requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, &h.ID)
	afterReject, err := ds.Host(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, orbitNodeKey, *afterReject.OrbitNodeKey)

	// osqueryd uses the same secret seconds later
	enrolledOsquery, err := ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, row.TeamID, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
	require.NoError(t, err)
	require.Equal(t, h.ID, enrolledOsquery.ID)
	row, err = ds.GetHostOneTimeEnrollSecret(ctx, row.Secret)
	require.NoError(t, err)
	require.NotNil(t, row.OsqueryUsedAt)

	// and not twice
	_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, row.TeamID, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
	requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, &h.ID)

	// once spent, a re-delivery mints a fresh secret and keeps the spent row
	fresh := mintOneTimeSecret(t, ds, h.UUID)
	require.NotEqual(t, row.ID, fresh.ID)
	require.NotEqual(t, row.Secret, fresh.Secret)
	spent, err := ds.GetHostOneTimeEnrollSecret(ctx, row.Secret)
	require.NoError(t, err)
	require.NotNil(t, spent.ConsumedAt)
}

func testOneTimeEnrollSecretEnrollRules(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("second plane outside the window is rejected", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		row := mintOneTimeSecret(t, ds, h.UUID)
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)

		_, err = ds.writer(ctx).ExecContext(ctx,
			`UPDATE host_one_time_enroll_secrets SET consumed_at = NOW(6) - INTERVAL 61 MINUTE WHERE id = ?`, row.ID)
		require.NoError(t, err)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, &h.ID)
	})

	t.Run("osquery may use it first, orbit follows", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		row := mintOneTimeSecret(t, ds, h.UUID)
		_, err := ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		row, err = ds.GetHostOneTimeEnrollSecret(ctx, row.Secret)
		require.NoError(t, err)
		require.NotNil(t, row.OsqueryUsedAt)
		require.NotNil(t, row.OrbitUsedAt)
	})

	t.Run("secret bound to another live host is rejected", func(t *testing.T) {
		victim := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		attacker := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		row := mintOneTimeSecret(t, ds, victim.UUID)

		// the service layer's identifier check would normally stop this; the
		// datastore still refuses to let the secret land on a different row.
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(attacker, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(attacker, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)

		a, err := ds.Host(ctx, attacker.ID)
		require.NoError(t, err)
		require.Nil(t, a.OrbitNodeKey)
		row, err = ds.GetHostOneTimeEnrollSecret(ctx, row.Secret)
		require.NoError(t, err)
		require.Nil(t, row.ConsumedAt)
	})

	t.Run("secret deleted between lookup and enrollment is spent", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		row := mintOneTimeSecret(t, ds, h.UUID)
		require.NoError(t, deleteHostOneTimeEnrollSecrets(ctx, ds.writer(ctx), h.ID))
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, nil)
	})
}

func testOneTimeEnrollSecretRejectShared(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	reject := fleet.WithEnrollOrbitRejectSharedSecretForMDMHosts(true)
	rejectOsquery := fleet.WithEnrollOsqueryRejectSharedSecretForMDMHosts(true)

	t.Run("Fleet MDM enrolled Mac", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		nanoEnroll(t, ds, h, false)

		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, reject)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectOsquery)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)

		// flag off: shared secrets keep working as before
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil)...)
		require.NoError(t, err)

		// a one-time secret is not subject to the rule
		row := mintOneTimeSecret(t, ds, h.UUID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectOsquery, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
	})

	t.Run("ABM assigned but not yet enrolled Mac", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "org", EncryptedToken: []byte(uuid.NewString()), RenewAt: time.Now().Add(365 * 24 * time.Hour)})
		require.NoError(t, err)
		require.NoError(t, ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID, map[uint]time.Time{}))

		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, reject)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)
	})

	t.Run("iPhone matched by serial", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "ios", nil)
		nanoEnroll(t, ds, h, false)
		_, err := ds.EnrollOsquery(ctx,
			fleet.WithEnrollOsqueryMDMEnabled(true),
			fleet.WithEnrollOsqueryHostID(uuid.NewString()),
			fleet.WithEnrollOsqueryHardwareUUID(uuid.NewString()),
			fleet.WithEnrollOsqueryHardwareSerial(h.HardwareSerial),
			fleet.WithEnrollOsqueryNodeKey(uuid.NewString()),
			rejectOsquery,
		)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)
	})

	t.Run("a deleted Fleet MDM Mac is recreated with a shared secret", func(t *testing.T) {
		// Deleting a host keeps its nano_enrollments row, and macOS hosts are only
		// recreated by fleetd enrolling again, so the insert branch must accept
		// the shared secret.
		h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		nanoEnroll(t, ds, h, false)
		require.NoError(t, ds.DeleteHost(ctx, h.ID))

		recreated, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, reject)...)
		require.NoError(t, err)
		require.NotEqual(t, h.ID, recreated.ID)
		require.Equal(t, h.UUID, recreated.UUID)
	})

	t.Run("hosts outside Fleet MDM are unaffected", func(t *testing.T) {
		thirdPartyMDMMac := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(thirdPartyMDMMac, nil, reject)...)
		require.NoError(t, err)

		linux := newOneTimeSecretTestHost(t, ds, "ubuntu", nil)
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(linux, nil, reject)...)
		require.NoError(t, err)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(linux, nil, rejectOsquery)...)
		require.NoError(t, err)

		// brand new host, no row to match
		_, err = ds.EnrollOrbit(ctx,
			fleet.WithEnrollOrbitMDMEnabled(true),
			fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{HardwareUUID: uuid.NewString(), HardwareSerial: uuid.NewString(), Platform: "darwin"}),
			fleet.WithEnrollOrbitNodeKey(uuid.NewString()),
			reject,
		)
		require.NoError(t, err)
	})
}

func testOneTimeEnrollSecretResetAndDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	nanoEnroll(t, ds, h, false)

	// MDM re-enrollment clears everything, spent or not
	spent := mintOneTimeSecret(t, ds, h.UUID)
	_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(spent.ID))...)
	require.NoError(t, err)
	unconsumed := mintOneTimeSecret(t, ds, h.UUID)
	require.NotEqual(t, spent.ID, unconsumed.ID)

	require.NoError(t, ds.MDMResetEnrollment(ctx, h.UUID, false))
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, spent.Secret)
	require.True(t, fleet.IsNotFound(err))
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, unconsumed.Secret)
	require.True(t, fleet.IsNotFound(err))

	// a re-delivery to a host holding an unconsumed secret (an admin resend)
	// hands out that same secret and writes nothing
	spent = mintOneTimeSecret(t, ds, h.UUID)
	_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(spent.ID))...)
	require.NoError(t, err)
	unconsumed = mintOneTimeSecret(t, ds, h.UUID)
	redelivered := mintOneTimeSecret(t, ds, h.UUID)
	require.Equal(t, unconsumed.ID, redelivered.ID)
	require.Equal(t, unconsumed.Secret, redelivered.Secret)

	// MDM turn off clears everything
	_, _, err = ds.MDMTurnOff(ctx, h.UUID)
	require.NoError(t, err)
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, spent.Secret)
	require.True(t, fleet.IsNotFound(err))
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, unconsumed.Secret)
	require.True(t, fleet.IsNotFound(err))

	// host deletion clears everything
	last := mintOneTimeSecret(t, ds, h.UUID)
	require.NoError(t, ds.DeleteHost(ctx, h.ID))
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, last.Secret)
	require.True(t, fleet.IsNotFound(err))
}

func testOneTimeEnrollSecretCleanup(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	other := newOneTimeSecretTestHost(t, ds, "darwin", nil)

	insert := func(hostID uint, consumedAgo *time.Duration) int64 {
		var consumedAt *time.Time
		if consumedAgo != nil {
			consumedAt = new(time.Now().Add(-*consumedAgo))
		}
		res, err := ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_one_time_enroll_secrets (secret, host_id, platform, hardware_uuid, hardware_serial, consumed_at)
			VALUES (?, ?, 'darwin', 'u', 's', ?)`, uuid.NewString(), hostID, consumedAt)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	twoHours := 2 * time.Hour
	oneMinute := time.Minute

	supersededOld := insert(h.ID, &twoHours)     // spent, older, past the window: removed
	supersededRecent := insert(h.ID, &oneMinute) // spent, older, inside the window: kept
	newestSpent := insert(h.ID, &twoHours)       // spent but newest for the host: kept
	unconsumed := insert(other.ID, nil)          // live: kept
	orphan := insert(999999, &twoHours)          // host gone: removed

	n, err := ds.CleanupHostOneTimeEnrollSecrets(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)

	var remaining []int64
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &remaining, `SELECT id FROM host_one_time_enroll_secrets ORDER BY id`))
	require.ElementsMatch(t, []int64{supersededRecent, newestSpent, unconsumed}, remaining)
	require.NotContains(t, remaining, supersededOld)
	require.NotContains(t, remaining, orphan)

	// idempotent
	n, err = ds.CleanupHostOneTimeEnrollSecrets(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
}

func testFleetdProfileByTeamAndIdentifier(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "profile-team"})
	require.NoError(t, err)

	var contents bytes.Buffer
	require.NoError(t, mobileconfig.FleetdProfileTemplate.Execute(&contents, mobileconfig.FleetdProfileOptions{
		EnrollSecret: fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret),
		ServerURL:    "https://fleet.example.com",
		PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
		PayloadName:  fleetmdm.FleetdConfigProfileName,
	}))
	cp, err := fleet.NewMDMAppleConfigProfile(contents.Bytes(), &team.ID)
	require.NoError(t, err)
	require.NoError(t, ds.BulkUpsertMDMAppleConfigProfiles(ctx, []*fleet.MDMAppleConfigProfile{cp}))

	got, err := ds.GetMDMAppleConfigProfileByTeamAndIdentifier(ctx, &team.ID, mobileconfig.FleetdConfigPayloadIdentifier)
	require.NoError(t, err)
	require.Equal(t, team.ID, *got.TeamID)
	require.Equal(t, mobileconfig.FleetdConfigPayloadIdentifier, got.Identifier)
	require.Contains(t, string(got.Mobileconfig), fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret))

	// "no team" has none
	_, err = ds.GetMDMAppleConfigProfileByTeamAndIdentifier(ctx, nil, mobileconfig.FleetdConfigPayloadIdentifier)
	require.True(t, fleet.IsNotFound(err))

	require.NoError(t, ds.DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx, &team.ID, mobileconfig.FleetdConfigPayloadIdentifier))
	_, err = ds.GetMDMAppleConfigProfileByTeamAndIdentifier(ctx, &team.ID, mobileconfig.FleetdConfigPayloadIdentifier)
	require.True(t, fleet.IsNotFound(err))
}
