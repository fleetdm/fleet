package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/jmoiron/sqlx"
)

const hostOneTimeEnrollSecretColumns = `id, secret, host_id, mdm_windows_enrollment_id, team_id, platform, hardware_uuid, hardware_serial, created_at, consumed_at, orbit_used_at, osquery_used_at` // nolint:gosec // Not hardcoded credentials

func (ds *Datastore) GetHostOneTimeEnrollSecret(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ctxerr.Wrap(ctx, notFound("HostOneTimeEnrollSecret"), "empty secret")
	}
	var s fleet.HostOneTimeEnrollSecret
	// Secrets are minted on the primary moments before an agent presents them,
	// so a replica read could miss a freshly minted row.
	err := sqlx.GetContext(ctx, ds.writer(ctx), &s,
		`SELECT `+hostOneTimeEnrollSecretColumns+` FROM host_one_time_enroll_secrets WHERE secret = ?`, secret)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("HostOneTimeEnrollSecret"), "no matching one-time enroll secret")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get one-time enroll secret")
	}
	return &s, nil
}

func deleteHostOneTimeEnrollSecrets(ctx context.Context, tx sqlx.ExtContext, hostID uint) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM host_one_time_enroll_secrets WHERE host_id = ?`, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host one-time enroll secrets")
	}
	return nil
}

// CleanupHostOneTimeEnrollSecrets deletes, in batches, spent secrets that a
// newer secret for the same host has superseded (once past the second-plane
// window, so a late osquery enrollment is not orphaned) and secrets whose host
// no longer exists. Unconsumed secrets whose host records still exist are never
// removed here: a host holds at most one and it stays valid until used.
//
// Windows secrets need no sweep of their own: consuming one records its host,
// and deleting the enrollment cascades to its secrets.
func (ds *Datastore) CleanupHostOneTimeEnrollSecrets(ctx context.Context) (int64, error) {
	const batchSize = 1000
	windowSeconds := int64(fleet.HostOneTimeEnrollSecretSecondPlaneWindow / time.Second)
	deletes := []struct {
		stmt string
		args []any
	}{
		{
			stmt: `DELETE FROM host_one_time_enroll_secrets WHERE id IN (
				SELECT id FROM (
					SELECT DISTINCT s.id
					FROM host_one_time_enroll_secrets s
					JOIN host_one_time_enroll_secrets newer ON newer.host_id = s.host_id AND newer.id > s.id
					WHERE s.consumed_at IS NOT NULL AND s.consumed_at < NOW(6) - INTERVAL ? SECOND
					LIMIT ?
				) AS superseded
			)`,
			args: []any{windowSeconds, batchSize},
		},
		{
			stmt: `DELETE FROM host_one_time_enroll_secrets WHERE id IN (
				SELECT id FROM (
					SELECT s.id
					FROM host_one_time_enroll_secrets s
					LEFT JOIN hosts h ON h.id = s.host_id
					WHERE s.host_id IS NOT NULL AND h.id IS NULL
					LIMIT ?
				) AS orphaned
			)`,
			args: []any{batchSize},
		},
	}

	var total int64
	for _, d := range deletes {
		for {
			res, err := ds.writer(ctx).ExecContext(ctx, d.stmt, d.args...)
			if err != nil {
				return total, ctxerr.Wrap(ctx, err, "cleanup host one-time enroll secrets")
			}
			n, _ := res.RowsAffected()
			total += n
			if n < batchSize {
				break
			}
		}
	}
	return total, nil
}

// mintHostOneTimeEnrollSecret returns the one-time enroll secret to embed in
// the fleetd configuration profile being delivered to the given MDM
// enrollment. A host has at most one unconsumed secret: if one exists it is
// returned again (the device may fetch the same command more than once, and
// re-deliveries, including an admin resend, must not churn a valid secret),
// otherwise a new one is minted and bound to the host's identifiers and team.
func (ds *Datastore) mintHostOneTimeEnrollSecret(ctx context.Context, enrollmentID string) (string, error) {
	// User-channel enrollment IDs are "<udid>:<userid>"; the fleetd profile is
	// device-scoped so this should never happen.
	if strings.Contains(enrollmentID, ":") {
		return "", ctxerr.Errorf(ctx, "one-time enroll secrets are only minted for device enrollments, got %q", enrollmentID)
	}

	var secret string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hosts []struct {
			ID             uint    `db:"id"`
			TeamID         *uint   `db:"team_id"`
			Platform       string  `db:"platform"`
			HardwareSerial string  `db:"hardware_serial"`
			OsqueryHostID  *string `db:"osquery_host_id"`
		}
		// FOR UPDATE on the hosts row serializes concurrent mints for the same
		// device (a NotNow re-serve or a refetch can race the first delivery).
		// Locking the secrets table alone does not work: two transactions that
		// both find no unconsumed row hold compatible gap locks and can both
		// insert.
		if err := sqlx.SelectContext(ctx, tx, &hosts,
			`SELECT id, team_id, platform, hardware_serial, osquery_host_id FROM hosts WHERE uuid = ? ORDER BY id FOR UPDATE`,
			enrollmentID); err != nil {
			return ctxerr.Wrap(ctx, err, "load host for one-time enroll secret")
		}
		if len(hosts) == 0 {
			return ctxerr.Wrap(ctx, notFound("Host").WithName(enrollmentID), "minting one-time enroll secret")
		}

		// Several rows can share a UUID (VM clones, or an orbit enrolled with
		// --host-identifier=instance next to the MDM-created row). Bind the secret
		// to the row the orbit enrollment will land on: matchHostDuringEnrollment
		// first tries osquery_host_id = <uuid> (unique, so at most one row), then
		// falls back to the lowest id for the serial match.
		host := hosts[0]
		for _, h := range hosts {
			if h.OsqueryHostID != nil && strings.EqualFold(*h.OsqueryHostID, enrollmentID) {
				host = h
				break
			}
		}
		if len(hosts) > 1 {
			ids := make([]uint, 0, len(hosts))
			for _, h := range hosts {
				ids = append(ids, h.ID)
			}
			ds.logger.WarnContext(ctx, "multiple hosts share a uuid, binding one-time enroll secret to the host orbit will match",
				"uuid", enrollmentID, "host_ids", ids, "chosen_host_id", host.ID)
		}

		var existing []string
		if err := sqlx.SelectContext(ctx, tx, &existing,
			`SELECT secret FROM host_one_time_enroll_secrets WHERE host_id = ? AND consumed_at IS NULL ORDER BY id DESC LIMIT 1`,
			host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "load unconsumed one-time enroll secret")
		}
		if len(existing) == 1 {
			secret = existing[0]
			return nil
		}

		tok, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generate one-time enroll secret")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO host_one_time_enroll_secrets
				(secret, host_id, team_id, platform, hardware_uuid, hardware_serial)
			VALUES (?, ?, ?, ?, ?, ?)`,
			string(tok), host.ID, host.TeamID, host.Platform, enrollmentID, host.HardwareSerial); err != nil {
			return ctxerr.Wrap(ctx, err, "insert one-time enroll secret")
		}
		secret = string(tok)
		return nil
	})
	if err != nil {
		return "", err
	}
	return secret, nil
}

// MintWindowsMDMOneTimeEnrollSecret makes sure the given Windows MDM enrollment has a live one-time enroll secret, for the
// fleetd install Fleet is about to send it.
func (ds *Datastore) MintWindowsMDMOneTimeEnrollSecret(ctx context.Context, enrollmentID uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		minted, err := mintWindowsMDMOneTimeEnrollSecretsDB(ctx, tx, []uint{enrollmentID})
		if err != nil {
			return err
		}
		if minted.found == 0 {
			return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"), "minting one-time enroll secret")
		}
		return nil
	})
}

type windowsOneTimeEnrollSecretMintResult struct {
	// found is how many of the requested enrollments exist.
	found int
	// inserted is how many of those needed a new secret, because they had no live one.
	inserted int
}

// mintWindowsMDMOneTimeEnrollSecretsDB gives each existing enrollment in enrollmentIDs a live secret, reusing an unconsumed one
// where there is one. Reuse is what keeps the installer command line and the profile carrying the same secret, and what keeps
// repeat calls harmless: Fleet re-enqueues the fleetd install on every session-start alert while fleetd looks absent.
//
// The FOR UPDATE serializes concurrent mints for the same enrollment. Without it, two sessions that both find no live secret
// would both insert, leaving two valid secrets for one device.
//
// When the enrollment is already linked to a host, which it always is on an administrator resend, the secret is bound to that
// host the way an Apple secret is: host_id and hardware_uuid come from the host row. That is what enforces that only that host
// can use it. MatchesHost compares the hardware UUID the agent presents, and consumeHostOneTimeEnrollSecret refuses an enrollment
// that lands on any other row while the bound host exists. The second check matters because the first skips an empty presented
// value. An existing unbound live secret, minted before the host existed, is bound as well, so it cannot stay unbound through a
// resend.
//
// Otherwise, as on a first fleetd install, there is no host to bind to: the automatic enrollment flows carry no Fleet host UUID.
// The secret binds only to the enrollment, host_id stays NULL, and the host is recorded on first use. The serial is copied when a
// DevDetail response already landed, as provenance.
func mintWindowsMDMOneTimeEnrollSecretsDB(
	ctx context.Context, tx sqlx.ExtContext, enrollmentIDs []uint,
) (windowsOneTimeEnrollSecretMintResult, error) {
	var result windowsOneTimeEnrollSecretMintResult
	if len(enrollmentIDs) == 0 {
		return result, nil
	}

	stmt, args, err := sqlx.In(
		`SELECT id, hardware_serial, host_uuid FROM mdm_windows_enrollments WHERE id IN (?) ORDER BY id FOR UPDATE`, enrollmentIDs)
	if err != nil {
		return result, ctxerr.Wrap(ctx, err, "build windows mdm enrollment lock for one-time enroll secrets")
	}
	var enrollments []windowsEnrollmentMintRow
	if err := sqlx.SelectContext(ctx, tx, &enrollments, stmt, args...); err != nil {
		return result, ctxerr.Wrap(ctx, err, "lock windows mdm enrollments for one-time enroll secrets")
	}
	result.found = len(enrollments)
	if len(enrollments) == 0 {
		return result, nil
	}

	lockedIDs := make([]uint, 0, len(enrollments))
	for _, e := range enrollments {
		lockedIDs = append(lockedIDs, e.ID)
	}
	stmt, args, err = sqlx.In(`
		SELECT DISTINCT mdm_windows_enrollment_id FROM host_one_time_enroll_secrets
		WHERE mdm_windows_enrollment_id IN (?) AND consumed_at IS NULL`, lockedIDs)
	if err != nil {
		return result, ctxerr.Wrap(ctx, err, "build live windows one-time enroll secret lookup")
	}
	var live []uint
	if err := sqlx.SelectContext(ctx, tx, &live, stmt, args...); err != nil {
		return result, ctxerr.Wrap(ctx, err, "load live windows one-time enroll secrets")
	}
	hasLive := make(map[uint]struct{}, len(live))
	for _, id := range live {
		hasLive[id] = struct{}{}
	}

	boundHosts, err := windowsEnrollmentBoundHostsDB(ctx, tx, enrollments)
	if err != nil {
		return result, err
	}
	for _, e := range enrollments {
		host, known := boundHosts[e.ID]
		if _, ok := hasLive[e.ID]; !ok || !known {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE host_one_time_enroll_secrets SET host_id = ?, hardware_uuid = ?
			WHERE mdm_windows_enrollment_id = ? AND consumed_at IS NULL AND host_id IS NULL`,
			host.ID, host.UUID, e.ID); err != nil {
			return result, ctxerr.Wrap(ctx, err, "bind live windows one-time enroll secret to its host")
		}
	}

	var (
		placeholders []string
		insertArgs   []any
	)
	for _, e := range enrollments {
		if _, ok := hasLive[e.ID]; ok {
			continue
		}
		tok, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
		if err != nil {
			return result, ctxerr.Wrap(ctx, err, "generate windows one-time enroll secret")
		}
		var serial string
		if e.HardwareSerial != nil {
			serial = *e.HardwareSerial
		}
		// team_id stays NULL on purpose. The secret replaces the global enroll secret, which also had no team, so the host still
		// enrolls into no team and the existing default-fleet transfer (maybeAssignWindowsEnrollmentDefaultFleet) applies
		// afterwards with its own guards intact. Binding a team here would silently bypass those.
		var (
			hostID       *uint
			hardwareUUID string
		)
		if host, known := boundHosts[e.ID]; known {
			hostID, hardwareUUID = &host.ID, host.UUID
		}
		placeholders = append(placeholders, `(?, ?, ?, NULL, 'windows', ?, ?)`)
		insertArgs = append(insertArgs, string(tok), hostID, e.ID, hardwareUUID, serial)
	}
	if len(placeholders) == 0 {
		return result, nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO host_one_time_enroll_secrets
			(secret, host_id, mdm_windows_enrollment_id, team_id, platform, hardware_uuid, hardware_serial)
		VALUES `+strings.Join(placeholders, ", "), insertArgs...); err != nil {
		return result, ctxerr.Wrap(ctx, err, "insert windows one-time enroll secrets")
	}
	result.inserted = len(placeholders)
	return result, nil
}

type windowsEnrollmentMintRow struct {
	ID             uint    `db:"id"`
	HardwareSerial *string `db:"hardware_serial"`
	HostUUID       string  `db:"host_uuid"`
}

type windowsEnrollmentBoundHost struct {
	ID   uint
	UUID string
}

// windowsEnrollmentBoundHostsDB returns, for each enrollment already linked to a host, the host row a one-time enroll secret for
// it should bind to. Enrollments with no host_uuid, or whose host no longer exists, are absent from the result.
//
// Several rows can share a UUID (VM clones, or an orbit enrolled with --host-identifier=instance). Like the Apple mint, this binds
// to the row the orbit enrollment will land on: the one whose osquery_host_id is the UUID, else the lowest id. Binding to any
// other row would make consumeHostOneTimeEnrollSecret refuse the legitimate host.
func windowsEnrollmentBoundHostsDB(
	ctx context.Context, tx sqlx.ExtContext, enrollments []windowsEnrollmentMintRow,
) (map[uint]windowsEnrollmentBoundHost, error) {
	var uuids []string
	for _, e := range enrollments {
		if e.HostUUID != "" {
			uuids = append(uuids, e.HostUUID)
		}
	}
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt, args, err := sqlx.In(`SELECT id, uuid, osquery_host_id FROM hosts WHERE uuid IN (?) ORDER BY id`, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build hosts lookup for windows one-time enroll secrets")
	}
	var hosts []struct {
		ID            uint    `db:"id"`
		UUID          string  `db:"uuid"`
		OsqueryHostID *string `db:"osquery_host_id"`
	}
	if err := sqlx.SelectContext(ctx, tx, &hosts, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load hosts for windows one-time enroll secrets")
	}

	// Keyed case-insensitively, because the enrollment's host_uuid and the hosts row need not agree on case.
	chosen := make(map[string]windowsEnrollmentBoundHost, len(hosts))
	for _, h := range hosts {
		key := strings.ToLower(h.UUID)
		_, seen := chosen[key]
		matchesIdentifier := h.OsqueryHostID != nil && strings.EqualFold(*h.OsqueryHostID, h.UUID)
		// Rows arrive in id order, so the first seen is the lowest id; an osquery_host_id match overrides it.
		if !seen || matchesIdentifier {
			chosen[key] = windowsEnrollmentBoundHost{ID: h.ID, UUID: h.UUID}
		}
	}

	bound := make(map[uint]windowsEnrollmentBoundHost, len(enrollments))
	for _, e := range enrollments {
		if h, ok := chosen[strings.ToLower(e.HostUUID)]; ok {
			bound[e.ID] = h
		}
	}
	return bound, nil
}

// liveWindowsMDMOneTimeEnrollSecret returns the unconsumed secret minted for the enrollment, or "" when there is none, which is
// the normal state for a host that already runs fleetd.
//
// It reads the primary. The fleetd install is minted for and delivered in the same management request (processNewSessionAlert
// runs before getPendingMDMCmds), so a replica even slightly behind would resolve to nothing and ship an installer without a
// secret.
func (ds *Datastore) liveWindowsMDMOneTimeEnrollSecret(ctx context.Context, enrollmentID uint) (string, error) {
	var secrets []string
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &secrets, `
		SELECT secret FROM host_one_time_enroll_secrets
		WHERE mdm_windows_enrollment_id = ? AND consumed_at IS NULL
		ORDER BY id DESC LIMIT 1`, enrollmentID); err != nil {
		return "", ctxerr.Wrap(ctx, err, "load live windows one-time enroll secret")
	}
	if len(secrets) == 0 {
		return "", nil
	}
	return secrets[0], nil
}

// isWindowsEnrollSecretProfileDB reports whether the profile is the Fleet-managed Fleetd enroll secret profile. The name is
// reserved, so a user cannot author one that passes this check.
//
// The resend hooks below key on this rather than on auth.mdm_windows_one_time_enroll_secrets, which the datastore does not see.
// The service refuses to resend this profile while the switch is off (errWindowsEnrollSecretProfileOff), and the reconciler
// deletes it, so these hooks only ever run with the switch on.
func isWindowsEnrollSecretProfileDB(ctx context.Context, tx sqlx.ExtContext, profileUUID string) (bool, error) {
	var names []string
	if err := sqlx.SelectContext(ctx, tx, &names,
		`SELECT name FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, profileUUID); err != nil {
		return false, ctxerr.Wrap(ctx, err, "load windows profile name")
	}
	return len(names) == 1 && names[0] == fleetmdm.FleetWindowsEnrollSecretProfileName, nil
}

// mintWindowsEnrollSecretOnResendDB mints for a host whose Fleetd enroll secret profile an administrator just resent. It runs in
// the transaction that resets the profile's status, so the reconciler cannot deliver the profile before the secret exists.
//
// It mints only for the enrollment the profile will actually reach. A host can have several enrollment rows, and profile delivery
// goes to the most recent one (getEnrollmentIDsByHostUUIDDB), so a secret for any older row would never be delivered and would sit
// as a live credential that no device holds. Reusing the delivery lookup keeps the two from drifting apart.
func (ds *Datastore) mintWindowsEnrollSecretOnResendDB(ctx context.Context, tx sqlx.ExtContext, hostUUID, profileUUID string) error {
	isSecretProfile, err := isWindowsEnrollSecretProfileDB(ctx, tx, profileUUID)
	if err != nil || !isSecretProfile {
		return err
	}
	enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDDB(ctx, tx, []string{hostUUID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load windows mdm enrollment for resent enroll secret profile")
	}
	_, err = mintWindowsMDMOneTimeEnrollSecretsDB(ctx, tx, enrollmentIDs)
	return err
}

// windowsEnrollSecretBatchResendTargetsDB returns the enrollments of the hosts a batch resend of the Fleetd enroll secret profile
// is about to reset, or nil for any other profile. It has to run before the status reset: afterwards the targets are
// indistinguishable from hosts whose first delivery was already pending, and those must not be minted for.
//
// FOR UPDATE makes this the same set the reset then updates, rather than a snapshot another transaction could add to. Each host
// maps to its most recent enrollment only, for the same reason as mintWindowsEnrollSecretOnResendDB.
func (ds *Datastore) windowsEnrollSecretBatchResendTargetsDB(
	ctx context.Context, tx sqlx.ExtContext, profileUUID string, status fleet.MDMDeliveryStatus,
) ([]uint, error) {
	isSecretProfile, err := isWindowsEnrollSecretProfileDB(ctx, tx, profileUUID)
	if err != nil || !isSecretProfile {
		return nil, err
	}
	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, `
		SELECT host_uuid FROM host_mdm_windows_profiles
		WHERE profile_uuid = ? AND status = ?
		FOR UPDATE`, profileUUID, status); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load hosts for batch-resent enroll secret profile")
	}
	enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDDB(ctx, tx, hostUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load windows mdm enrollments for batch-resent enroll secret profile")
	}
	return enrollmentIDs, nil
}

// consumeHostOneTimeEnrollSecret records the use of a one-time enroll secret
// by one enrollment plane, inside the enrollment transaction so the node key
// rotation and the consumption commit or roll back together. matchedHostID is
// the host row the enrollment landed on (a freshly inserted row is fine as
// long as the bound host no longer exists).
func consumeHostOneTimeEnrollSecret(ctx context.Context, tx sqlx.ExtContext, id uint, plane fleet.EnrollmentPlane, matchedHostID uint) error {
	// consumed_at is written with the database clock, so the window is judged
	// there too rather than against this server's clock.
	var s struct {
		fleet.HostOneTimeEnrollSecret
		OutsideWindow bool `db:"outside_window"`
	}
	err := sqlx.GetContext(ctx, tx, &s,
		`SELECT `+hostOneTimeEnrollSecretColumns+`,
			(consumed_at IS NOT NULL AND consumed_at < NOW(6) - INTERVAL ? SECOND) AS outside_window
		FROM host_one_time_enroll_secrets WHERE id = ? FOR UPDATE`,
		int64(fleet.HostOneTimeEnrollSecretSecondPlaneWindow/time.Second), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Deleted between lookup and consumption (admin rotation or MDM reset).
		return ctxerr.Wrap(ctx, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedOneTimeSecretSpent}, "one-time enroll secret no longer exists")
	case err != nil:
		return ctxerr.Wrap(ctx, err, "lock one-time enroll secret")
	}

	if s.HostID != nil && *s.HostID != matchedHostID {
		var boundExists bool
		if err := sqlx.GetContext(ctx, tx, &boundExists, `SELECT EXISTS (SELECT 1 FROM hosts WHERE id = ?)`, *s.HostID); err != nil {
			return ctxerr.Wrap(ctx, err, "check bound host for one-time enroll secret")
		}
		if boundExists {
			return ctxerr.Wrap(ctx, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, HostID: s.HostID},
				"one-time enroll secret bound to another host")
		}
	}

	if s.UsedAt(plane) != nil || s.OutsideWindow {
		return ctxerr.Wrap(ctx, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedOneTimeSecretSpent, HostID: s.HostID}, "one-time enroll secret already used")
	}

	var column string
	switch plane {
	case fleet.EnrollmentPlaneOrbit:
		column = "orbit_used_at"
	case fleet.EnrollmentPlaneOsquery:
		column = "osquery_used_at"
	default:
		return ctxerr.Errorf(ctx, "unknown enrollment plane %q", plane)
	}
	// host_id is recorded on first use, which matters only for an enrollment-bound (Windows) secret: it is minted before a
	// hosts row exists, so without this the bound-host check above would have nothing to compare and the second plane could
	// be claimed from a different machine inside the window. COALESCE leaves an already-bound secret alone.
	if _, err := tx.ExecContext(ctx,
		`UPDATE host_one_time_enroll_secrets
		 SET `+column+` = NOW(6), consumed_at = COALESCE(consumed_at, NOW(6)), host_id = COALESCE(host_id, ?)
		 WHERE id = ?`, matchedHostID, id); err != nil {
		return ctxerr.Wrap(ctx, err, "consume one-time enroll secret")
	}
	return nil
}

// rejectSharedSecretForMDMManagedAppleHost refuses a shared enroll secret for a
// matched Apple host that is enrolled in Fleet MDM or assigned to Fleet in Apple
// Business Manager: such hosts get a one-time secret through the fleetd
// configuration profile and must enroll with it. Only a matched row is checked.
// A deleted, manually enrolled Mac is recreated only by fleetd enrolling again
// (MDM check-in recreates iOS/iPadOS hosts only), so the insert branch stays
// open to shared secrets for that host.
func rejectSharedSecretForMDMManagedAppleHost(ctx context.Context, tx sqlx.ExtContext, hostID uint, platform string) error {
	if !fleet.IsApplePlatform(platform) {
		return nil
	}
	var managed bool
	err := sqlx.GetContext(ctx, tx, &managed, `
		SELECT EXISTS (
			SELECT 1 FROM hosts h JOIN nano_enrollments ne ON ne.id = h.uuid AND ne.enabled = 1 WHERE h.id = ?
		) OR EXISTS (
			SELECT 1 FROM host_dep_assignments hda WHERE hda.host_id = ? AND hda.deleted_at IS NULL
		)`, hostID, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check MDM management of matched host")
	}
	if managed {
		return ctxerr.Wrap(ctx, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, HostID: &hostID},
			"shared enroll secret presented for MDM-managed Apple host")
	}
	return nil
}

// rejectSharedSecretForMDMManagedWindowsHost refuses a shared enroll secret for a matched Windows host that is enrolled in Fleet MDM,
// so that another machine cannot take the host over by presenting its identifiers. Such a host gets a one-time secret through the
// Fleetd enroll secret profile, which an admin resends when fleetd has to enroll again. An enrollment row linked to the host is the
// signal: an unenroll alert or a re-enrollment of the device deletes it. As on Apple, only a matched row is checked, so a deleted
// host can still come back with a shared secret.
func rejectSharedSecretForMDMManagedWindowsHost(ctx context.Context, tx sqlx.ExtContext, hostID uint, platform string) error {
	if platform != "windows" {
		return nil
	}
	var managed bool
	err := sqlx.GetContext(ctx, tx, &managed, `
		SELECT EXISTS (
			SELECT 1 FROM hosts h JOIN mdm_windows_enrollments mwe ON mwe.host_uuid = h.uuid WHERE h.id = ? AND h.uuid != ''
		)`, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check Windows MDM management of matched host")
	}
	if managed {
		return ctxerr.Wrap(ctx, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, HostID: &hostID},
			"shared enroll secret presented for MDM-managed Windows host")
	}
	return nil
}
