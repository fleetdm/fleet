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
