package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO microsoft_compliance_partner_integrations (tenant_id, proxy_server_secret) VALUES (?, ?);`,
			tenantID, proxyServerSecret,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting new microsoft_compliance_partner_integrations")
		}
		return nil
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
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Currently only one global integration is supported.
		if _, err := tx.ExecContext(ctx, `DELETE FROM microsoft_compliance_partner_integrations;`); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting microsoft_compliance_partner_integrations")
		}
		// Remove all last reported statuses.
		if _, err := tx.ExecContext(ctx, `DELETE FROM microsoft_compliance_partner_host_statuses;`); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting microsoft_compliance_partner_host_statuses")
		}
		return nil
	})
}

func (ds *Datastore) LoadHostConditionalAccessStatus(ctx context.Context, hostID uint) (*fleet.HostConditionalAccessStatus, error) {
	var hostConditionalAccessStatus fleet.HostConditionalAccessStatus
	if err := sqlx.GetContext(ctx,
		ds.reader(ctx),
		&hostConditionalAccessStatus,
		`SELECT
			mcphs.host_id, mcphs.device_id, mcphs.user_principal_name, mcphs.compliant, mcphs.created_at, mcphs.updated_at, mcphs.managed,
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
	hostConditionalAccessStatus.OSVersion = strings.TrimPrefix(hostConditionalAccessStatus.OSVersion, "macOS ")
	if strings.HasPrefix(hostConditionalAccessStatus.OSVersion, "Windows") {
		hostConditionalAccessStatus.OSVersion = extractWindowsBuildVersion(hostConditionalAccessStatus.OSVersion)
	}
	return &hostConditionalAccessStatus, nil
}

// extractWindowsBuildVersion extracts the build version from a Windows' hosts.os_version.
// For Windows hosts, hosts.os_version is of the form e.g.:
//   - "Windows Server 2025 Datacenter 24H2 10.0.26100.32230"
//   - "Windows 10 Pro 22H2 10.0.19045.6456"
func extractWindowsBuildVersion(osVersion string) string {
	osVersion = strings.TrimSpace(osVersion)
	lastSpace := strings.LastIndex(osVersion, " ")
	if lastSpace == -1 {
		return osVersion
	}
	return osVersion[lastSpace+1:]
}

func (ds *Datastore) CreateHostConditionalAccessStatus(ctx context.Context, hostID uint, deviceID string, userPrincipalName string) error {
	// Most of the time this information won't change, so use the reader first.
	var hostConditionalAccessStatus fleet.HostConditionalAccessStatus
	err := sqlx.GetContext(ctx,
		ds.reader(ctx),
		&hostConditionalAccessStatus,
		`SELECT device_id, user_principal_name FROM microsoft_compliance_partner_host_statuses WHERE host_id = ?`,
		hostID,
	)
	switch {
	case err == nil:
		if deviceID == hostConditionalAccessStatus.DeviceID && userPrincipalName == hostConditionalAccessStatus.UserPrincipalName {
			// Nothing to do, the Entra account on the device is still the same.
			return nil
		}
		// If we got here it means the host's Entra data has changed, so we will override the host's status row.
	case errors.Is(err, sql.ErrNoRows):
		// OK, let's create one.
	default:
		return ctxerr.Wrap(ctx, err, "failed to get microsoft_compliance_partner_host_statuses")
	}

	// Create or override existing row for the host.
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO microsoft_compliance_partner_host_statuses
		(host_id, device_id, user_principal_name)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
		device_id = VALUES(device_id),
		user_principal_name = VALUES(user_principal_name),
		managed = NULL,
		compliant = NULL`,
		hostID, deviceID, userPrincipalName,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create host conditional access status")
	}
	return nil
}

func (ds *Datastore) SetHostConditionalAccessStatus(ctx context.Context, hostID uint, managed, compliant bool) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE microsoft_compliance_partner_host_statuses SET managed = ?, compliant = ? WHERE host_id = ?;`,
		managed, compliant, hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host conditional access status")
	}
	return nil
}
