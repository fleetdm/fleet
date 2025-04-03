package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ConditionalAccessMicrosoftCreateIntegration(
	ctx context.Context, tenantID string, proxyServerSecret string,
) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Currently only one global integration is supported, thus we need to delete the existing
		// one before creating a new one.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM microsoft_compliance_partner_integrations;`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting microsoft_compliance_partner_integrations")
		}

		switch _, err := tx.ExecContext(ctx,
			`INSERT INTO microsoft_compliance_partner_integrations (tenant_id, proxy_server_secret) VALUES (?, ?);`,
			tenantID, proxyServerSecret,
		); {
		case err == nil:
			return nil
		case IsDuplicate(err):
			return ctxerr.Wrap(ctx, alreadyExists("MicrosoftCompliancePartnerIntegration", tenantID))
		default:
			return ctxerr.Wrap(ctx, err, "inserting new microsoft_compliance_partner_integrations")
		}
	})
}

func (ds *Datastore) ConditionalAccessMicrosoftMarkSetupDone(ctx context.Context) error {
	// Currently only one global integration is supported.
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE microsoft_compliance_partner_integrations SET setup_done = true;`,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting microsoft_compliance_partner_integrations")
	}
	return nil
}

func (ds *Datastore) ConditionalAccessMicrosoftGet(ctx context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
	return getConditionalAccessMicrosoft(ctx, ds.reader(ctx))
}

func getConditionalAccessMicrosoft(ctx context.Context, q sqlx.QueryerContext) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
	var integration fleet.ConditionalAccessMicrosoftIntegration
	err := sqlx.GetContext(
		ctx, q, &integration,
		// Currently only one global integration is supported.
		`SELECT tenant_id, proxy_server_secret, setup_done FROM microsoft_compliance_partner_integrations;`,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MicrosoftCompliancePartnerIntegration"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting microsoft_compliance_partner_integrations")
	}
	return &integration, nil
}

func (ds *Datastore) ConditionalAccessMicrosoftDelete(ctx context.Context) error {
	// Currently only one global integration is supported.
	if _, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM microsoft_compliance_partner_integrations;`); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting microsoft_compliance_partner_integrations")
	}
	return nil
}

func (ds *Datastore) LoadHostConditionalAccessStatus(ctx context.Context, hostID uint) (*fleet.HostConditionalAccessStatus, error) {
	var hostConditionalAccessStatus fleet.HostConditionalAccessStatus
	if err := sqlx.GetContext(ctx,
		ds.reader(ctx),
		&hostConditionalAccessStatus,
		`SELECT
			mcphs.host_id, mcphs.device_id, mcphs.compliant, mcphs.created_at, mcphs.updated_at,
			h.os_version, hdn.display_name
		FROM microsoft_compliance_partner_host_statuses mcphs
		JOIN host_display_names hdn ON hdn.host_id=mcphs.host_id
		JOIN hosts h ON h.id=mcphs.host_id
		WHERE mcphs.host_id = ?`,
		hostID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostConditionalAccessStatus").WithID(hostID))
		}
	}
	return &hostConditionalAccessStatus, nil
}

func (ds *Datastore) CreateHostConditionalAccessStatus(ctx context.Context, hostID uint, deviceID string) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO microsoft_compliance_partner_host_statuses (host_id, device_id) VALUES (?, ?)`,
		hostID, deviceID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create host conditional access status")
	}
	return nil
}

func (ds *Datastore) SetHostConditionalAccessStatus(ctx context.Context, hostID uint, compliant bool) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE microsoft_compliance_partner_host_statuses SET compliant = ? WHERE host_id = ?;`,
		compliant, hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host conditional access status")
	}
	return nil
}
