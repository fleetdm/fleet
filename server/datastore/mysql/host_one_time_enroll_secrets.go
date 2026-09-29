package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const hostOneTimeEnrollSecretColumns = `id, secret, host_id, team_id, platform, hardware_uuid, hardware_serial, created_at, consumed_at, orbit_used_at, osquery_used_at` // nolint:gosec // Not hardcoded credentials

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
	if _, err := tx.ExecContext(ctx,
		`UPDATE host_one_time_enroll_secrets SET `+column+` = NOW(6), consumed_at = COALESCE(consumed_at, NOW(6)) WHERE id = ?`, id); err != nil {
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
