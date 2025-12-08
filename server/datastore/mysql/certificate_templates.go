package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetCertificateTemplateById(ctx context.Context, id uint) (*fleet.CertificateTemplateResponseFull, error) {
	var template fleet.CertificateTemplateResponseFull
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &template, `
		SELECT
			certificate_templates.id,
			certificate_templates.name,
			certificate_templates.team_id,
			certificate_templates.subject_name,
			certificate_templates.created_at,
			certificate_authorities.id AS certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_authorities.type AS certificate_authority_type,
			certificate_authorities.challenge_encrypted AS scep_challenge_encrypted,
			host_certificate_templates.status AS status,
			host_certificate_templates.fleet_challenge AS fleet_challenge
		FROM certificate_templates
		INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
		LEFT JOIN host_certificate_templates
			ON host_certificate_templates.certificate_template_id = certificate_templates.id
		WHERE certificate_templates.id = ?
	`, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting certificate_template by id")
	}

	if template.Status != nil && *template.Status == fleet.MDMDeliveryPending {
		if template.SCEPChallengeEncrypted != nil {
			decryptedChallenge, err := decrypt(template.SCEPChallengeEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting scep challenge")
			}
			template.SCEPChallenge = ptr.String(string(decryptedChallenge))
		}
	} else {
		// Ensure challenges are nil if not in pending status
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
			certificate_templates.certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_templates.created_at
		%s
`, fromClause)

	stmtPaged, args := appendListOptionsWithCursorToSQL(stmt, args, &opts)

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

func (ds *Datastore) CreateCertificateTemplate(ctx context.Context, certificateTemplate *fleet.CertificateTemplate) (*fleet.CertificateTemplateResponseFull, error) {
	result, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name
		) VALUES (?, ?, ?, ?)
	`, certificateTemplate.Name, certificateTemplate.TeamID, certificateTemplate.CertificateAuthorityID, certificateTemplate.SubjectName)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting certificate_template")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert id for certificate_template")
	}

	return &fleet.CertificateTemplateResponseFull{
		CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
			ID:                     uint(id), //nolint:gosec
			Name:                   certificateTemplate.Name,
			CertificateAuthorityId: certificateTemplate.CertificateAuthorityID,
		},
		SubjectName: certificateTemplate.SubjectName,
		TeamID:      certificateTemplate.TeamID,
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

	const sqlInsertCertificate = `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name
		) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			team_id = VALUES(team_id)
	`

	teamsModifiedSet := make(map[uint]struct{})
	for _, cert := range certificateTemplates {
		result, err := ds.writer(ctx).ExecContext(ctx, sqlInsertCertificate, cert.Name, cert.TeamID, cert.CertificateAuthorityID, cert.SubjectName)
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
	ct.name, 
	hct.status,
	hct.detail
FROM host_certificate_templates hct
	INNER JOIN certificate_templates ct ON ct.id = hct.certificate_template_id 
WHERE hct.host_uuid = ?`

	var hTemplates []fleet.HostCertificateTemplate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hTemplates, stmt, hostUUID); err != nil {
		return nil, err
	}
	return hTemplates, nil
}

func (ds *Datastore) GetMDMProfileSummaryFromHostCertificateTemplates(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	var stmt string
	var args []interface{}

	if teamID != nil {
		stmt = `
SELECT
	hct.status AS status,
	COUNT(DISTINCT hct.host_uuid) AS n
FROM host_certificate_templates hct
INNER JOIN certificate_templates ct ON hct.certificate_template_id = ct.id
WHERE ct.team_id = ?
GROUP BY hct.status`
		args = append(args, *teamID)
	} else {
		stmt = `
SELECT
	hct.status AS status,
	COUNT(DISTINCT hct.host_uuid) AS n
FROM host_certificate_templates hct
GROUP BY hct.status`
	}

	var dest []struct {
		Count  uint   `db:"n"`
		Status string `db:"status"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, args...); err != nil {
		return nil, err
	}

	byStatus := make(map[string]uint)
	for _, s := range dest {
		if _, ok := byStatus[s.Status]; ok {
			return nil, fmt.Errorf("duplicate status %s found", s.Status)
		}
		byStatus[s.Status] = s.Count
	}

	var res fleet.MDMProfilesSummary
	for s, c := range byStatus {
		switch fleet.MDMDeliveryStatus(s) {
		case fleet.MDMDeliveryFailed:
			res.Failed = c
		case fleet.MDMDeliveryPending:
			res.Pending = c
		case fleet.MDMDeliveryVerifying:
			res.Verifying = c
		case fleet.MDMDeliveryVerified:
			res.Verified = c
		default:
			return nil, fmt.Errorf("unknown status %s", s)
		}
	}

	return &res, nil
}
