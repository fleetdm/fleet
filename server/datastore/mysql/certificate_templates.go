package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

// certificateTemplateAllowedOrderKeys defines the allowed order keys for ListCertificateTemplates.
// SECURITY: This prevents information disclosure via arbitrary column sorting.
var certificateTemplateAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"id": "certificate_templates.id",
}

// subjectAlternativeNameForStorage returns the value to bind for the subject_alternative_name column.
// Whitespace-only and empty values are stored as SQL NULL; non-empty values are stored verbatim
// (no per-token trimming) to preserve admin intent and keep GitOps round-trips idempotent.
func subjectAlternativeNameForStorage(san string) any {
	if strings.TrimSpace(san) == "" {
		return nil
	}
	return san
}

const certificateTemplateResponseSql = `
	SELECT
		certificate_templates.id,
		certificate_templates.name,
		certificate_templates.team_id,
		certificate_templates.subject_name,
		COALESCE(certificate_templates.subject_alternative_name, '') AS subject_alternative_name,
		certificate_templates.created_at,
		certificate_authorities.id AS certificate_authority_id,
		certificate_authorities.name AS certificate_authority_name,
		certificate_authorities.type AS certificate_authority_type
	FROM certificate_templates
	INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
`

func (ds *Datastore) GetCertificateTemplateById(ctx context.Context, id uint) (*fleet.CertificateTemplateResponse, error) {
	var template fleet.CertificateTemplateResponse
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &template, certificateTemplateResponseSql+`
		WHERE certificate_templates.id = ?
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("CertificateTemplate").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting certificate_template by id")
	}

	return &template, nil
}

func (ds *Datastore) GetCertificateTemplatesByIdsAndTeam(ctx context.Context, ids []uint, teamID uint) ([]*fleet.CertificateTemplateResponse, error) {
	var certificateTemplates []*fleet.CertificateTemplateResponse

	if len(ids) == 0 {
		return certificateTemplates, nil
	}
	// for no team pass 0 as teamID
	query, args, err := sqlx.In(certificateTemplateResponseSql+`
		WHERE certificate_templates.team_id = ? AND certificate_templates.id IN (?)
	`, teamID, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for certificate_templates by team id and ids")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &certificateTemplates, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query certificate_template by team id and ids")
	}

	return certificateTemplates, nil
}

func (ds *Datastore) GetCertificateTemplateByTeamIDAndName(ctx context.Context, teamID uint, name string) (*fleet.CertificateTemplateResponse, error) {
	var template fleet.CertificateTemplateResponse
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &template, certificateTemplateResponseSql+`
		WHERE certificate_templates.team_id = ? AND certificate_templates.name = ?
	`, teamID, name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting certificate_template by team id and name")
	}

	return &template, nil
}

// GetCertificateTemplateByIdForHost gets a certificate template by ID with host-specific status and challenge.
// This is used when a host (fleetd/Android agent) requests its certificate.
func (ds *Datastore) GetCertificateTemplateByIdForHost(ctx context.Context, id uint, hostUUID string) (*fleet.CertificateTemplateResponseForHost, error) {
	var template fleet.CertificateTemplateResponseForHost
	stmt := fmt.Sprintf(`
		SELECT
			certificate_templates.id,
			certificate_templates.name,
			certificate_templates.team_id,
			certificate_templates.subject_name,
			COALESCE(certificate_templates.subject_alternative_name, '') AS subject_alternative_name,
			certificate_templates.created_at,
			certificate_authorities.id AS certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_authorities.type AS certificate_authority_type,
			certificate_authorities.challenge_encrypted AS scep_challenge_encrypted,
			host_certificate_templates.status AS status,
			COALESCE(BIN_TO_UUID(host_certificate_templates.uuid, true), '') AS uuid,
			host_certificate_templates.fleet_challenge AS fleet_challenge
		FROM certificate_templates
		INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
		INNER JOIN host_certificate_templates
			ON host_certificate_templates.certificate_template_id = certificate_templates.id
			AND host_certificate_templates.host_uuid = ?
			AND host_certificate_templates.operation_type = '%s'
		WHERE certificate_templates.id = ?
	`, fleet.MDMOperationTypeInstall)
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &template, stmt, hostUUID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("CertificateTemplateForHost"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting certificate_template by id for host")
	}

	// Only include challenges if status is "delivered"
	if template.Status == fleet.CertificateTemplateDelivered {
		if template.SCEPChallengeEncrypted != nil {
			decryptedChallenge, err := decrypt(template.SCEPChallengeEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting scep challenge")
			}
			template.SCEPChallenge = ptr.String(string(decryptedChallenge))
		}
	} else {
		// Ensure challenges are nil if not in delivered status
		template.SCEPChallenge = nil
		template.FleetChallenge = nil
	}

	return &template, nil
}

func (ds *Datastore) GetCertificateTemplatesByTeamID(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
	// for no team pass 0 as teamID
	args := []any{teamID}

	fromClause := `
		FROM certificate_templates
		INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
		WHERE team_id = ?
`
	countStmt := fmt.Sprintf(`SELECT COUNT(1) %s`, fromClause)

	stmt := fmt.Sprintf(`
		SELECT
			certificate_templates.id,
			certificate_templates.name,
			certificate_templates.subject_name,
			COALESCE(certificate_templates.subject_alternative_name, '') AS subject_alternative_name,
			certificate_templates.certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_templates.created_at
		%s
`, fromClause)

	stmtPaged, args, err := appendListOptionsWithCursorToSQLSecure(stmt, args, &opts, certificateTemplateAllowedOrderKeys)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "apply list options")
	}

	var templates []*fleet.CertificateTemplateResponseSummary
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &templates, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "getting certificate_templates by team_id")
	}

	var metaData *fleet.PaginationMetadata
	if opts.IncludeMetadata {
		var count uint
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, countStmt, args...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "counting certificate templates")
		}
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opts.Page > 0, TotalResults: count}
		if len(templates) > int(opts.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			templates = templates[:len(templates)-1]
		}
	}

	return templates, metaData, nil
}

func (ds *Datastore) CreateCertificateTemplate(ctx context.Context, certificateTemplate *fleet.CertificateTemplate) (*fleet.CertificateTemplateResponse, error) {
	sanArg := subjectAlternativeNameForStorage(certificateTemplate.SubjectAlternativeName)
	result, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name,
			subject_alternative_name
		) VALUES (?, ?, ?, ?, ?)
	`, certificateTemplate.Name, certificateTemplate.TeamID, certificateTemplate.CertificateAuthorityID,
		certificateTemplate.SubjectName, sanArg)
	if err != nil {
		if IsDuplicate(err) {
			return nil, ctxerr.Wrap(ctx, alreadyExists("CertificateTemplate", certificateTemplate.Name), "inserting certificate_template")
		}
		return nil, ctxerr.Wrap(ctx, err, "inserting certificate_template")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert id for certificate_template")
	}

	storedSAN := ""
	if sanArg != nil {
		storedSAN = certificateTemplate.SubjectAlternativeName
	}

	return &fleet.CertificateTemplateResponse{
		CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
			ID:                     uint(id), //nolint:gosec
			Name:                   certificateTemplate.Name,
			SubjectName:            certificateTemplate.SubjectName,
			SubjectAlternativeName: storedSAN,
			CertificateAuthorityId: certificateTemplate.CertificateAuthorityID,
		},
		TeamID: certificateTemplate.TeamID,
	}, nil
}

func (ds *Datastore) DeleteCertificateTemplate(ctx context.Context, id uint) error {
	result, err := ds.writer(ctx).ExecContext(ctx, `
		DELETE FROM certificate_templates
		WHERE id = ?
	`, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting certificate_template")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting rows affected for certificate_template")
	}
	if rowsAffected == 0 {
		return notFound("CertificateTemplate").WithID(id)
	}

	return nil
}

func (ds *Datastore) BatchUpsertCertificateTemplates(ctx context.Context, certificateTemplates []*fleet.CertificateTemplate) ([]uint, error) {
	if len(certificateTemplates) == 0 {
		return nil, nil
	}

	// On duplicate (team_id, name), this is a no-op for content-bearing fields. SubjectName,
	// CertificateAuthorityID, and SubjectAlternativeName changes are handled upstream, so the
	// upsert intentionally does not propagate updates.
	const sqlInsertCertificate = `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name,
			subject_alternative_name
		) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			team_id = VALUES(team_id)
	`

	teamsModifiedSet := make(map[uint]struct{})
	for _, cert := range certificateTemplates {
		sanArg := subjectAlternativeNameForStorage(cert.SubjectAlternativeName)
		result, err := ds.writer(ctx).ExecContext(ctx, sqlInsertCertificate,
			cert.Name, cert.TeamID, cert.CertificateAuthorityID, cert.SubjectName, sanArg)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "upserting certificate_template")
		}

		if insertOnDuplicateDidInsertOrUpdate(result) {
			teamsModifiedSet[cert.TeamID] = struct{}{}
		}
	}

	teamsModified := make([]uint, 0, len(teamsModifiedSet))
	for teamID := range teamsModifiedSet {
		teamsModified = append(teamsModified, teamID)
	}

	return teamsModified, nil
}

func (ds *Datastore) BatchDeleteCertificateTemplates(ctx context.Context, certificateTemplateIDs []uint) (bool, error) {
	if len(certificateTemplateIDs) == 0 {
		return false, nil
	}

	const sqlDeleteCertificateTemplates = `
		DELETE FROM certificate_templates
		WHERE id IN (%s)
	`
	var placeholders strings.Builder
	args := make([]interface{}, 0, len(certificateTemplateIDs))

	for _, id := range certificateTemplateIDs {
		args = append(args, id)
		placeholders.WriteString("?,")
	}

	stmt := fmt.Sprintf(sqlDeleteCertificateTemplates, strings.TrimSuffix(placeholders.String(), ","))

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "deleting certificate_templates")
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func (ds *Datastore) GetHostCertificateTemplates(ctx context.Context, hostUUID string) ([]fleet.HostCertificateTemplate, error) {
	if hostUUID == "" {
		return nil, errors.New("hostUUID cannot be empty")
	}

	stmt := `
SELECT
	name,
	status,
	detail,
	operation_type,
	certificate_template_id
FROM host_certificate_templates
WHERE host_uuid = ?`

	var hTemplates []fleet.HostCertificateTemplate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hTemplates, stmt, hostUUID); err != nil {
		return nil, err
	}
	return hTemplates, nil
}

// CreatePendingCertificateTemplatesForExistingHosts creates pending certificate template records
// for all enrolled Android hosts in the team when a new certificate template is added.
// Note: teamID = 0 means "no team", which corresponds to hosts.team_id IS NULL.
func (ds *Datastore) CreatePendingCertificateTemplatesForExistingHosts(
	ctx context.Context,
	certificateTemplateID uint,
	teamID uint,
) (int64, error) {
	stmt := fmt.Sprintf(`
		INSERT INTO host_certificate_templates (
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status,
			operation_type,
			name,
			uuid
		)
		SELECT
			hosts.uuid,
			ct.id,
			NULL,
			'%s',
			'%s',
			ct.name,
			UUID_TO_BIN(UUID(), true)
		FROM hosts
		INNER JOIN host_mdm ON host_mdm.host_id = hosts.id
		INNER JOIN certificate_templates ct ON ct.id = ?
		WHERE
			(hosts.team_id = ? OR (? = 0 AND hosts.team_id IS NULL)) AND
			hosts.platform = '%s' AND
			host_mdm.enrolled = 1
		ON DUPLICATE KEY UPDATE host_uuid = host_uuid
	`, fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall, fleet.AndroidPlatform)
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, certificateTemplateID, teamID, teamID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create pending certificate templates for hosts")
	}
	return result.RowsAffected()
}

// CreatePendingCertificateTemplatesForNewHost creates pending certificate template records
// for a newly enrolled Android host based on their team's certificate templates.
// This is called during Android enrollment when the host is assigned to a team.
func (ds *Datastore) CreatePendingCertificateTemplatesForNewHost(
	ctx context.Context,
	hostUUID string,
	teamID uint,
) (int64, error) {
	stmt := fmt.Sprintf(`
		INSERT INTO host_certificate_templates (
			host_uuid,
			certificate_template_id,
			status,
			operation_type,
			name,
			uuid
		)
		SELECT
			?,
			id,
			'%s',
			'%s',
			name,
			UUID_TO_BIN(UUID(), true)
		FROM certificate_templates
		WHERE team_id = ?
		ON DUPLICATE KEY UPDATE
		    -- Unconditionally reset to pending install with a new UUID so the certificate is
		    -- re-delivered. This handles re-enrollment after work profile removal, where the device
		    -- lost all certs but the old records may still exist. Clear stale certificate metadata
		    -- from the previous lifecycle to match ResendHostCertificateTemplate behavior.
			uuid = UUID_TO_BIN(UUID(), true),
			status = '%s',
			operation_type = '%s',
			retry_count = 0,
			fleet_challenge = NULL,
			not_valid_before = NULL,
			not_valid_after = NULL,
			serial = NULL,
			detail = NULL
	`, fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall,
		fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall)
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, teamID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create pending certificate templates for new host")
	}
	return result.RowsAffected()
}

// ResendHostCertificateTemplate resets a certificate template for re-delivery. It sets retry_count
// to MaxCertificateInstallRetries so that the next failure is terminal with no automatic retry,
// giving the resend exactly one attempt. This matches Apple resend behavior.
func (ds *Datastore) ResendHostCertificateTemplate(ctx context.Context, hostID uint, templateID uint) error {
	stmt := fmt.Sprintf(`
		UPDATE
			host_certificate_templates hct
		INNER JOIN
			hosts h ON h.uuid = hct.host_uuid
		SET
			hct.uuid = UUID_TO_BIN(UUID(), true),
			hct.fleet_challenge = NULL,
			hct.not_valid_before = NULL,
			hct.not_valid_after = NULL,
			hct.serial = NULL,
			hct.detail = NULL,
			hct.retry_count = %d,
			hct.status = ?
		WHERE
			h.id = ? AND
			hct.certificate_template_id = ?
		`, fleet.MaxCertificateInstallRetries)

	const deleteChallenge = `
		DELETE c FROM
			challenges c
		INNER JOIN
			host_certificate_templates hct ON hct.fleet_challenge = c.challenge
		INNER JOIN
			hosts h ON h.uuid = hct.host_uuid
		WHERE
			h.id = ? AND
			hct.certificate_template_id = ?
		`

	if err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, deleteChallenge, hostID, templateID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting challenges associated with resent certificate template")
		}

		results, err := tx.ExecContext(ctx, stmt, fleet.CertificateTemplatePending, hostID, templateID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating host certificate template uuid")
		}

		affected, _ := results.RowsAffected()
		if affected == 0 {
			return ctxerr.Wrapf(ctx, notFound("HostCertificateTemplate"), "template %d does not exist for host %d", templateID, hostID)
		}

		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "resetting host certificate template for resend")
	}

	return nil
}
