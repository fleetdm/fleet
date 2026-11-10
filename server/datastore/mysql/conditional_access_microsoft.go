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
	encryptedSecret, err := encrypt([]byte(proxyServerSecret), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypt proxy server secret")
	}
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
			tenantID, encryptedSecret,
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
	return getConditionalAccessMicrosoft(ctx, ds.reader(ctx), ds.serverPrivateKey)
}

func getConditionalAccessMicrosoft(ctx context.Context, q sqlx.QueryerContext, serverPrivateKey string) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
	var row struct {
		TenantID          string `db:"tenant_id"`
		ProxyServerSecret []byte `db:"proxy_server_secret"`
		SetupDone         bool   `db:"setup_done"`
	}
	err := sqlx.GetContext(
		ctx, q, &row,
		// Currently only one global integration is supported.
		`SELECT tenant_id, proxy_server_secret, setup_done FROM microsoft_compliance_partner_integrations;`,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MicrosoftCompliancePartnerIntegration"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting microsoft_compliance_partner_integrations")
	}
	secret := decryptOrPassthrough(row.ProxyServerSecret, serverPrivateKey)
	return &fleet.ConditionalAccessMicrosoftIntegration{
		TenantID:          row.TenantID,
		ProxyServerSecret: string(secret),
		SetupDone:         row.SetupDone,
	}, nil
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

func (ds *Datastore) CreateHostConditionalAccessStatus(ctx context.Context, hostID uint, deviceID string, userPrincipalName string, verified bool) (bool, error) {
	// Most of the time this information won't change, so use the reader first.
	var current struct {
		DeviceID          string `db:"device_id"`
		UserPrincipalName string `db:"user_principal_name"`
		Verified          bool   `db:"verified"`
	}
	err := sqlx.GetContext(ctx,
		ds.reader(ctx),
		&current,
		`SELECT device_id, user_principal_name, verified FROM microsoft_compliance_partner_host_statuses WHERE host_id = ?`,
		hostID,
	)
	switch {
	case err == nil:
		if deviceID == current.DeviceID && userPrincipalName == current.UserPrincipalName && verified == current.Verified {
			// Nothing to do, the Entra account on the device is still the same.
			return false, nil
		}
		// If we got here it means the host's Entra data has changed, so we will override the host's status row.
	case errors.Is(err, sql.ErrNoRows):
		// OK, let's create one.
	default:
		return false, ctxerr.Wrap(ctx, err, "failed to get microsoft_compliance_partner_host_statuses")
	}

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// A device ID belongs to one host; it moves to another host only after the old host is
		// deleted. The locking read keeps a concurrent claim from assigning it twice.
		var others []struct {
			HostID   uint `db:"host_id"`
			Verified bool `db:"verified"`
		}
		if err := sqlx.SelectContext(ctx, tx, &others,
			`SELECT host_id, verified FROM microsoft_compliance_partner_host_statuses WHERE device_id = ? AND host_id <> ? FOR UPDATE`,
			deviceID, hostID,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "check Entra device ID assignment")
		}
		if len(others) > 0 {
			// A verified claim proves the device ID is this host's, so it replaces unverified
			// claims, but not another verified one.
			otherHostIDs := make([]uint, 0, len(others))
			for _, o := range others {
				if !verified || o.Verified {
					return ctxerr.Wrap(ctx, &fleet.ConflictError{Message: "Entra device ID is already assigned to another host"})
				}
				otherHostIDs = append(otherHostIDs, o.HostID)
			}
			stmt, args, err := sqlx.In(`DELETE FROM microsoft_compliance_partner_host_statuses WHERE host_id IN (?)`, otherHostIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build delete unverified holders query")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete unverified Entra device ID holders")
			}
			ds.logger.InfoContext(ctx, "moved Entra device ID from hosts with an unverified claim",
				"host_id", hostID, "previous_host_ids", otherHostIDs)
		}

		// Create or override existing row for the host.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO microsoft_compliance_partner_host_statuses
			(host_id, device_id, user_principal_name, verified)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			device_id = VALUES(device_id),
			user_principal_name = VALUES(user_principal_name),
			verified = VALUES(verified),
			managed = NULL,
			compliant = NULL`,
			hostID, deviceID, userPrincipalName, verified,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "create host conditional access status")
		}
		return nil
	}); err != nil {
		return false, err
	}
	return true, nil
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
