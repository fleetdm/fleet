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
	stmt := fmt.Sprintf(`
		SELECT DISTINCT
			hosts.uuid
		FROM certificate_templates
		INNER JOIN hosts ON (hosts.team_id = certificate_templates.team_id OR (hosts.team_id IS NULL AND certificate_templates.team_id = 0))
		INNER JOIN host_mdm ON host_mdm.host_id = hosts.id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.platform = '%s' AND
			host_mdm.enrolled = 1 AND
			host_certificate_templates.id IS NULL
		ORDER BY hosts.uuid
		LIMIT ? OFFSET ?
	`, fleet.AndroidPlatform)

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, limit, offset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android host uuids with certificate templates")
	}

	return hostUUIDs, nil
}

// ListCertificateTemplatesForHosts returns ALL certificate templates for the given host UUIDs.
// This includes:
// 1. Templates matching the host's current team (for install operations)
// 2. Templates marked for removal (which may be from a previous team after team transfer)
func (ds *Datastore) ListCertificateTemplatesForHosts(ctx context.Context, hostUUIDs []string) ([]fleet.CertificateTemplateForHost, error) {
	if len(hostUUIDs) == 0 {
		return nil, nil
	}

	// Query 1: Templates matching host's current team
	// Query 2: UNION with removal entries from host_certificate_templates
	//          (these may reference templates from a different team after team transfer)
	// UNION removes duplicates if a template appears in both result sets
	query, args, err := sqlx.In(fmt.Sprintf(`
		SELECT
			hosts.uuid AS host_uuid,
			certificate_templates.id AS certificate_template_id,
			host_certificate_templates.fleet_challenge AS fleet_challenge,
			host_certificate_templates.status AS status,
			host_certificate_templates.operation_type AS operation_type,
			certificate_authorities.type AS ca_type,
			certificate_authorities.name AS ca_name
		FROM certificate_templates
		INNER JOIN hosts ON (hosts.team_id = certificate_templates.team_id OR (hosts.team_id IS NULL AND certificate_templates.team_id = 0))
		INNER JOIN certificate_authorities ON certificate_authorities.id = certificate_templates.certificate_authority_id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.uuid IN (?)

		UNION

		SELECT
			hct.host_uuid AS host_uuid,
			hct.certificate_template_id AS certificate_template_id,
			hct.fleet_challenge AS fleet_challenge,
			hct.status AS status,
			hct.operation_type AS operation_type,
			ca.type AS ca_type,
			ca.name AS ca_name
		FROM host_certificate_templates hct
		INNER JOIN certificate_templates ct ON ct.id = hct.certificate_template_id
		INNER JOIN certificate_authorities ca ON ca.id = ct.certificate_authority_id
		WHERE
			hct.host_uuid IN (?)
			AND hct.operation_type = '%s'

		ORDER BY host_uuid, certificate_template_id
	`, fleet.MDMOperationTypeRemove), hostUUIDs, hostUUIDs)
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
			host_certificate_templates.operation_type AS operation_type,
			certificate_authorities.type AS ca_type,
			certificate_authorities.name AS ca_name
		FROM certificate_templates
		INNER JOIN hosts ON (hosts.team_id = certificate_templates.team_id OR (hosts.team_id IS NULL AND certificate_templates.team_id = 0))
		INNER JOIN certificate_authorities ON certificate_authorities.id = certificate_templates.certificate_authority_id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.host_uuid = hosts.uuid
			AND host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE
			hosts.uuid = ? AND certificate_templates.id = ?
	`

	var result fleet.CertificateTemplateForHost
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, hostUUID, certificateTemplateID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("CertificateTemplateForHost"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get certificate template for host")
	}

	return &result, nil
}

// GetHostCertificateTemplateRecord returns the host_certificate_templates record directly without
// requiring the parent certificate_template to exist. Used for status updates on orphaned records.
func (ds *Datastore) GetHostCertificateTemplateRecord(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.HostCertificateTemplate, error) {
	const stmt = `
		SELECT
			id,
			name,
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status,
			operation_type,
			detail,
			created_at,
			updated_at
		FROM host_certificate_templates
		WHERE host_uuid = ? AND certificate_template_id = ?
	`

	var result fleet.HostCertificateTemplate
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, hostUUID, certificateTemplateID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostCertificateTemplate"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host certificate template record")
	}

	return &result, nil
}

// BulkInsertHostCertificateTemplates inserts multiple host_certificate_templates records
func (ds *Datastore) BulkInsertHostCertificateTemplates(ctx context.Context, hostCertTemplates []fleet.HostCertificateTemplate) error {
	if len(hostCertTemplates) == 0 {
		return nil
	}

	const argsCount = 6

	const sqlInsert = `
		INSERT INTO host_certificate_templates (
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status,
			operation_type,
			name
		) VALUES %s
	`

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(hostCertTemplates)*argsCount)

	for _, hct := range hostCertTemplates {
		args = append(args, hct.HostUUID, hct.CertificateTemplateID, hct.FleetChallenge, hct.Status, hct.OperationType, hct.Name)
		placeholders.WriteString("(?,?,?,?,?,?),")
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

// DeleteHostCertificateTemplate deletes a single host_certificate_template record
// identified by host_uuid and certificate_template_id.
func (ds *Datastore) DeleteHostCertificateTemplate(ctx context.Context, hostUUID string, certificateTemplateID uint) error {
	const stmt = `DELETE FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, certificateTemplateID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_certificate_template")
	}

	return nil
}

func (ds *Datastore) UpsertCertificateStatus(
	ctx context.Context,
	hostUUID string,
	certificateTemplateID uint,
	status fleet.MDMDeliveryStatus,
	detail *string,
	operationType fleet.MDMOperationType,
) error {
	// Validate the status.
	if !status.IsValid() {
		return ctxerr.Wrap(ctx, fmt.Errorf("Invalid status '%s'", string(status)))
	}

	updateStmt := `
		UPDATE host_certificate_templates
		SET status = ?, detail = ?, operation_type = ?
		WHERE host_uuid = ? AND certificate_template_id = ?`

	// Attempt to update the certificate status for the given host and template.
	result, err := ds.writer(ctx).ExecContext(ctx, updateStmt, status, detail, operationType, hostUUID, certificateTemplateID)
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
		// Also get the name for insertion.
		var templateInfo struct {
			ID   uint   `db:"id"`
			Name string `db:"name"`
		}
		err := ds.writer(ctx).GetContext(ctx, &templateInfo, `SELECT id, name FROM certificate_templates WHERE id = ?`, certificateTemplateID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("CertificateTemplate").WithMessage(fmt.Sprintf("No certificate template found for template ID '%d'",
					certificateTemplateID)))
			}
			return ctxerr.Wrap(ctx, err, "could not read certificate template for inserting new record")
		}

		insertStmt := `
			INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, detail, fleet_challenge, operation_type, name)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		params := []any{hostUUID, certificateTemplateID, status, detail, "", operationType, templateInfo.Name}

		if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, params...); err != nil {
			return ctxerr.Wrap(ctx, err, "could not insert new host certificate template")
		}
	}

	return nil
}

// ListAndroidHostUUIDsWithPendingCertificateTemplates returns hosts that have
// certificate templates in 'pending' status ready for delivery (both install and remove operations).
func (ds *Datastore) ListAndroidHostUUIDsWithPendingCertificateTemplates(
	ctx context.Context,
	offset int,
	limit int,
) ([]string, error) {
	stmt := fmt.Sprintf(`
		SELECT DISTINCT host_uuid
		FROM host_certificate_templates
		WHERE status = '%s'
		ORDER BY host_uuid
		LIMIT ? OFFSET ?
	`, fleet.CertificateTemplatePending)
	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, limit, offset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host uuids with pending certificate templates")
	}
	return hostUUIDs, nil
}

// GetAndTransitionCertificateTemplatesToDelivering retrieves all certificate templates
// for a host (both install and remove operations), transitions any pending ones to 'delivering' status,
// and returns both the newly delivering template IDs and the existing (verified/delivered) ones.
// This prevents concurrent cron runs from processing the same templates.
func (ds *Datastore) GetAndTransitionCertificateTemplatesToDelivering(
	ctx context.Context,
	hostUUID string,
) (*fleet.HostCertificateTemplatesForDelivery, error) {
	result := &fleet.HostCertificateTemplatesForDelivery{}
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Select ALL templates (both install and remove) for this host
		var rows []struct {
			ID                    uint                            `db:"id"`
			CertificateTemplateID uint                            `db:"certificate_template_id"`
			Status                fleet.CertificateTemplateStatus `db:"status"`
			OperationType         fleet.MDMOperationType          `db:"operation_type"`
		}
		const selectStmt = `
			SELECT id, certificate_template_id, status, operation_type
			FROM host_certificate_templates
			WHERE host_uuid = ?
			FOR UPDATE
		`
		if err := sqlx.SelectContext(ctx, tx, &rows, selectStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "select templates")
		}

		if len(rows) == 0 {
			return nil
		}

		// Separate templates by status and build the Templates list
		var pendingIDs []uint // primary key IDs for UPDATE (only pending ones need transitioning)
		for _, r := range rows {
			switch r.Status {
			case fleet.CertificateTemplatePending:
				pendingIDs = append(pendingIDs, r.ID)
				result.DeliveringTemplateIDs = append(result.DeliveringTemplateIDs, r.CertificateTemplateID)
				// Status will be delivering after transition
				result.Templates = append(result.Templates, fleet.HostCertificateTemplateForDelivery{
					CertificateTemplateID: r.CertificateTemplateID,
					Status:                fleet.CertificateTemplateDelivering,
					OperationType:         r.OperationType,
				})
			case fleet.CertificateTemplateDelivering:
				// Already delivering (from a previous failed run), include in delivering list; should be very rare
				result.DeliveringTemplateIDs = append(result.DeliveringTemplateIDs, r.CertificateTemplateID)
				result.Templates = append(result.Templates, fleet.HostCertificateTemplateForDelivery{
					CertificateTemplateID: r.CertificateTemplateID,
					Status:                r.Status,
					OperationType:         r.OperationType,
				})
			default:
				// delivered, verified, failed
				result.Templates = append(result.Templates, fleet.HostCertificateTemplateForDelivery{
					CertificateTemplateID: r.CertificateTemplateID,
					Status:                r.Status,
					OperationType:         r.OperationType,
				})
			}
		}

		if len(pendingIDs) == 0 {
			return nil // No pending templates to transition
		}

		// Transition only pending templates to delivering
		updateStmt, args, err := sqlx.In(fmt.Sprintf(`
			UPDATE host_certificate_templates
			SET status = '%s', updated_at = NOW()
			WHERE id IN (?)
		`, fleet.CertificateTemplateDelivering), pendingIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build update to delivering query")
		}
		if _, err := tx.ExecContext(ctx, updateStmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update to delivering")
		}
		return nil
	})
	return result, err
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

	// Build UPDATE with CASE for each template's challenge.
	// This is called once per host, so the CASE size is bounded by templates per host (small).
	// Using a single UPDATE per host is more efficient than individual updates when processing many hosts.
	var caseStmt strings.Builder
	args := make([]any, 0, len(challenges)*3+1) // CASE args + hostUUID + IN args
	caseStmt.WriteString("CASE certificate_template_id ")
	for templateID, challenge := range challenges {
		caseStmt.WriteString("WHEN ? THEN ? ")
		args = append(args, templateID, challenge)
	}
	caseStmt.WriteString("END")

	// Add hostUUID for WHERE clause
	args = append(args, hostUUID)

	// Build IN clause for template IDs
	inPlaceholders := make([]string, 0, len(challenges))
	for templateID := range challenges {
		inPlaceholders = append(inPlaceholders, "?")
		args = append(args, templateID)
	}

	query := fmt.Sprintf(`
		UPDATE host_certificate_templates
		SET
			status = '%s',
			fleet_challenge = %s,
			updated_at = NOW()
		WHERE
			host_uuid = ? AND
			status = '%s' AND
			certificate_template_id IN (%s)
	`, fleet.CertificateTemplateDelivered, caseStmt.String(), fleet.CertificateTemplateDelivering, strings.Join(inPlaceholders, ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "transition to delivered")
	}

	return nil
}

// RevertHostCertificateTemplatesToPending reverts specific host certificate templates from 'delivering' back to 'pending'.
func (ds *Datastore) RevertHostCertificateTemplatesToPending(
	ctx context.Context,
	hostUUID string,
	certificateTemplateIDs []uint,
) error {
	if len(certificateTemplateIDs) == 0 {
		return nil
	}

	stmtTemplate := fmt.Sprintf(`
		UPDATE host_certificate_templates
		SET status = '%s', updated_at = NOW()
		WHERE host_uuid = ? AND status = '%s' AND operation_type = '%s'
		AND certificate_template_id IN (?)
	`, fleet.CertificateTemplatePending, fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeInstall)
	stmt, args, err := sqlx.In(stmtTemplate, hostUUID, certificateTemplateIDs)
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
	stmt := fmt.Sprintf(`
		UPDATE host_certificate_templates
		SET status = '%s', updated_at = NOW()
		WHERE
			status = '%s' AND
			updated_at < DATE_SUB(NOW(), INTERVAL ? SECOND)
	`, fleet.CertificateTemplatePending, fleet.CertificateTemplateDelivering)
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, int(staleDuration.Seconds()))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "revert stale certificate templates")
	}
	return result.RowsAffected()
}

// SetHostCertificateTemplatesToPendingRemove prepares certificate templates for removal.
// For a given certificate template ID, it deletes any rows with status in (pending, failed)
// and updates all other rows to status=pending, operation_type=remove.
func (ds *Datastore) SetHostCertificateTemplatesToPendingRemove(
	ctx context.Context,
	certificateTemplateID uint,
) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Delete rows with status in (pending, failed) - these were never successfully installed
		deleteStmt := fmt.Sprintf(`
			DELETE FROM host_certificate_templates
			WHERE certificate_template_id = ? AND status IN ('%s', '%s')
		`, fleet.CertificateTemplatePending, fleet.CertificateTemplateFailed)
		if _, err := tx.ExecContext(ctx, deleteStmt, certificateTemplateID); err != nil {
			return ctxerr.Wrap(ctx, err, "delete pending/failed host certificate templates")
		}

		// Update all remaining rows to status=pending, operation_type=remove
		updateStmt := fmt.Sprintf(`
			UPDATE host_certificate_templates
			SET status = '%s', operation_type = '%s'
			WHERE certificate_template_id = ?
		`, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove)
		if _, err := tx.ExecContext(ctx, updateStmt, certificateTemplateID); err != nil {
			return ctxerr.Wrap(ctx, err, "update host certificate templates to pending remove")
		}

		return nil
	})
}

// SetHostCertificateTemplatesToPendingRemoveForHost prepares all certificate templates
// for a specific host for removal. Used during team transfer to mark old team's templates
// for removal before creating new pending templates for the new team.
// Records with operation_type=remove are left unchanged (removal already in progress).
func (ds *Datastore) SetHostCertificateTemplatesToPendingRemoveForHost(
	ctx context.Context,
	hostUUID string,
) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Delete rows with status in (pending, failed) and operation_type=install
		// These certificates were never successfully installed on the device
		deleteStmt := fmt.Sprintf(`
			DELETE FROM host_certificate_templates
			WHERE host_uuid = ? AND status IN ('%s', '%s') AND operation_type = '%s'
		`, fleet.CertificateTemplatePending, fleet.CertificateTemplateFailed, fleet.MDMOperationTypeInstall)
		if _, err := tx.ExecContext(ctx, deleteStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "delete pending/failed install host certificate templates for host")
		}

		// Update remaining install rows to status=pending, operation_type=remove
		updateStmt := fmt.Sprintf(`
			UPDATE host_certificate_templates
			SET status = '%s', operation_type = '%s'
			WHERE host_uuid = ? AND operation_type = '%s'
		`, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, fleet.MDMOperationTypeInstall)
		if _, err := tx.ExecContext(ctx, updateStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "update host certificate templates to pending remove for host")
		}

		return nil
	})
}
