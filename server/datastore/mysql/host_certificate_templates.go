package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ListAndroidHostUUIDsWithDeliverableCertificateTemplates returns a batch of host UUIDs that have certificate templates to deliver
func (ds *Datastore) ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx context.Context, offset int, limit int) ([]string, error) {
	const stmt = `
		SELECT DISTINCT
			hosts.uuid
		FROM certificate_templates
		INNER JOIN hosts ON hosts.team_id = certificate_templates.team_id
		INNER JOIN host_mdm ON host_mdm.host_id = hosts.id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.platform = 'android' AND
			host_mdm.enrolled = 1 AND
			host_certificate_templates.id IS NULL
		ORDER BY hosts.uuid
		LIMIT ? OFFSET ?
	`

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, limit, offset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android host uuids with certificate templates")
	}

	return hostUUIDs, nil
}

// ListCertificateTemplatesForHosts returns ALL certificate templates for the given host UUIDs
func (ds *Datastore) ListCertificateTemplatesForHosts(ctx context.Context, hostUUIDs []string) ([]fleet.CertificateTemplateForHost, error) {
	if len(hostUUIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT
			hosts.uuid AS host_uuid,
			certificate_templates.id AS certificate_template_id,
			host_certificate_templates.fleet_challenge AS fleet_challenge,
			host_certificate_templates.status AS status
		FROM certificate_templates
		INNER JOIN hosts ON hosts.team_id = certificate_templates.team_id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.uuid IN (?)
		ORDER BY hosts.uuid, certificate_templates.id
	`, hostUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build query for certificate templates")
	}

	var results []fleet.CertificateTemplateForHost
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list certificate templates for android hosts")
	}

	return results, nil
}

// GetCertificateTemplateForHost returns a certificate template for a specific host and certificate template ID
func (ds *Datastore) GetCertificateTemplateForHost(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
	const stmt = `
		SELECT
			hosts.uuid AS host_uuid,
			certificate_templates.id AS certificate_template_id,
			host_certificate_templates.fleet_challenge AS fleet_challenge,
			host_certificate_templates.status AS status,
			certificate_authorities.type AS ca_type,
			certificate_authorities.name AS ca_name
		FROM certificate_templates
		INNER JOIN hosts ON hosts.team_id = certificate_templates.team_id
		INNER JOIN certificate_authorities ON certificate_authorities.id = certificate_templates.certificate_authority_id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.uuid = ? AND certificate_templates.id = ?
	`

	var result fleet.CertificateTemplateForHost
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, hostUUID, certificateTemplateID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get certificate template for host")
	}

	return &result, nil
}

// BulkInsertHostCertificateTemplates inserts multiple host_certificate_templates records
func (ds *Datastore) BulkInsertHostCertificateTemplates(ctx context.Context, hostCertTemplates []fleet.HostCertificateTemplate) error {
	if len(hostCertTemplates) == 0 {
		return nil
	}

	const argsCount = 5

	const sqlInsert = `
		INSERT INTO host_certificate_templates (
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status,
			operation_type
		) VALUES %s
	`

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(hostCertTemplates)*argsCount)

	for _, hct := range hostCertTemplates {
		args = append(args, hct.HostUUID, hct.CertificateTemplateID, hct.FleetChallenge, hct.Status, hct.OperationType)
		placeholders.WriteString("(?,?,?,?,?),")
	}

	stmt := fmt.Sprintf(sqlInsert, strings.TrimSuffix(placeholders.String(), ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk insert host_certificate_templates")
	}

	return nil
}

// DeleteHostCertificateTemplates deletes specific host_certificate_templates records
// identified by (host_uuid, certificate_template_id) pairs.
func (ds *Datastore) DeleteHostCertificateTemplates(ctx context.Context, hostCertTemplates []fleet.HostCertificateTemplate) error {
	if len(hostCertTemplates) == 0 {
		return nil
	}

	// Build placeholders and args for tuple matching
	var placeholders strings.Builder
	args := make([]any, 0, len(hostCertTemplates)*2)

	for i, hct := range hostCertTemplates {
		if i > 0 {
			placeholders.WriteString(",")
		}
		placeholders.WriteString("(?,?)")
		args = append(args, hct.HostUUID, hct.CertificateTemplateID)
	}

	stmt := fmt.Sprintf(
		"DELETE FROM host_certificate_templates WHERE (host_uuid, certificate_template_id) IN (%s)",
		placeholders.String(),
	)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_certificate_templates")
	}

	return nil
}

func (ds *Datastore) UpsertCertificateStatus(
	ctx context.Context,
	hostUUID string,
	certificateTemplateID uint,
	status fleet.MDMDeliveryStatus,
	detail *string,
) error {
	updateStmt := `
    UPDATE host_certificate_templates
    SET status = ?, detail = ?
    WHERE host_uuid = ? AND certificate_template_id = ?`

	insertStmt := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, detail, fleet_challenge, operation_type)
		VALUES (?, ?, ?, ?, ?, ?)`

	// Validate the status.
	if !status.IsValid() {
		return ctxerr.Wrap(ctx, fmt.Errorf("Invalid status '%s'", string(status)))
	}

	// Attempt to update the certificate status for the given host and template.
	result, err := ds.writer(ctx).ExecContext(ctx, updateStmt, status, detail, hostUUID, certificateTemplateID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// If no records were updated, then insert a new status.
	if rowsAffected == 0 {
		// We need to check whether the certificate template exists ... we do this way because
		// there are no FK constraints between host_certificate_templates and certificate_templates.
		var result uint
		err := ds.writer(ctx).GetContext(ctx, &result, `SELECT id FROM certificate_templates WHERE id = ?`, certificateTemplateID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("CertificateTemplate").WithMessage(fmt.Sprintf("No certificate template found for template ID '%d'",
					certificateTemplateID)))
			}
			return ctxerr.Wrap(ctx, err, "could not read certificate template for inserting new record")
		}

		// Default to install operation type for new records
		params := []any{hostUUID, certificateTemplateID, status, detail, "", fleet.MDMOperationTypeInstall}
		if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, params...); err != nil {
			return ctxerr.Wrap(ctx, err, "could not insert new host certificate template")
		}
	}

	return nil
}

// ListAndroidHostUUIDsWithPendingCertificateTemplates returns hosts that have
// certificate templates in 'pending' status ready for delivery.
func (ds *Datastore) ListAndroidHostUUIDsWithPendingCertificateTemplates(
	ctx context.Context,
	offset int,
	limit int,
) ([]string, error) {
	const stmt = `
		SELECT DISTINCT host_uuid
		FROM host_certificate_templates
		WHERE
			status = 'pending' AND
			operation_type = 'install'
		ORDER BY host_uuid
		LIMIT ? OFFSET ?
	`
	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, limit, offset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host uuids with pending certificate templates")
	}
	return hostUUIDs, nil
}

// TransitionCertificateTemplatesToDelivering atomically transitions certificate templates
// from 'pending' to 'delivering' status. Returns templates that were successfully transitioned.
// This prevents concurrent cron runs from processing the same templates.
func (ds *Datastore) TransitionCertificateTemplatesToDelivering(
	ctx context.Context,
	hostUUID string,
) ([]fleet.HostCertificateTemplate, error) {
	var templates []fleet.HostCertificateTemplate
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Select templates that are pending
		const selectStmt = `
			SELECT
				id, host_uuid, certificate_template_id, fleet_challenge, status, operation_type
			FROM host_certificate_templates
			WHERE
				host_uuid = ? AND
				status = 'pending' AND
				operation_type = 'install'
			FOR UPDATE
		`
		if err := sqlx.SelectContext(ctx, tx, &templates, selectStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "select pending templates")
		}

		if len(templates) == 0 {
			return nil // Another process already picked them up
		}

		// Transition to delivering
		const updateStmt = `
			UPDATE host_certificate_templates
			SET status = 'delivering', updated_at = NOW()
			WHERE host_uuid = ? AND status = 'pending' AND operation_type = 'install'
		`
		if _, err := tx.ExecContext(ctx, updateStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "update to delivering")
		}

		// Update returned templates with new status
		for i := range templates {
			templates[i].Status = fleet.CertificateTemplateDelivering
		}

		return nil
	})
	return templates, err
}

// TransitionCertificateTemplatesToDelivered transitions templates from 'delivering' to 'delivered'
// and sets the fleet_challenge for each template.
func (ds *Datastore) TransitionCertificateTemplatesToDelivered(
	ctx context.Context,
	hostUUID string,
	challenges map[uint]string, // certificateTemplateID -> challenge
) error {
	if len(challenges) == 0 {
		return nil
	}

	// Build UPDATE with CASE for each template's challenge
	var caseStmt strings.Builder
	var templateIDs []any
	caseStmt.WriteString("CASE certificate_template_id ")
	for templateID, challenge := range challenges {
		caseStmt.WriteString("WHEN ? THEN ? ")
		templateIDs = append(templateIDs, templateID, challenge)
	}
	caseStmt.WriteString("END")

	// Build IN clause for template IDs
	inPlaceholders := make([]string, 0, len(challenges))
	for templateID := range challenges {
		inPlaceholders = append(inPlaceholders, "?")
		templateIDs = append(templateIDs, templateID)
	}

	query := fmt.Sprintf(`
		UPDATE host_certificate_templates
		SET
			status = 'delivered',
			fleet_challenge = %s,
			updated_at = NOW()
		WHERE
			host_uuid = ? AND
			status = 'delivering' AND
			certificate_template_id IN (%s)
	`, caseStmt.String(), strings.Join(inPlaceholders, ","))

	args := templateIDs
	// Insert hostUUID before the IN clause placeholders
	args = append(args[:len(challenges)*2], append([]any{hostUUID}, args[len(challenges)*2:]...)...)

	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "transition to delivered")
	}

	return nil
}

// RevertCertificateTemplatesToPending reverts specific templates from 'delivering' back to 'pending'.
func (ds *Datastore) RevertCertificateTemplatesToPending(
	ctx context.Context,
	hostUUID string,
	certificateTemplateIDs []uint,
) error {
	if len(certificateTemplateIDs) == 0 {
		return nil
	}

	stmt, args, err := sqlx.In(`
		UPDATE host_certificate_templates
		SET status = 'pending', updated_at = NOW()
		WHERE host_uuid = ? AND status = 'delivering' AND operation_type = 'install'
		AND certificate_template_id IN (?)
	`, hostUUID, certificateTemplateIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build revert to pending query")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "revert to pending")
	}
	return nil
}

// RevertStaleCertificateTemplates reverts certificate templates stuck in 'delivering' status
// for longer than the specified duration back to 'pending'. This is a safety net for
// server crashes during AMAPI calls.
func (ds *Datastore) RevertStaleCertificateTemplates(
	ctx context.Context,
	staleDuration time.Duration,
) (int64, error) {
	const stmt = `
		UPDATE host_certificate_templates
		SET status = 'pending', updated_at = NOW()
		WHERE
			status = 'delivering' AND
			updated_at < DATE_SUB(NOW(), INTERVAL ? SECOND)
	`
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, int(staleDuration.Seconds()))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "revert stale certificate templates")
	}
	return result.RowsAffected()
}
