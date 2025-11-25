package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetCertificateTemplateById(ctx context.Context, id uint) (*fleet.CertificateTemplateResponseFull, error) {
	var template fleet.CertificateTemplateResponseFull
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &template, `
		SELECT
			certificate_templates.id,
			certificate_templates.name,
			certificate_templates.team_id,
			certificate_templates.certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_templates.subject_name,
			certificate_templates.created_at
		FROM certificate_templates
		INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
		WHERE certificate_templates.id = ?
	`, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting certificate_template by id")
	}
	return &template, nil
}

func (ds *Datastore) GetCertificateTemplatesByTeamID(ctx context.Context, teamID uint, page, perPage int) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
	var templates []*fleet.CertificateTemplateResponseSummary

	if perPage <= 0 {
		perPage = 20
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &templates, `
		SELECT
			certificate_templates.id,
			certificate_templates.name,
			certificate_templates.certificate_authority_id,
			certificate_authorities.name AS certificate_authority_name,
			certificate_templates.created_at
		FROM certificate_templates
		INNER JOIN certificate_authorities ON certificate_templates.certificate_authority_id = certificate_authorities.id
		WHERE team_id = ?
		ORDER BY certificate_templates.id ASC
		LIMIT ? OFFSET ?
	`, teamID, perPage+1, perPage*page,
	); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "getting certificate_templates by team_id")
	}

	paginationMetaData := &fleet.PaginationMetadata{
		HasPreviousResults: page > 0,
	}
	if len(templates) > perPage {
		templates = templates[:perPage]
		paginationMetaData.HasNextResults = true
	}

	return templates, paginationMetaData, nil
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

func (ds *Datastore) BatchUpsertCertificateTemplates(ctx context.Context, certificateTemplates []*fleet.CertificateTemplate) error {
	if len(certificateTemplates) == 0 {
		return nil
	}

	const argsCountInsertCertificate = 4

	const sqlInsertCertificate = `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name
		) VALUES %s
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			team_id = VALUES(team_id),
			certificate_authority_id = VALUES(certificate_authority_id),
			subject_name = VALUES(subject_name)
	`

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(certificateTemplates)*argsCountInsertCertificate)

	for _, cert := range certificateTemplates {
		args = append(args, cert.Name, cert.TeamID, cert.CertificateAuthorityID, cert.SubjectName)
		placeholders.WriteString("(?,?,?,?),")
	}

	stmt := fmt.Sprintf(sqlInsertCertificate, strings.TrimSuffix(placeholders.String(), ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upserting certificate_templates")
	}

	return nil
}

func (ds *Datastore) BatchDeleteCertificateTemplates(ctx context.Context, certificateTemplateIDs []uint) error {
	if len(certificateTemplateIDs) == 0 {
		return nil
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

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting certificate_templates")
	}

	return nil
}
