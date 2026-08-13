package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// microsoftGraphCredentialRow mirrors the mdm_microsoft_graph_credentials table. The secret is read as the raw encrypted
// blob and decrypted into fleet.MicrosoftGraphCredential.ClientSecret, so the encrypted form never escapes this file.
type microsoftGraphCredentialRow struct {
	TenantID          string     `db:"tenant_id"`
	ClientID          string     `db:"client_id"`
	ClientSecret      []byte     `db:"client_secret"`
	CredentialInvalid bool       `db:"credential_invalid"`
	LastSyncedAt      *time.Time `db:"last_synced_at"`
	LastSyncError     *string    `db:"last_sync_error"`
}

func (r microsoftGraphCredentialRow) toCredential(secret string) *fleet.MicrosoftGraphCredential {
	return &fleet.MicrosoftGraphCredential{
		TenantID:          r.TenantID,
		ClientID:          r.ClientID,
		ClientSecret:      secret,
		CredentialInvalid: r.CredentialInvalid,
		LastSyncedAt:      r.LastSyncedAt,
		LastSyncError:     r.LastSyncError,
	}
}

// ListMicrosoftGraphCredentials returns every stored Graph credential with its client secret decrypted.
func (ds *Datastore) ListMicrosoftGraphCredentials(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
	const stmt = `
SELECT tenant_id, client_id, client_secret, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
ORDER BY tenant_id`

	var rows []microsoftGraphCredentialRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credentials")
	}

	creds := make([]*fleet.MicrosoftGraphCredential, 0, len(rows))
	for _, row := range rows {
		secret, err := decrypt(row.ClientSecret, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decrypt microsoft graph client secret for tenant %s", row.TenantID)
		}
		creds = append(creds, row.toCredential(string(secret)))
	}
	return creds, nil
}

// ListMicrosoftGraphCredentialMetadata returns the stored credentials without their client secrets.
func (ds *Datastore) ListMicrosoftGraphCredentialMetadata(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
	const stmt = `
SELECT tenant_id, client_id, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
ORDER BY tenant_id`

	var creds []*fleet.MicrosoftGraphCredential
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &creds, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credential metadata")
	}
	return creds, nil
}

// ReplaceMicrosoftGraphCredentials reconciles the stored credentials in one transaction: every credential in upsert is
// stored, and every tenant in deleteTenantIDs is removed.
func (ds *Datastore) ReplaceMicrosoftGraphCredentials(
	ctx context.Context,
	upsert []*fleet.MicrosoftGraphCredential,
	deleteTenantIDs []string,
) error {
	if len(upsert) == 0 && len(deleteTenantIDs) == 0 {
		return nil
	}

	// Encrypt before opening the transaction so a bad private key fails without holding one open.
	encryptedSecrets := make([][]byte, len(upsert))
	for i, cred := range upsert {
		encrypted, err := encrypt([]byte(cred.ClientSecret), ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypt microsoft graph client secret with datastore.serverPrivateKey")
		}
		encryptedSecrets[i] = encrypted
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Storing a credential resets all of its sync state.
		const upsertStmt = `
INSERT INTO mdm_microsoft_graph_credentials (tenant_id, client_id, client_secret)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
	client_id = VALUES(client_id),
	client_secret = VALUES(client_secret),
	last_synced_at = NULL,
	credential_invalid = 0,
	last_sync_error = NULL`

		for i, cred := range upsert {
			if _, err := tx.ExecContext(ctx, upsertStmt, cred.TenantID, cred.ClientID, encryptedSecrets[i]); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert microsoft graph credential")
			}
		}

		if len(deleteTenantIDs) > 0 {
			deleteStmt, args, err := sqlx.In(`DELETE FROM mdm_microsoft_graph_credentials WHERE tenant_id IN (?)`, deleteTenantIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build delete microsoft graph credentials statement")
			}
			if _, err := tx.ExecContext(ctx, deleteStmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete microsoft graph credentials")
			}
		}

		return nil
	})
}

// SetMicrosoftGraphCredentialInvalid flips the per-tenant flag reporting that a credential needs an admin's attention.
func (ds *Datastore) SetMicrosoftGraphCredentialInvalid(ctx context.Context, tenantID string, invalid bool) (bool, error) {
	const stmt = `UPDATE mdm_microsoft_graph_credentials SET credential_invalid = ? WHERE tenant_id = ? AND credential_invalid != ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, invalid, tenantID, invalid)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "update mdm_microsoft_graph_credentials credential_invalid")
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// RecordMicrosoftGraphSyncResult stamps the outcome of a sync pass for a tenant. A nil syncErr records a success and
// clears any previous error; a non-nil one records the message for display alongside the credential.
func (ds *Datastore) RecordMicrosoftGraphSyncResult(ctx context.Context, tenantID string, syncErr *string) error {
	const stmt = `UPDATE mdm_microsoft_graph_credentials SET last_synced_at = NOW(6), last_sync_error = ? WHERE tenant_id = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, syncErr, tenantID); err != nil {
		return ctxerr.Wrap(ctx, err, "record microsoft graph sync result")
	}
	return nil
}

const hostAutopilotDeviceColumns = 6

// hostAutopilotDeviceBatchSize is how many devices go into one INSERT statement. At 6 placeholders each that is 6k per
// statement, well under MySQL's 65535 limit. A var so tests can shrink it and exercise the batch boundary.
var hostAutopilotDeviceBatchSize = 1000

// BatchUpsertHostAutopilotDevices stores the Autopilot metadata for many hosts. Re-inserting after a soft delete clears
// deleted_at, so a device that leaves and later rejoins the Autopilot list becomes live again rather than staying
// invisible behind a stale tombstone.
func (ds *Datastore) BatchUpsertHostAutopilotDevices(ctx context.Context, devices []*fleet.HostAutopilotDevice) error {
	return batchUpsertHostAutopilotDevicesDB(ctx, ds.writer(ctx), devices)
}

func batchUpsertHostAutopilotDevicesDB(ctx context.Context, tx sqlx.ExtContext, devices []*fleet.HostAutopilotDevice) error {
	const stmt = `
INSERT INTO host_autopilot_devices
	(host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id)
VALUES %s
ON DUPLICATE KEY UPDATE
	autopilot_device_id = VALUES(autopilot_device_id),
	entra_device_id = VALUES(entra_device_id),
	group_tag = VALUES(group_tag),
	hardware_serial = VALUES(hardware_serial),
	tenant_id = VALUES(tenant_id),
	deleted_at = NULL`

	err := common_mysql.BatchProcessSimple(devices, hostAutopilotDeviceBatchSize, func(batch []*fleet.HostAutopilotDevice) error {
		args := make([]any, 0, len(batch)*hostAutopilotDeviceColumns)
		for _, dev := range batch {
			args = append(args, dev.HostID, dev.AutopilotDeviceID, dev.EntraDeviceID, dev.GroupTag, dev.HardwareSerial, dev.TenantID)
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(batch)), ",")

		if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})

	return ctxerr.Wrap(ctx, err, "batch upsert host autopilot devices")
}

// BatchSoftDeleteHostAutopilotDevices tombstones the Autopilot records for the given hosts, marking the devices as no
// longer present in the tenant's Autopilot registry. The host row itself is untouched: a device that is deregistered
// from Autopilot stops being a pending host, but an already-enrolled host keeps reporting in.
func (ds *Datastore) BatchSoftDeleteHostAutopilotDevices(ctx context.Context, hostIDs []uint) error {
	const stmt = `
UPDATE host_autopilot_devices
SET deleted_at = NOW(6)
WHERE deleted_at IS NULL AND host_id IN (?)`

	err := common_mysql.BatchProcessSimple(hostIDs, hostAutopilotDeviceBatchSize, func(batch []uint) error {
		expanded, args, err := sqlx.In(stmt, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, expanded, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})

	return ctxerr.Wrap(ctx, err, "batch soft delete host autopilot devices")
}

// ListHostAutopilotDevices returns the live (not soft-deleted) Autopilot records for a tenant.
func (ds *Datastore) ListHostAutopilotDevices(ctx context.Context, tenantID string) ([]*fleet.HostAutopilotDevice, error) {
	const stmt = `
SELECT host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id
FROM host_autopilot_devices
WHERE tenant_id = ? AND deleted_at IS NULL
ORDER BY host_id`

	var devices []*fleet.HostAutopilotDevice
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &devices, stmt, tenantID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host autopilot devices")
	}
	return devices, nil
}

// GetHostAutopilotDevice returns the live Autopilot record for a host, or a not-found error. A soft-deleted record
// reads as not found: the device has left Autopilot, and only the host survives.
func (ds *Datastore) GetHostAutopilotDevice(ctx context.Context, hostID uint) (*fleet.HostAutopilotDevice, error) {
	const stmt = `
SELECT host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id
FROM host_autopilot_devices
WHERE host_id = ? AND deleted_at IS NULL`

	var device fleet.HostAutopilotDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &device, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostAutopilotDevice").WithID(hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host autopilot device")
	}
	return &device, nil
}

// IngestWindowsAutopilotDevices creates a pending Windows host for every device whose hardware serial has no host yet,
// and stores the Autopilot metadata for every device passed in. HostID on the input is ignored and resolved from the
// serial, so the caller does not have to know which devices are new. A device whose serial already belongs to a host
// (typically one that has since enrolled) only gets its Autopilot metadata refreshed.
func (ds *Datastore) IngestWindowsAutopilotDevices(ctx context.Context, devices []*fleet.HostAutopilotDevice) error {
	if len(devices) == 0 {
		return nil
	}

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config for windows autopilot ingest")
	}
	// A pending Autopilot host must point at the same mobile_device_management_solutions row as an enrolled Windows
	// host. That table is keyed on (name, server_url), and Windows enrollment registers the discovery URL, so
	// resolving anything else here would file pending hosts under a second "Fleet" solution.
	serverURL, err := microsoft_mdm.ResolveWindowsMDMDiscovery(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolve windows mdm discovery url")
	}

	// Collapse duplicate serials before any write. The Autopilot device ID is unique and we now store it, but it
	// cannot be the key here: a pending host is created before the device ever boots, when its serial is the only
	// identity it has, and host_autopilot_devices is keyed by host_id. Two records for one serial therefore have to
	// become one host.
	//
	// Keying on the Autopilot ID instead would mean two host rows sharing a serial, which the fleetd enrollment path
	// cannot resolve: it matches on serial with ORDER BY h.id LIMIT 1 and orbit never sees the Autopilot ID. It would
	// also pick the wrong one, since that branch requires enrolled = 0 and the MDM path would already have linked the
	// correct host by Autopilot ID. That is worse than collapsing. Revisit if orbit ever reports the ID; see
	// ai/graph/autopilot-enrollment-matching-research.md.
	//
	// Done here as well as in the sync so the invariant holds for any caller; the sync repeats it only to log a count.
	deduped, _ := fleet.DedupeAutopilotDevicesBySerial(devices)

	// One transaction per chunk rather than one for the whole list. A tenant can register 100k+ devices, and the
	// first sync creates a host, an MDM row and two label memberships for each. Doing that in a single transaction
	// would hold row locks across the hosts table for the duration and ship one enormous binlog event, stalling
	// replicas and concurrent enrollments. Chunking is safe because the work is idempotent: a chunk that fails is
	// simply redone on the next sync.
	return common_mysql.BatchProcessSimple(deduped, hostAutopilotDeviceBatchSize, func(batch []*fleet.HostAutopilotDevice) error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			return ingestWindowsAutopilotDevicesDB(ctx, tx, serverURL, batch)
		})
	})
}

func ingestWindowsAutopilotDevicesDB(ctx context.Context, tx sqlx.ExtContext, serverURL string, devices []*fleet.HostAutopilotDevice) error {
	{
		serials := make([]string, 0, len(devices))
		for _, dev := range devices {
			serials = append(serials, dev.HardwareSerial)
		}

		// Resolve before inserting so we know which hosts this pass creates: only those get a host_mdm row and label
		// memberships.
		hostIDBySerial, err := hostIDsBySerialDB(ctx, tx, serials)
		if err != nil {
			return err
		}

		newSerials := make([]string, 0, len(serials))
		for _, serial := range serials {
			if _, ok := hostIDBySerial[serial]; !ok {
				newSerials = append(newSerials, serial)
				// Guard against the same serial appearing twice in one Graph page.
				hostIDBySerial[serial] = 0
			}
		}

		if len(newSerials) > 0 {
			if err := insertPendingWindowsAutopilotHostsDB(ctx, tx, newSerials); err != nil {
				return err
			}
			created, err := hostIDsBySerialDB(ctx, tx, newSerials)
			if err != nil {
				return err
			}
			newHostIDs := make([]uint, 0, len(created))
			for serial, id := range created {
				hostIDBySerial[serial] = id
				newHostIDs = append(newHostIDs, id)
			}
			if err := upsertWindowsAutopilotHostMDMInfoDB(ctx, tx, serverURL, newHostIDs); err != nil {
				return err
			}
			if err := upsertWindowsAutopilotHostLabelsDB(ctx, tx, newHostIDs); err != nil {
				return err
			}
			// Without a host_display_names row the host is invisible in the UI: the host list INNER JOINs that table
			// whenever it sorts by display_name, which is the default. ADE ingestion does the same thing.
			displayNameHosts := make([]fleet.Host, 0, len(created))
			for serial, id := range created {
				displayNameHosts = append(displayNameHosts, fleet.Host{ID: id, HardwareSerial: serial})
			}
			if err := insertHostDisplayNamesIfAbsent(ctx, tx, displayNameHosts...); err != nil {
				return ctxerr.Wrap(ctx, err, "insert display names for pending autopilot hosts")
			}
		}

		resolved := make([]*fleet.HostAutopilotDevice, 0, len(devices))
		for _, dev := range devices {
			hostID, ok := hostIDBySerial[dev.HardwareSerial]
			if !ok || hostID == 0 {
				// The insert above should have covered every serial; skip rather than write host_id 0.
				continue
			}
			copied := *dev
			copied.HostID = hostID
			resolved = append(resolved, &copied)
		}
		return batchUpsertHostAutopilotDevicesDB(ctx, tx, resolved)
	}
}

// hostIDsBySerialDB maps hardware serials to host IDs. When more than one host shares a serial the lowest ID wins, so
// repeated syncs resolve to the same host.
func hostIDsBySerialDB(ctx context.Context, tx sqlx.ExtContext, serials []string) (map[string]uint, error) {
	byS := make(map[string]uint, len(serials))
	err := common_mysql.BatchProcessSimple(serials, hostAutopilotDeviceBatchSize, func(batch []string) error {
		// Scoped to Windows: a macOS host can carry a colliding serial, and claiming it would attach Autopilot metadata
		// to the wrong host and expose it to Autopilot removal. Newly created rows have platform 'windows' already.
		stmt, args, err := sqlx.In(
			`SELECT id, hardware_serial FROM hosts WHERE hardware_serial IN (?) AND platform = 'windows' ORDER BY id DESC`, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause")
		}
		var rows []struct {
			ID             uint   `db:"id"`
			HardwareSerial string `db:"hardware_serial"`
		}
		if err := sqlx.SelectContext(ctx, tx, &rows, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "select host ids by serial")
		}
		// Descending order means the lowest ID is written last and wins.
		for _, row := range rows {
			byS[row.HardwareSerial] = row.ID
		}
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "map host ids by serial")
	}
	return byS, nil
}

// insertPendingWindowsAutopilotHostsDB creates the bare host rows, matching how ADE ingestion creates pending Apple
// hosts. The never-timestamp sentinel on last_enrolled_at and detail_updated_at marks the host as never having checked
// in; it does not by itself protect the host from CleanupExpiredHosts, which treats the sentinel as null and falls
// through to created_at. Protection comes from the host_autopilot_devices cross-reference in that query, mirroring the
// host_dep_assignments one that protects ADE hosts.
func insertPendingWindowsAutopilotHostsDB(ctx context.Context, tx sqlx.ExtContext, serials []string) error {
	stmt := `
INSERT INTO hosts (hardware_serial, hardware_model, platform, last_enrolled_at, detail_updated_at, osquery_host_id, refetch_requested)
VALUES %s`

	err := common_mysql.BatchProcessSimple(serials, hostAutopilotDeviceBatchSize, func(batch []string) error {
		args := make([]any, 0, len(batch))
		for _, serial := range batch {
			args = append(args, serial)
		}
		values := strings.TrimSuffix(strings.Repeat(
			"(?, '', 'windows', '"+server.NeverTimestamp+"', '"+server.NeverTimestamp+"', NULL, 1),", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})
	return ctxerr.Wrap(ctx, err, "insert pending windows autopilot hosts")
}

// upsertWindowsAutopilotHostMDMInfoDB marks the hosts as pending: enrolled=0 with installed_from_dep=1 renders as
// "Pending" through the generated host_mdm.enrollment_status column, and flips to "On (automatic)" when the device
// enrolls for real.
func upsertWindowsAutopilotHostMDMInfoDB(ctx context.Context, tx sqlx.ExtContext, serverURL string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO mobile_device_management_solutions (name, server_url) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE server_url = VALUES(server_url)`,
		fleet.WellKnownMDMFleet, serverURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert windows mdm solution")
	}
	var mdmID int64
	if insertOnDuplicateDidInsertOrUpdate(result) {
		mdmID, _ = result.LastInsertId()
	} else {
		if err := sqlx.GetContext(ctx, tx, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`,
			fleet.WellKnownMDMFleet, serverURL); err != nil {
			return ctxerr.Wrap(ctx, err, "query windows mdm solution id")
		}
	}

	stmt := `
INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, mdm_id, is_server)
VALUES %s
ON DUPLICATE KEY UPDATE
	enrolled = VALUES(enrolled),
	server_url = VALUES(server_url),
	installed_from_dep = VALUES(installed_from_dep),
	mdm_id = VALUES(mdm_id)`

	err = common_mysql.BatchProcessSimple(hostIDs, hostAutopilotDeviceBatchSize, func(batch []uint) error {
		args := make([]any, 0, len(batch)*4)
		for _, hostID := range batch {
			args = append(args, hostID, serverURL, mdmID)
		}
		values := strings.TrimSuffix(strings.Repeat("(?, 0, ?, 1, ?, 0),", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})
	return ctxerr.Wrap(ctx, err, "upsert windows autopilot host mdm info")
}

// upsertWindowsAutopilotHostLabelsDB puts the pending hosts into the "All Hosts" and "MS Windows" builtin labels so
// they show up in host lists before osquery ever runs on the device.
func upsertWindowsAutopilotHostLabelsDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	var labels []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	if err := sqlx.SelectContext(ctx, tx, &labels,
		`SELECT id, name FROM labels WHERE label_type = ? AND name IN (?, ?)`,
		fleet.LabelTypeBuiltIn, fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameWindows); err != nil {
		return ctxerr.Wrap(ctx, err, "get builtin labels for windows autopilot hosts")
	}
	if len(labels) != 2 {
		// Builtin labels can be deleted. Skip rather than fail the whole sync over label membership.
		return nil
	}

	stmt := `INSERT INTO label_membership (host_id, label_id) VALUES %s ON DUPLICATE KEY UPDATE host_id = host_id`
	err := common_mysql.BatchProcessSimple(hostIDs, hostAutopilotDeviceBatchSize, func(batch []uint) error {
		args := make([]any, 0, len(batch)*len(labels)*2)
		for _, hostID := range batch {
			for _, label := range labels {
				args = append(args, hostID, label.ID)
			}
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(batch)*len(labels)), ",")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})
	return ctxerr.Wrap(ctx, err, "insert windows autopilot label membership")
}

// RemoveWindowsAutopilotHosts handles devices that have left the tenant's Autopilot registry. A host still in the
// pending state is deleted outright, because it only ever existed as a placeholder and will now never enroll. A host
// that has since enrolled is kept and only loses its Autopilot metadata, so a deregistered but live device keeps
// reporting in. Mirrors DeleteHostDEPAssignments.
func (ds *Datastore) RemoveWindowsAutopilotHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Chunked for the same reason as ingestion: deleting a host fans out across every table in hostRefs, so a tenant
	// that deregisters its whole fleet must not become one unbounded transaction.
	return common_mysql.BatchProcessSimple(hostIDs, hostAutopilotDeviceBatchSize, func(batch []uint) error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			return removeWindowsAutopilotHostsDB(ctx, tx, batch)
		})
	})
}

func removeWindowsAutopilotHostsDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	{
		// Pending state alone is not enough to delete. A device that installs fleetd but never MDM-enrolls keeps
		// enrolled = 0 and installed_from_dep = 1, so deleting on that predicate would destroy a live host and
		// everything hostRefs cleans up with it. Require that the host has never checked in by either route.
		stmt, args, err := sqlx.In(`
SELECT h.id
FROM hosts h
JOIN host_mdm hm ON hm.host_id = h.id
WHERE h.id IN (?) AND hm.enrolled = 0 AND hm.installed_from_dep = 1
	AND h.osquery_host_id IS NULL AND h.orbit_node_key IS NULL`, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause for pending autopilot hosts")
		}
		var pendingHostIDs []uint
		if err := sqlx.SelectContext(ctx, tx, &pendingHostIDs, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "select pending autopilot hosts")
		}

		// Tombstone every Autopilot row first. The rows for deleted hosts go away with the host through hostRefs, but
		// tombstoning first keeps the enrolled-host case correct if the delete below fails and the tx retries.
		tombstone, args, err := sqlx.In(
			`UPDATE host_autopilot_devices SET deleted_at = NOW(6) WHERE deleted_at IS NULL AND host_id IN (?)`, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause for autopilot tombstone")
		}
		if _, err := tx.ExecContext(ctx, tombstone, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "tombstone host autopilot devices")
		}

		if len(pendingHostIDs) == 0 {
			return nil
		}
		return deleteHosts(ctx, tx, pendingHostIDs)
	}
}

// UpdateMicrosoftGraphCredentialInvalidAggregate recomputes MDM.MicrosoftGraphCredentialInvalid from the credentials
// table and saves the app config only when the value actually changed.
//
// The flag is stored rather than derived on read so that GET /config never joins the credentials table, which means it
// is only as fresh as its last recomputation. Both the credential write paths and the sync cron have to call this, and
// they live in different packages: the premium service cannot be imported by cron, so the shared logic belongs here.
// The read is forced to the primary because every caller reaches this immediately after writing.
func (ds *Datastore) UpdateMicrosoftGraphCredentialInvalidAggregate(ctx context.Context) error {
	ctx = ctxdb.RequirePrimary(ctx, true)

	var anyInvalid bool
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &anyInvalid,
		`SELECT EXISTS(SELECT 1 FROM mdm_microsoft_graph_credentials WHERE credential_invalid = 1)`); err != nil {
		return ctxerr.Wrap(ctx, err, "check for invalid microsoft graph credentials")
	}

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if appCfg.MDM.MicrosoftGraphCredentialInvalid == anyInvalid {
		return nil
	}

	appCfg.MDM.MicrosoftGraphCredentialInvalid = anyInvalid
	if err := ds.SaveAppConfig(ctx, appCfg); err != nil {
		return ctxerr.Wrap(ctx, err, "save app config with microsoft graph credential status")
	}
	return nil
}

// HostIDByAutopilotDeviceID resolves a pending Autopilot host from the Autopilot device ID (the ZTDID). The device
// supplies this at Windows MDM enrollment and Microsoft Graph returns the same value, so it links an enrollment to its
// pending host exactly, without depending on a hardware serial that can be duplicated or a placeholder.
func (ds *Datastore) HostIDByAutopilotDeviceID(ctx context.Context, autopilotDeviceID string) (uint, error) {
	const stmt = `
SELECT host_id FROM host_autopilot_devices
WHERE autopilot_device_id = ? AND deleted_at IS NULL
ORDER BY host_id LIMIT 1`

	var hostID uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostID, stmt, autopilotDeviceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, notFound("HostAutopilotDevice").WithMessage(autopilotDeviceID))
		}
		return 0, ctxerr.Wrap(ctx, err, "get host id by autopilot device id")
	}
	return hostID, nil
}
