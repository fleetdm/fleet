package mysql

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
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
		{"OverwriteWarningOnlyWhenEnrolled", testOneTimeEnrollSecretOverwriteWarning},
		{"RejectSharedSecretForAppleMDMHosts", testOneTimeEnrollSecretRejectShared},
		{"ResetTurnOffAndDelete", testOneTimeEnrollSecretResetAndDelete},
		{"Cleanup", testOneTimeEnrollSecretCleanup},
		{"WindowsMint", testOneTimeEnrollSecretWindowsMint},
		{"WindowsResendMints", testOneTimeEnrollSecretWindowsResendMints},
		{"WindowsHostBinding", testOneTimeEnrollSecretWindowsHostBinding},
		{"WindowsDeletedHostFleet", testOneTimeEnrollSecretWindowsDeletedHostFleet},
		{"WindowsDeleteUnusedSecrets", testOneTimeEnrollSecretWindowsDeleteUnusedSecrets},
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

func testOneTimeEnrollSecretOverwriteWarning(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	oldLogger := ds.logger
	t.Cleanup(func() { ds.logger = oldLogger })
	var buf bytes.Buffer
	ds.logger = logging.NewSlogLogger(logging.Options{Output: &buf, Debug: true})

	h := newOneTimeSecretTestHost(t, ds, "darwin", nil)
	row := mintOneTimeSecret(t, ds, h.UUID)
	_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
	require.NoError(t, err)
	_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "osquery host with duplicate identifier has enrolled in Fleet and will overwrite existing host data")
	require.Contains(t, buf.String(), fmt.Sprintf("host_id=%d", h.ID))

	buf.Reset()
	_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
	requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, &h.ID)
	_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
	requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretSpent, &h.ID)
	require.NotContains(t, buf.String(), "will overwrite existing host data")

	_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil)...)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "orbit host with duplicate identifier has enrolled in Fleet and will overwrite existing host data")
	require.Contains(t, buf.String(), fmt.Sprintf("host_id=%d", h.ID))
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
	reject := fleet.WithEnrollOrbitRejectSharedSecretForAppleMDMHosts(true)
	rejectOsquery := fleet.WithEnrollOsqueryRejectSharedSecretForAppleMDMHosts(true)

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

	rejectWindows := fleet.WithEnrollOrbitRejectSharedSecretForWindowsMDMHosts(true)
	rejectWindowsOsquery := fleet.WithEnrollOsqueryRejectSharedSecretForWindowsMDMHosts(true)
	t.Run("Fleet MDM enrolled Windows host", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-"+h.UUID, h.UUID)

		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectWindowsOsquery)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)

		// The platforms have separate switches: the Apple option alone leaves a Windows host alone.
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, reject)...)
		require.NoError(t, err)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectOsquery)...)
		require.NoError(t, err)

		// a one-time secret, which the admin gets delivered by resending the profile, is not subject to the rule
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		row := liveWindowsSecret(t, ds, device.ID)
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)

		// A new osquery identifier misses the host, but the hardware UUID is still the host's, so the shared secret is refused and
		// the host keeps its enrollment.
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectWindowsOsquery, fleet.WithEnrollOsqueryHostID("new-instance-"+h.UUID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil)
		_, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID)
		require.NoError(t, err)
	})

	t.Run("Windows host without a linked enrollment", func(t *testing.T) {
		notInMDM := newOneTimeSecretTestHost(t, ds, "windows", nil)
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(notInMDM, nil, rejectWindows)...)
		require.NoError(t, err)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(notInMDM, nil, rejectWindowsOsquery)...)
		require.NoError(t, err)
	})

	t.Run("Windows host that unenrolled from MDM", func(t *testing.T) {
		h := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-"+h.UUID, h.UUID)
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &h.ID)

		// the unenroll alert deletes the enrollment
		require.NoError(t, ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID))
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		require.NoError(t, err)
	})

	t.Run("a deleted Fleet MDM Windows host cannot be recreated with a shared secret", func(t *testing.T) {
		// A session after the delete pushed the device a one-time secret, so it has its own way back. The presented platform is
		// not trusted, so claiming another one does not help.
		h := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-"+h.UUID, h.UUID)
		require.NoError(t, ds.DeleteHost(ctx, h.ID))
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))

		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil)
		asMac := *h
		asMac.Platform = "darwin"
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(&asMac, nil, rejectWindows)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectWindowsOsquery)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil)
		_, err = ds.HostLiteByIdentifier(ctx, h.UUID)
		require.True(t, fleet.IsNotFound(err), "no host may be created for the refused enrollments")

		// The way back is the pushed secret.
		row := liveWindowsSecret(t, ds, device.ID)
		recreated, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		require.Equal(t, h.UUID, recreated.UUID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, rejectWindowsOsquery)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &recreated.ID)

		// Deleted again, as when re-imaged: the shared secret stays refused until the device enrolls in MDM again, which replaces
		// the linked enrollment.
		require.NoError(t, ds.DeleteHost(ctx, recreated.ID))
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil)
		_, err = ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, device.MDMHardwareID)
		require.NoError(t, err)
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, rejectWindows)...)
		require.NoError(t, err)
	})

	t.Run("a new Windows host no enrollment claims enrolls with a shared secret", func(t *testing.T) {
		fresh := &fleet.Host{UUID: strings.ToUpper(uuid.NewString()), HardwareSerial: "SERIAL-FRESH", Platform: "windows"}
		enrolled, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(fresh, nil, rejectWindows)...)
		require.NoError(t, err)

		// orbit enrolls the device in MDM within seconds, and osquery's first enrollment with the same shared secret still lands.
		insertWindowsEnrollment(t, ds, "hw-"+fresh.UUID, fresh.UUID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(fresh, nil, rejectWindowsOsquery)...)
		require.NoError(t, err)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(fresh, nil, rejectWindowsOsquery)...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &enrolled.ID)
	})

	t.Run("a Windows Autopilot host is only matched with a one-time secret", func(t *testing.T) {
		serial := "SERIAL-AP-" + uuid.NewString()[:8]
		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
			{AutopilotDeviceID: uuid.NewString(), HardwareSerial: serial, TenantID: "tenant"},
		}))
		device := &fleet.Host{UUID: strings.ToUpper(uuid.NewString()), HardwareSerial: serial, Platform: "windows"}
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(device, nil, rejectWindows)...)
		var rejected *fleet.EnrollmentRejectedError
		require.ErrorAs(t, err, &rejected)
		require.Equal(t, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, rejected.Reason)

		enrollment := insertWindowsEnrollment(t, ds, "hw-"+device.UUID, "")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, enrollment.ID))
		claimed, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(device, nil, rejectWindows,
			fleet.WithEnrollOrbitOneTimeEnrollSecret(liveWindowsSecret(t, ds, enrollment.ID).ID))...)
		require.NoError(t, err)
		require.Equal(t, *rejected.HostID, claimed.ID, "the one-time secret claims the pending host")
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

// insertWindowsEnrollment creates a Windows MDM enrollment linked to hostUUID. An empty hostUUID leaves it unlinked, the state a
// device is in before fleetd exists.
func insertWindowsEnrollment(t *testing.T, ds *Datastore, hardwareID, hostUUID string) *fleet.MDMWindowsEnrolledDevice {
	t.Helper()
	ctx := t.Context()

	deviceID := "device-" + hardwareID
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            deviceID,
		MDMHardwareID:          hardwareID,
		HostUUID:               hostUUID,
		MDMDeviceState:         "enrolled",
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-" + hardwareID,
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "user@example.com",
		MDMEnrollProtoVersion:  "4.0",
		MDMEnrollClientVersion: "10.0",
	}))

	device, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	require.NoError(t, err)
	return device
}

// linkWindowsEnrollment links an existing enrollment to a host, as fleetd enrolling does.
func linkWindowsEnrollment(t *testing.T, ds *Datastore, device *fleet.MDMWindowsEnrolledDevice, hostUUID string) {
	t.Helper()
	_, err := ds.UpdateMDMWindowsEnrollmentsHostUUID(t.Context(), hostUUID, device.MDMDeviceID)
	require.NoError(t, err)
}

// liveWindowsSecret returns the enrollment's live one-time enroll secret, failing the test if there is none.
func liveWindowsSecret(t *testing.T, ds *Datastore, enrollmentID uint) *fleet.HostOneTimeEnrollSecret {
	t.Helper()
	secret, err := ds.GetLiveWindowsMDMOneTimeEnrollSecret(t.Context(), enrollmentID)
	require.NoError(t, err)
	require.NotEmpty(t, secret)
	row, err := ds.GetHostOneTimeEnrollSecret(t.Context(), secret)
	require.NoError(t, err)
	return row
}

func countWindowsOneTimeEnrollSecrets(t *testing.T, ds *Datastore, enrollmentID uint) int {
	t.Helper()
	var n int
	require.NoError(t, sqlx.GetContext(t.Context(), ds.writer(t.Context()), &n,
		`SELECT COUNT(*) FROM host_one_time_enroll_secrets WHERE mdm_windows_enrollment_id = ?`, enrollmentID))
	return n
}

func testOneTimeEnrollSecretWindowsMint(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	device := insertWindowsEnrollment(t, ds, "hw-mint", "")

	// A host that needs no secret, typically one already running fleetd, has none, and looking must not mint one.
	secret, err := ds.GetLiveWindowsMDMOneTimeEnrollSecret(ctx, device.ID)
	require.NoError(t, err)
	require.Empty(t, secret)
	require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, device.ID), "looking up must never mint")

	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
	stored := liveWindowsSecret(t, ds, device.ID)

	// Minting again must reuse the live secret. Fleet re-enqueues the fleetd install on every session-start alert while fleetd
	// looks absent, and a second secret would be a second valid credential for the same device.
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
	require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, device.ID))

	require.Equal(t, device.ID, *stored.MDMWindowsEnrollmentID)
	require.Nil(t, stored.HostID, "a first install has no host to bind to")
	require.Empty(t, stored.HardwareUUID)
	require.Nil(t, stored.TeamID, "the team is applied after linkage, as it was with the global secret")
	require.Equal(t, "windows", stored.Platform)

	err = ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID+9999)
	require.True(t, fleet.IsNotFound(err), "an unknown enrollment must not mint a secret")

	// A consumed secret is not handed out again, so redelivering the profile afterwards, on a team transfer say, writes nothing.
	// A fresh secret is minted only by a new decision.
	usedByOrbit, err := ds.WindowsMDMEnrollSecretUsedByOrbit(ctx, device.ID)
	require.NoError(t, err)
	require.False(t, usedByOrbit)
	h := newOneTimeSecretTestHost(t, ds, "windows", nil)
	_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(stored.ID))...)
	require.NoError(t, err)
	usedByOrbit, err = ds.WindowsMDMEnrollSecretUsedByOrbit(ctx, device.ID)
	require.NoError(t, err)
	require.True(t, usedByOrbit, "orbit enrolling with the enrollment's secret means fleetd is installed")
	secret, err = ds.GetLiveWindowsMDMOneTimeEnrollSecret(ctx, device.ID)
	require.NoError(t, err)
	require.Empty(t, secret)
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
	rotated := liveWindowsSecret(t, ds, device.ID)
	require.NotEqual(t, stored.Secret, rotated.Secret)

	// The set-based core, which the batch resend uses: unknown enrollments are skipped rather than failing the batch, and only
	// the enrollments without a live secret get one.
	other := insertWindowsEnrollment(t, ds, "hw-mint-other", "")
	var result windowsOneTimeEnrollSecretMintResult
	require.NoError(t, ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		result, err = mintWindowsMDMOneTimeEnrollSecretsDB(ctx, tx, []uint{device.ID, other.ID, device.ID + 9999})
		return err
	}))
	require.Equal(t, windowsOneTimeEnrollSecretMintResult{found: 2, inserted: 1}, result, "device already had a live secret")

	// The deleted-host push queues once per live secret, and a failed queue leaves no secret behind for later sessions to mistake
	// for one already sent.
	const pushLocURI = "./Device/qa"
	pushCmd := func(commandUUID string) *fleet.MDMWindowsCommand {
		return &fleet.MDMWindowsCommand{CommandUUID: commandUUID, RawCommand: []byte("<Replace/>"), TargetLocURI: pushLocURI}
	}
	requirePushed := func(enrollmentID uint, want bool) {
		got, err := ds.WindowsMDMEnrollSecretPushed(ctx, enrollmentID, pushLocURI)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	pushed := insertWindowsEnrollment(t, ds, "hw-mint-pushed", "")
	queued, err := ds.QueueWindowsMDMEnrollSecretPush(ctx, pushed.ID, pushed.MDMDeviceID, pushCmd("push-1"), nil)
	require.NoError(t, err)
	require.True(t, queued)
	queued, err = ds.QueueWindowsMDMEnrollSecretPush(ctx, pushed.ID, pushed.MDMDeviceID, pushCmd("push-2"), nil)
	require.NoError(t, err)
	require.False(t, queued, "the device was already sent the live secret")

	// A secret minted for a fleetd install was never pushed, so the push reuses it and queues the install alongside.
	installed := insertWindowsEnrollment(t, ds, "hw-mint-installed", "")
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, installed.ID))
	requirePushed(installed.ID, false)
	installCmd := &fleet.MDMWindowsCommand{CommandUUID: "install-1", RawCommand: []byte("<Exec/>"), TargetLocURI: "./Device/install"}
	queued, err = ds.QueueWindowsMDMEnrollSecretPush(ctx, installed.ID, installed.MDMDeviceID, pushCmd("push-3"), installCmd)
	require.NoError(t, err)
	require.True(t, queued)
	require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, installed.ID), "the install's secret is reused")
	pending, err := ds.MDMWindowsGetPendingCommands(ctx, installed.ID)
	require.NoError(t, err)
	require.Len(t, pending, 2)

	// The device answers the push: its result replaces the queued entry. A failed push means the value was never written, so it
	// goes out again, reusing the live secret; a successful one counts as delivered.
	deliver := func(commandUUID, status string) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response_gz) VALUES (?, '')`, installed.ID)
			if err != nil {
				return err
			}
			responseID, _ := res.LastInsertId()
			if _, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code)
				VALUES (?, ?, '', ?, ?)`, installed.ID, commandUUID, responseID, status); err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, `DELETE FROM windows_mdm_command_queue WHERE enrollment_id = ?`, installed.ID)
			return err
		})
	}
	deliver("push-3", "500")
	requirePushed(installed.ID, false)
	queued, err = ds.QueueWindowsMDMEnrollSecretPush(ctx, installed.ID, installed.MDMDeviceID, pushCmd("push-3b"), nil)
	require.NoError(t, err)
	require.True(t, queued)
	require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, installed.ID), "the re-push carries the same secret")
	deliver("push-3b", "200")
	requirePushed(installed.ID, true)

	// A push queued before the live secret was minted delivered an older secret.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE windows_mdm_commands c, host_one_time_enroll_secrets s
			SET c.created_at = NOW() - INTERVAL 1 HOUR, s.consumed_at = NOW(6)
			WHERE c.command_uuid = 'push-3b' AND s.mdm_windows_enrollment_id = ?`, installed.ID)
		return err
	})
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, installed.ID))
	requirePushed(installed.ID, false)

	// The install is queued in the same transaction, so its failure takes the push and the secret with it.
	failed := insertWindowsEnrollment(t, ds, "hw-mint-failed", "")
	_, err = ds.QueueWindowsMDMEnrollSecretPush(ctx, failed.ID, failed.MDMDeviceID, pushCmd("push-4"), installCmd)
	require.Error(t, err, "the install command UUID is taken")
	require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, failed.ID))

	// Re-enrollment deletes the enrollment row, and the foreign key cascade is what invalidates the secret minted for it.
	_, err = ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, device.MDMHardwareID)
	require.NoError(t, err)
	_, err = ds.GetHostOneTimeEnrollSecret(ctx, rotated.Secret)
	require.True(t, fleet.IsNotFound(err), "the secret must go with its enrollment")
}

func testOneTimeEnrollSecretWindowsHostBinding(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	newWindowsHost := func(hostUUID, osqueryHostID string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(), PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			OsqueryHostID: new(osqueryHostID), NodeKey: new("nk-" + osqueryHostID), UUID: hostUUID,
			Hostname: "host-" + osqueryHostID, Platform: "windows",
		})
		require.NoError(t, err)
		return h
	}

	t.Run("a linked enrollment binds to its host and only that host can use it", func(t *testing.T) {
		victim := newOneTimeSecretTestHost(t, ds, "windows", nil)
		other := newOneTimeSecretTestHost(t, ds, "windows", nil)
		// Lowercase on purpose: the enrollment and the hosts row need not agree on case.
		device := insertWindowsEnrollment(t, ds, "hw-bind-linked", strings.ToLower(victim.UUID))
		require.Equal(t, victim.ID, *device.LinkedHostID, "the session's enrollment load finds the linked host")

		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		row := liveWindowsSecret(t, ds, device.ID)
		require.Equal(t, victim.ID, *row.HostID)
		require.Equal(t, victim.UUID, row.HardwareUUID)

		// MatchesHost skips a UUID the agent does not present, so what stops another machine is consumption: its enrollment
		// lands on a row that is not the bound host, and while the bound host exists that is refused, on both planes.
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(other, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(other, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)

		require.Equal(t, row.Secret, liveWindowsSecret(t, ds, device.ID).Secret, "a refused attempt must not spend the secret")

		// The host it was minted for can use it.
		enrolled, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(victim, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		require.Equal(t, victim.ID, enrolled.ID)
	})

	t.Run("an unbound secret is bound once its host is known", func(t *testing.T) {
		// A first-install secret that was never used, then the enrollment gets linked, then an administrator resends. Reuse
		// keeps the same secret, and it must not stay unbound through that.
		host := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-bind-reuse", "")
		require.Nil(t, device.LinkedHostID)
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		before := liveWindowsSecret(t, ds, device.ID)

		linkWindowsEnrollment(t, ds, device, host.UUID)
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		after := liveWindowsSecret(t, ds, device.ID)
		require.Equal(t, before.Secret, after.Secret, "reused, not replaced")
		require.Equal(t, host.ID, *after.HostID)
		require.Equal(t, host.UUID, after.HardwareUUID)
	})

	t.Run("duplicate UUIDs bind to the row orbit will land on", func(t *testing.T) {
		// Binding to any other row would make check 2 refuse the legitimate host.
		shared := strings.ToUpper(uuid.NewString())
		_ = newWindowsHost(shared, "instance-"+shared[:8])
		identifierMatch := newWindowsHost(shared, shared)
		device := insertWindowsEnrollment(t, ds, "hw-bind-dup", shared)
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		require.Equal(t, identifierMatch.ID, *liveWindowsSecret(t, ds, device.ID).HostID, "the osquery_host_id match wins over the lower id")

		sharedAgain := strings.ToUpper(uuid.NewString())
		lowest := newWindowsHost(sharedAgain, "instance-a-"+sharedAgain[:8])
		_ = newWindowsHost(sharedAgain, "instance-b-"+sharedAgain[:8])
		deviceAgain := insertWindowsEnrollment(t, ds, "hw-bind-dup-again", sharedAgain)
		require.Equal(t, lowest.ID, *deviceAgain.LinkedHostID, "hosts sharing a UUID must not multiply the enrollment row")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, deviceAgain.ID))
		require.Equal(t, lowest.ID, *liveWindowsSecret(t, ds, deviceAgain.ID).HostID, "with no identifier match, the lowest id")
	})

	t.Run("a spent secret is swept by host once a resend supersedes it", func(t *testing.T) {
		// The first-install secret is minted unbound, and consuming it records the host. That is what lets the host-keyed sweep
		// the Apple path uses cover Windows too.
		h := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-bind-sweep", "")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		spent := liveWindowsSecret(t, ds, device.ID)
		_, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(spent.ID))...)
		require.NoError(t, err)
		consumed, err := ds.GetHostOneTimeEnrollSecret(ctx, spent.Secret)
		require.NoError(t, err)
		require.Equal(t, h.ID, *consumed.HostID)

		linkWindowsEnrollment(t, ds, device, h.UUID)
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		live := liveWindowsSecret(t, ds, device.ID)

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_one_time_enroll_secrets SET consumed_at = NOW(6) - INTERVAL 2 HOUR WHERE id = ?`, spent.ID)
			return err
		})
		_, err = ds.CleanupHostOneTimeEnrollSecrets(ctx)
		require.NoError(t, err)
		_, err = ds.GetHostOneTimeEnrollSecret(ctx, spent.Secret)
		require.True(t, fleet.IsNotFound(err), "the superseded secret must be swept")
		_, err = ds.GetHostOneTimeEnrollSecret(ctx, live.Secret)
		require.NoError(t, err, "the live secret must survive the sweep")
	})

	t.Run("a first-install secret cannot take over a host another enrollment claims", func(t *testing.T) {
		// The attacker's device is legitimately enrolled in MDM, so it is handed a first-install secret over its own channel,
		// but its orbit presents the victim's identifiers. Nothing binds that secret to a host.
		victim := newOneTimeSecretTestHost(t, ds, "windows", nil)
		insertWindowsEnrollment(t, ds, "hw-claim-victim", victim.UUID)
		attacker := insertWindowsEnrollment(t, ds, "hw-claim-attacker", "")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, attacker.ID))
		row := liveWindowsSecret(t, ds, attacker.ID)
		before, err := ds.Host(ctx, victim.ID)
		require.NoError(t, err)

		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(victim, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(victim, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)

		after, err := ds.Host(ctx, victim.ID)
		require.NoError(t, err)
		require.Equal(t, before.OrbitNodeKey, after.OrbitNodeKey, "the refused enrollment must not rotate the victim's node key")
		require.Equal(t, before.NodeKey, after.NodeKey)
		require.Equal(t, row.Secret, liveWindowsSecret(t, ds, attacker.ID).Secret, "a refused attempt must not spend the secret")
	})

	t.Run("a first-install secret cannot take over a host that isn't Windows", func(t *testing.T) {
		// A Mac has no Windows MDM enrollment to claim it, and the agent reports its own platform, so the stored platform decides.
		victim := newOneTimeSecretTestHost(t, ds, "darwin", nil)
		attacker := insertWindowsEnrollment(t, ds, "hw-mac-attacker", "")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, attacker.ID))
		row := liveWindowsSecret(t, ds, attacker.ID)
		before, err := ds.Host(ctx, victim.ID)
		require.NoError(t, err)

		asWindows := *victim
		asWindows.Platform = "windows"
		_, err = ds.EnrollOrbit(ctx, orbitEnrollOpts(&asWindows, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		requireEnrollmentRejected(t, err, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, &victim.ID)

		after, err := ds.Host(ctx, victim.ID)
		require.NoError(t, err)
		require.Equal(t, before.OrbitNodeKey, after.OrbitNodeKey, "the refused enrollment must not rotate the victim's node key")
		require.Equal(t, row.Secret, liveWindowsSecret(t, ds, attacker.ID).Secret, "a refused attempt must not spend the secret")
	})

	t.Run("the secret's own enrollment is not a competing claim", func(t *testing.T) {
		// A deleted host's device, recovered through the push: its enrollment is still linked to the old UUID, and the secret
		// minted for that enrollment recreates the host.
		h := newOneTimeSecretTestHost(t, ds, "windows", nil)
		device := insertWindowsEnrollment(t, ds, "hw-claim-own", h.UUID)
		require.NoError(t, ds.DeleteHost(ctx, h.ID))
		reloaded, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID)
		require.NoError(t, err)
		require.Nil(t, reloaded.LinkedHostID, "the deleted host leaves its enrollment linked by UUID, with no host behind it")
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, device.ID))
		row := liveWindowsSecret(t, ds, device.ID)
		require.Nil(t, row.HostID)

		recreated, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, nil, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		require.Equal(t, h.UUID, recreated.UUID)
		_, err = ds.EnrollOsquery(ctx, osqueryEnrollOpts(h, nil, fleet.WithEnrollOsqueryOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
	})
}

func testOneTimeEnrollSecretWindowsResendMints(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	secretProfileSyncML := []byte(`<Add><Item><Target><LocURI>` +
		`./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetdEnrollSecret/Policy/FleetdEnrollSecretAdmx` +
		`</LocURI></Target></Item></Add>`)
	secretProfile, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name: fleetmdm.FleetWindowsEnrollSecretProfileName, SyncML: secretProfileSyncML,
	}, nil)
	require.NoError(t, err)
	secretProfileUUID := secretProfile.ProfileUUID
	byName, err := ds.ListMDMWindowsConfigProfilesByName(ctx, fleetmdm.FleetWindowsEnrollSecretProfileName)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	require.Equal(t, secretProfileUUID, byName[0].ProfileUUID)
	require.Nil(t, byName[0].TeamID, "no team is stored as 0 and reported as nil")

	// The host's profile list shows it, since resending it from the host is the recovery action.
	listedHostUUID := uuid.NewString()
	require.NoError(t, ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{{
		ProfileUUID: secretProfileUUID, ProfileName: fleetmdm.FleetWindowsEnrollSecretProfileName, HostUUID: listedHostUUID,
		CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Checksum: []byte("checksum"),
	}}))
	listed, err := ds.GetHostMDMWindowsProfiles(ctx, listedHostUUID)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	otherProfile, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name: "Custom settings", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Custom</LocURI></Target></Item></Replace>`),
	}, nil)
	require.NoError(t, err)

	// Each host already runs fleetd and is linked to its enrollment, which is the shape a resend always has: a profile only
	// reaches a host that has a hosts row.
	newLinkedHost := func(name string, status *fleet.MDMDeliveryStatus, profiles ...string) *fleet.MDMWindowsEnrolledDevice {
		hostUUID := newOneTimeSecretTestHost(t, ds, "windows", nil).UUID
		device := insertWindowsEnrollment(t, ds, "hw-"+name, hostUUID)
		var payloads []*fleet.MDMWindowsBulkUpsertHostProfilePayload
		for _, profileUUID := range profiles {
			payloads = append(payloads, &fleet.MDMWindowsBulkUpsertHostProfilePayload{
				ProfileUUID: profileUUID, ProfileName: name, HostUUID: hostUUID, CommandUUID: uuid.NewString(),
				OperationType: fleet.MDMOperationTypeInstall, Status: status, Checksum: []byte("checksum"),
			})
		}
		require.NoError(t, ds.BulkUpsertMDMWindowsHostProfiles(ctx, payloads))
		return device
	}

	t.Run("single resend", func(t *testing.T) {
		device := newLinkedHost("single", new(fleet.MDMDeliveryVerified), secretProfileUUID, otherProfile.ProfileUUID)

		// Routine resends of other profiles mint nothing.
		require.NoError(t, ds.ResendHostMDMProfile(ctx, device.HostUUID, otherProfile.ProfileUUID))
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, device.ID))

		// Resending the enroll secret profile is the recovery action, so the host gets a secret to be delivered.
		require.NoError(t, ds.ResendHostMDMProfile(ctx, device.HostUUID, secretProfileUUID))
		liveWindowsSecret(t, ds, device.ID)
	})

	t.Run("only the enrollment the profile reaches is minted for", func(t *testing.T) {
		// Delivery goes to the host's most recent enrollment, so a secret for an older row would never reach a device and would
		// sit as a live credential nobody holds.
		older := newLinkedHost("two-enrollments", new(fleet.MDMDeliveryVerified), secretProfileUUID)
		newer := insertWindowsEnrollment(t, ds, "hw-two-enrollments-newer", older.HostUUID)
		require.NoError(t, ds.ResendHostMDMProfile(ctx, older.HostUUID, secretProfileUUID))
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, older.ID))
		require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, newer.ID))

		olderBatch := newLinkedHost("two-enrollments-batch", new(fleet.MDMDeliveryFailed), secretProfileUUID)
		newerBatch := insertWindowsEnrollment(t, ds, "hw-two-enrollments-batch-newer", olderBatch.HostUUID)
		_, err := ds.BatchResendMDMProfileToHosts(ctx, secretProfileUUID,
			fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
		require.NoError(t, err)
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, olderBatch.ID))
		require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, newerBatch.ID))
	})

	t.Run("batch resend", func(t *testing.T) {
		failed := newLinkedHost("batch-failed", new(fleet.MDMDeliveryFailed), secretProfileUUID)
		verified := newLinkedHost("batch-verified", new(fleet.MDMDeliveryVerified), secretProfileUUID)
		// Status NULL is a first delivery that is merely pending. After the reset it looks exactly like a resend target, which
		// is why the targets are read before it: minting here would hand a host that runs fleetd a secret it never needed.
		pending := newLinkedHost("batch-pending", nil, secretProfileUUID)

		count, err := ds.BatchResendMDMProfileToHosts(ctx, secretProfileUUID,
			fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
		require.NoError(t, err)
		require.EqualValues(t, 1, count)

		require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, failed.ID))
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, verified.ID), "not a target of this batch")
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, pending.ID), "pending is not a resend")

		// Other profiles' batch resends mint nothing.
		otherFailed := newLinkedHost("batch-other", new(fleet.MDMDeliveryFailed), otherProfile.ProfileUUID)
		_, err = ds.BatchResendMDMProfileToHosts(ctx, otherProfile.ProfileUUID,
			fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
		require.NoError(t, err)
		require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, otherFailed.ID))
	})
}

func testOneTimeEnrollSecretWindowsDeletedHostFleet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	newTeam := func(name string) *uint {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "deleted-host-" + name})
		require.NoError(t, err)
		return &team.ID
	}
	previous, defaultFleet, other := newTeam("previous"), newTeam("default"), newTeam("other")

	// deleteHost deletes a Windows MDM host whose enrollment survives.
	deleteHost := func(t *testing.T, name string, teamID *uint) (*fleet.Host, *fleet.MDMWindowsEnrolledDevice) {
		h := newOneTimeSecretTestHost(t, ds, "windows", teamID)
		device := insertWindowsEnrollment(t, ds, "hw-fleet-"+name, h.UUID)
		require.NoError(t, ds.DeleteHost(ctx, h.ID))
		return h, device
	}
	// nextSecret is the secret the next push to the enrollment carries.
	nextSecret := func(t *testing.T, enrollmentID uint) *fleet.HostOneTimeEnrollSecret {
		require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, enrollmentID))
		return liveWindowsSecret(t, ds, enrollmentID)
	}
	recordedTeam := func(t *testing.T, enrollmentID uint) *uint {
		var teamID *uint
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &teamID, `SELECT deleted_host_team_id FROM mdm_windows_enrollments WHERE id = ?`, enrollmentID)
		})
		return teamID
	}
	// sharingHost has no hostname, so it counts as an incoming host until it reports details.
	sharingHost := func(t *testing.T, platform, hostUUID, osqueryHostID string, teamID *uint) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(), PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			OsqueryHostID: new(osqueryHostID), NodeKey: new("nk-" + osqueryHostID), UUID: hostUUID, Platform: platform, TeamID: teamID,
		})
		require.NoError(t, err)
		return h
	}

	for _, tc := range []struct {
		name          string
		teamID        *uint
		deleteTeam    bool
		notRecorded   bool
		defaultFleet  *uint
		wantSecretFor *uint
	}{
		{name: "back to its fleet", teamID: previous, wantSecretFor: previous},
		{name: "fleet deleted since", teamID: newTeam("gone"), deleteTeam: true, defaultFleet: defaultFleet, wantSecretFor: defaultFleet},
		{name: "no fleet stays unassigned", defaultFleet: defaultFleet},
		{name: "no fleet and no default fleet"},
		{name: "fleet not recorded", teamID: previous, notRecorded: true, defaultFleet: defaultFleet, wantSecretFor: defaultFleet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, ds.SetWindowsEnrollmentDefaultFleet(ctx, tc.defaultFleet))
			h, device := deleteHost(t, tc.name, tc.teamID)
			if tc.deleteTeam {
				require.NoError(t, ds.DeleteTeam(ctx, *tc.teamID))
			}
			if tc.notRecorded {
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `UPDATE mdm_windows_enrollments SET deleted_host_team_id = NULL WHERE id = ?`, device.ID)
					return err
				})
			}
			require.Equal(t, tc.wantSecretFor, nextSecret(t, device.ID).TeamID)

			// The host comes back with the same UUID: a new link until the marker is cleared.
			updated, err := ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, h.UUID, device.MDMDeviceID)
			require.NoError(t, err)
			require.Equal(t, !tc.notRecorded, updated)
			require.NoError(t, ds.MDMWindowsClearDeletedHostTeam(ctx, device.MDMDeviceID))
			require.Nil(t, recordedTeam(t, device.ID))
			updated, err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, h.UUID, device.MDMDeviceID)
			require.NoError(t, err)
			require.False(t, updated)
		})
	}

	t.Run("a pending Autopilot host moves to the secret's fleet", func(t *testing.T) {
		// Autopilot re-creates a deleted device as a pending host in the default fleet, and orbit claims it by serial.
		require.NoError(t, ds.SetWindowsEnrollmentDefaultFleet(ctx, defaultFleet))
		h, device := deleteHost(t, "autopilot", previous)
		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{autopilotDevice(h.HardwareSerial, "")}))
		pending := hostIDsBySerial(t, ds, h.HardwareSerial)
		require.Len(t, pending, 1)

		row := nextSecret(t, device.ID)
		claimed, err := ds.EnrollOrbit(ctx, orbitEnrollOpts(h, row.TeamID, fleet.WithEnrollOrbitOneTimeEnrollSecret(row.ID))...)
		require.NoError(t, err)
		require.Equal(t, pending[0], claimed.ID)
		stored, err := ds.HostLite(ctx, claimed.ID)
		require.NoError(t, err)
		require.Equal(t, previous, stored.TeamID)
	})

	t.Run("a dual-boot Mac sharing the UUID is not the enrollment's host", func(t *testing.T) {
		windows := newOneTimeSecretTestHost(t, ds, "windows", previous)
		device := insertWindowsEnrollment(t, ds, "hw-fleet-dual-boot", windows.UUID)
		mac := sharingHost(t, "darwin", windows.UUID, "mac-"+windows.UUID, other)
		require.NoError(t, ds.DeleteHost(ctx, windows.ID))

		enrolled, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID)
		require.NoError(t, err)
		require.Nil(t, enrolled.LinkedHostID)
		row := nextSecret(t, device.ID)
		require.Nil(t, row.HostID)
		require.Equal(t, previous, row.TeamID)

		// Deleting the Mac later leaves the Windows host's fleet in place.
		require.NoError(t, ds.DeleteHost(ctx, mac.ID))
		require.Equal(t, previous, recordedTeam(t, device.ID))
	})

	// Among Windows hosts sharing the UUID, the one whose osquery_host_id is the UUID decides, as it is the one a secret binds to,
	// even when it isn't the lowest id. Both deletion paths follow that rule.
	sharedUUIDHosts := func(t *testing.T, name string) (first, owner *fleet.Host, device *fleet.MDMWindowsEnrolledDevice) {
		hostUUID := strings.ToUpper(uuid.NewString())
		sharingHost(t, "darwin", hostUUID, "mac-"+hostUUID, other)
		first = sharingHost(t, "windows", hostUUID, "other-"+hostUUID, other)
		owner = sharingHost(t, "windows", hostUUID, hostUUID, previous)
		return first, owner, insertWindowsEnrollment(t, ds, "hw-fleet-"+name, hostUUID)
	}
	t.Run("deleting Windows hosts sharing the UUID records the secret's host's fleet", func(t *testing.T) {
		first, owner, device := sharedUUIDHosts(t, "shared")
		require.NoError(t, ds.DeleteHosts(ctx, []uint{first.ID, owner.ID}))
		require.Equal(t, previous, recordedTeam(t, device.ID))
	})
	t.Run("incoming-host cleanup records the secret's host's fleet", func(t *testing.T) {
		_, _, device := sharedUUIDHosts(t, "incoming")
		_, err := ds.CleanupIncomingHosts(ctx, time.Now().Add(10*time.Minute))
		require.NoError(t, err)
		require.Equal(t, previous, recordedTeam(t, device.ID))
	})
}

func testOneTimeEnrollSecretWindowsDeleteUnusedSecrets(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// The secret minted for Fleet's fleetd install, before the enrollment linked to a host.
	unused := insertWindowsEnrollment(t, ds, "hw-unused", "")
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, unused.ID))
	require.NoError(t, ds.DeleteUnusedWindowsMDMOneTimeEnrollSecrets(ctx, unused.ID))
	require.Zero(t, countWindowsOneTimeEnrollSecrets(t, ds, unused.ID))

	// A used secret, and one bound to a host, like one an administrator resends, are kept.
	used := insertWindowsEnrollment(t, ds, "hw-used", "")
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, used.ID))
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_one_time_enroll_secrets SET consumed_at = NOW(6) WHERE mdm_windows_enrollment_id = ?`, used.ID)
		return err
	})
	h := newOneTimeSecretTestHost(t, ds, "windows", nil)
	bound := insertWindowsEnrollment(t, ds, "hw-bound", h.UUID)
	require.NoError(t, ds.MintWindowsMDMOneTimeEnrollSecret(ctx, bound.ID))
	require.NotNil(t, liveWindowsSecret(t, ds, bound.ID).HostID)
	for _, enrollmentID := range []uint{used.ID, bound.ID} {
		require.NoError(t, ds.DeleteUnusedWindowsMDMOneTimeEnrollSecrets(ctx, enrollmentID))
		require.Equal(t, 1, countWindowsOneTimeEnrollSecrets(t, ds, enrollmentID))
	}
}
