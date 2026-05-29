package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// UpsertHostGoogleCloudIdentityResolution records that Fleet has detected an
// EV-resolved Workspace identity on a host. Called from the osquery
// distributed-query ingest path with a raw_resource_id (from
// accounts.json) and the user's Workspace email. The canonical
// `device_user_resource` is filled in lazily by the resolution layer on
// first sync.
//
// On an identical (host, raw_resource_id, partner_suffix) triple this is a
// no-op except for updating the email if it changed.
func (ds *Datastore) UpsertHostGoogleCloudIdentityResolution(
	ctx context.Context,
	hostID uint,
	rawResourceID string,
	workspaceEmail string,
	partnerSuffix string,
) error {
	var existing struct {
		WorkspaceEmail string `db:"workspace_email"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &existing,
		`SELECT workspace_email
		 FROM host_google_cloud_identity_clientstates
		 WHERE host_id = ? AND raw_resource_id = ? AND partner_suffix = ?`,
		hostID, rawResourceID, partnerSuffix,
	)
	switch {
	case err == nil:
		if existing.WorkspaceEmail == workspaceEmail {
			// Identical row exists; nothing to do.
			return nil
		}
		// Email changed (rare). Update without resetting last_* — the
		// underlying deviceUser is the same.
		if _, err := ds.writer(ctx).ExecContext(ctx,
			`UPDATE host_google_cloud_identity_clientstates
			 SET workspace_email = ?
			 WHERE host_id = ? AND raw_resource_id = ? AND partner_suffix = ?`,
			workspaceEmail, hostID, rawResourceID, partnerSuffix,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "update host_google_cloud_identity_clientstates email")
		}
		return nil
	case errors.Is(err, sql.ErrNoRows):
		// Insert below.
	default:
		return ctxerr.Wrap(ctx, err, "load host_google_cloud_identity_clientstates")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_google_cloud_identity_clientstates
			(host_id, raw_resource_id, workspace_email, partner_suffix)
		VALUES (?, ?, ?, ?)`,
		hostID, rawResourceID, workspaceEmail, partnerSuffix,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "insert host_google_cloud_identity_clientstates")
	}
	return nil
}

// SetHostGoogleCloudIdentityResolvedDeviceUser records the canonical
// `devices/{deviceId}/deviceUsers/{deviceUserId}` resource name that
// resolution-layer's lookup call discovered for a given (host, raw_resource_id,
// partner_suffix) row. Called once per row, on first successful resolution.
func (ds *Datastore) SetHostGoogleCloudIdentityResolvedDeviceUser(
	ctx context.Context,
	hostID uint,
	rawResourceID string,
	partnerSuffix string,
	deviceUserResource string,
) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_google_cloud_identity_clientstates
		SET device_user_resource = ?
		WHERE host_id = ? AND raw_resource_id = ? AND partner_suffix = ?`,
		deviceUserResource, hostID, rawResourceID, partnerSuffix,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host_google_cloud_identity_clientstates device_user_resource")
	}
	return nil
}

// LoadHostGoogleCloudIdentityClientStates returns every ClientState row Fleet
// is tracking for a host. Empty slice (not nil error) is returned when the
// host has no resolved deviceUsers.
func (ds *Datastore) LoadHostGoogleCloudIdentityClientStates(
	ctx context.Context, hostID uint,
) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
	var rows []*fleet.HostGoogleCloudIdentityClientState
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT
			id, host_id, raw_resource_id, device_user_resource, workspace_email, partner_suffix,
			last_compliant, last_managed, last_score_reason, last_etag,
			last_synced_at, created_at, updated_at
		FROM host_google_cloud_identity_clientstates
		WHERE host_id = ?`,
		hostID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host_google_cloud_identity_clientstates")
	}
	return rows, nil
}

// SetHostGoogleCloudIdentityClientState records the values Fleet just
// successfully PATCHed to Cloud Identity, so the next sync can diff against
// them. Keyed by (host_id, raw_resource_id, partner_suffix) since
// device_user_resource may have been NULL when the row was created.
func (ds *Datastore) SetHostGoogleCloudIdentityClientState(
	ctx context.Context,
	hostID uint,
	rawResourceID string,
	partnerSuffix string,
	managed bool,
	compliant bool,
	scoreReason string,
	etag string,
) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_google_cloud_identity_clientstates
		SET last_managed = ?, last_compliant = ?, last_score_reason = ?, last_etag = ?, last_synced_at = ?
		WHERE host_id = ? AND raw_resource_id = ? AND partner_suffix = ?`,
		managed, compliant, scoreReason, etag, time.Now().UTC(),
		hostID, rawResourceID, partnerSuffix,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host_google_cloud_identity_clientstates last_*")
	}
	return nil
}

// DeleteHostGoogleCloudIdentityClientStates removes all rows for a host. Used
// when the integration is disabled for the host's team (or globally) so the
// next enable cycle re-resolves from scratch. The actual ClientState resource
// on Google's side is retracted by a separate sync step.
func (ds *Datastore) DeleteHostGoogleCloudIdentityClientStates(
	ctx context.Context, hostID uint,
) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_google_cloud_identity_clientstates WHERE host_id = ?`,
		hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_google_cloud_identity_clientstates")
	}
	return nil
}
