package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ListAndroidHostUUIDsWithCertificateTemplates returns a batch of host UUIDs that have certificate templates to deliver
func (ds *Datastore) ListAndroidHostUUIDsWithCertificateTemplates(ctx context.Context, offset int, limit int) ([]string, error) {
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
			hosts.uuid IN (?)
		ORDER BY hosts.uuid, certificate_templates.id
	`, hostUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build query for certificate templates")
	}

	query = ds.reader(ctx).Rebind(query)
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

// GetHostCertificateTemplate returns a specific host certificate template
func (ds *Datastore) GetHostCertificateTemplate(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.HostCertificateTemplate, error) {
	const stmt = `
		SELECT
			id,
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status,
			created_at,
			updated_at
		FROM host_certificate_templates
		WHERE host_uuid = ? AND certificate_template_id = ?
	`

	var result fleet.HostCertificateTemplate
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, hostUUID, certificateTemplateID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host certificate template")
	}

	return &result, nil
}

// BulkInsertHostCertificateTemplates inserts multiple host_certificate_templates records
func (ds *Datastore) BulkInsertHostCertificateTemplates(ctx context.Context, hostCertTemplates []fleet.HostCertificateTemplate) error {
	if len(hostCertTemplates) == 0 {
		return nil
	}

	const argsCount = 4

	const sqlInsert = `
		INSERT INTO host_certificate_templates (
			host_uuid,
			certificate_template_id,
			fleet_challenge,
			status
		) VALUES %s
	`

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(hostCertTemplates)*argsCount)

	for _, hct := range hostCertTemplates {
		args = append(args, hct.HostUUID, hct.CertificateTemplateID, hct.FleetChallenge, hct.Status)
		placeholders.WriteString("(?,?,?,?),")
	}

	stmt := fmt.Sprintf(sqlInsert, strings.TrimSuffix(placeholders.String(), ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk insert host_certificate_templates")
	}

	return nil
}
