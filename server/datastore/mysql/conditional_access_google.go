package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ConditionalAccessGoogleCreateIntegration(ctx context.Context, customerID, proxyServerSecret, oauthClientID string) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Only one global integration is supported.
		if _, err := tx.ExecContext(ctx, `DELETE FROM google_conditional_access_integrations`); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting google_conditional_access_integrations")
		}
		// Connecting is synchronous (verified with Google and registered with
		// fleetdm.com before this call), so the row is created as done.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO google_conditional_access_integrations (customer_id, proxy_server_secret, oauth_client_id, setup_done) VALUES (?, ?, ?, 1)`,
			customerID, proxyServerSecret, oauthClientID,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting google_conditional_access_integrations")
		}
		return nil
	})
}

func (ds *Datastore) ConditionalAccessGoogleGet(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	var integration fleet.ConditionalAccessGoogleIntegration
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &integration,
		`SELECT customer_id, proxy_server_secret, oauth_client_id, setup_done, setup_error FROM google_conditional_access_integrations`,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessIntegration"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_integrations")
	}
	return &integration, nil
}

func (ds *Datastore) ConditionalAccessGoogleSetSetupError(ctx context.Context, setupError *string) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE google_conditional_access_integrations SET setup_error = ?`, setupError,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "updating google_conditional_access_integrations setup_error")
	}
	return nil
}

func (ds *Datastore) ConditionalAccessGoogleDelete(ctx context.Context) error {
	if _, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM google_conditional_access_integrations`); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting google_conditional_access_integrations")
	}
	return nil
}
