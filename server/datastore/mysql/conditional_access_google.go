package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetGoogleConditionalAccessIntegration(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	var integration fleet.ConditionalAccessGoogleIntegration
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &integration, `
		SELECT customer_id, admin_email, proxy_server_secret, oauth_client_id, state, error_message
		FROM google_conditional_access_integrations
		WHERE id = 1`,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessIntegration"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_integrations")
	}
	return &integration, nil
}

func (ds *Datastore) SaveGoogleConditionalAccessIntegration(ctx context.Context, integration *fleet.ConditionalAccessGoogleIntegration) error {
	// The row is updated in place so created_at keeps the time the integration was
	// first set up.
	if _, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO google_conditional_access_integrations
			(id, customer_id, admin_email, proxy_server_secret, oauth_client_id, state, error_message)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			customer_id = VALUES(customer_id),
			admin_email = VALUES(admin_email),
			proxy_server_secret = VALUES(proxy_server_secret),
			oauth_client_id = VALUES(oauth_client_id),
			state = VALUES(state),
			error_message = VALUES(error_message)`,
		integration.CustomerID, integration.AdminEmail, integration.ProxyServerSecret,
		integration.OAuthClientID, integration.State, integration.ErrorMessage,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "saving google_conditional_access_integrations")
	}
	return nil
}

func (ds *Datastore) ListGoogleConditionalAccessDeviceUserNames(ctx context.Context) ([]string, error) {
	var names []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &names,
		`SELECT device_user_name FROM google_conditional_access_device_users ORDER BY device_user_name`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing google_conditional_access_device_users")
	}
	return names, nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessDevices(ctx context.Context) error {
	// The DeviceUser rows go with the device rows through the foreign key.
	if _, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM google_conditional_access_devices`); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting google_conditional_access_devices")
	}
	return nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessIntegration(ctx context.Context) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		for _, stmt := range []string{
			`DELETE FROM google_conditional_access_devices`,
			`DELETE FROM google_conditional_access_integrations`,
		} {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return ctxerr.Wrap(ctx, err, "deleting google conditional access rows")
			}
		}
		return nil
	})
}
