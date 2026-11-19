package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// GetConditionalAccessCertHostIDBySerialNumber retrieves the host_id for a valid certificate by serial number.
// This is a lightweight method optimized for authentication flows.
func (ds *Datastore) GetConditionalAccessCertHostIDBySerialNumber(ctx context.Context, serial uint64) (uint, error) {
	stmt := `
		SELECT host_id
		FROM conditional_access_scep_certificates
		WHERE serial = ? AND revoked = 0 AND not_valid_after > NOW()
	`
	var hostID uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostID, stmt, serial)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, notFound("ConditionalAccessCertificate"))
		}
		return 0, ctxerr.Wrap(ctx, err, "get conditional access cert host_id by serial")
	}
	return hostID, nil
}

// GetConditionalAccessCertCreatedAtByHostID retrieves the created_at timestamp of the most recent certificate for a host.
// This is a lightweight method for rate limiting checks.
func (ds *Datastore) GetConditionalAccessCertCreatedAtByHostID(ctx context.Context, hostID uint) (*time.Time, error) {
	stmt := `
		SELECT created_at
		FROM conditional_access_scep_certificates
		WHERE host_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var createdAt time.Time
	err := sqlx.GetContext(ctx, ds.reader(ctx), &createdAt, stmt, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("ConditionalAccessCertificate"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get conditional access cert created_at by host ID")
	}
	return &createdAt, nil
}

// RevokeOldConditionalAccessCerts revokes old certificates for hosts that have a newer certificate.
// It only revokes certificates where the newer certificate is older than the grace period.
// This prevents authentication failures during certificate rotation.
// Returns the number of certificates revoked.
func (ds *Datastore) RevokeOldConditionalAccessCerts(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	// Find and revoke old certificates where a newer certificate exists for the same host,
	// and the newer certificate is older than the grace period (to ensure it's stable)
	//
	// Explanation:
	// 1. Find the newest "stable" cert for each host (stable = issued before grace period)
	// 2. Revoke all certs with serial < newest stable serial for that host
	stmt := `
		UPDATE conditional_access_scep_certificates old_certs
		INNER JOIN (
			SELECT host_id, MAX(serial) as newest_stable_serial
			FROM conditional_access_scep_certificates
			WHERE not_valid_before < DATE_SUB(NOW(6), INTERVAL ? SECOND)
			  AND revoked = 0
			GROUP BY host_id
		) stable_certs ON old_certs.host_id = stable_certs.host_id
		SET old_certs.revoked = 1, old_certs.updated_at = NOW(6)
		WHERE old_certs.serial < stable_certs.newest_stable_serial
		  AND old_certs.revoked = 0
	`

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, int(gracePeriod.Seconds()))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "revoke old conditional access certificates")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get rows affected")
	}

	return rowsAffected, nil
}

// newConditionalAccessSCEPChallengeToken returns a challenge in hex rather than the URL-safe base64 of the Apple
// challenges: Windows sends the challenge as a PrintableString, which has no '_'.
func newConditionalAccessSCEPChallengeToken() ([]byte, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, err
	}
	return hex.AppendEncode(nil, token[:]), nil
}

func (ds *Datastore) FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx context.Context, enrollmentID uint) (string, error) {
	var challenge string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hostUUID string
		if err := sqlx.GetContext(ctx, tx, &hostUUID, `SELECT host_uuid FROM mdm_windows_enrollments WHERE id = ? FOR UPDATE`, enrollmentID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrollment").WithID(enrollmentID), "creating conditional access scep challenge")
			}
			return ctxerr.Wrap(ctx, err, "locking windows enrollment for conditional access scep challenge")
		}
		if hostUUID == "" {
			return ctxerr.Errorf(ctx, "windows enrollment %d has no host to bind a conditional access scep challenge to", enrollmentID)
		}
		var err error
		challenge, err = findOrCreateSCEPChallengeDB(ctx, tx, hostUUID, fleet.ConditionalAccessSCEPChallengePurpose,
			fleet.ConditionalAccessSCEPChallengeTTL, newConditionalAccessSCEPChallengeToken)
		return err
	})
	return challenge, err
}

func (ds *Datastore) ConsumeConditionalAccessSCEPChallenge(ctx context.Context, challenge, hostUUID string) error {
	if challenge == "" || hostUUID == "" {
		return fleet.NewInvalidArgumentError("challenge", "challenge and host UUID cannot be empty")
	}
	// The host is part of the match so another host's attempt fails without using the challenge up.
	res, err := ds.writer(ctx).ExecContext(ctx, `UPDATE mdm_apple_scep_challenges
		SET consumed_at = NOW(6)
		WHERE challenge = ? AND purpose = ? AND host_uuid = ? AND consumed_at IS NULL AND expires_at > NOW(6)`,
		challenge, fleet.ConditionalAccessSCEPChallengePurpose, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "consume conditional access scep challenge")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "consume conditional access scep challenge")
	}
	if affected == 0 {
		return ctxerr.Wrap(ctx, notFound("conditional access SCEP challenge"))
	}
	return nil
}
