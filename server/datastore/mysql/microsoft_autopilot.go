package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
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
func (ds *Datastore) ListMicrosoftGraphCredentialMetadata(ctx context.Context) ([]*fleet.MicrosoftGraphCredentialMetadata, error) {
	const stmt = `
SELECT tenant_id, client_id, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
ORDER BY tenant_id`

	// Non-nil so the endpoint serializes an empty list as [] rather than null.
	creds := make([]*fleet.MicrosoftGraphCredentialMetadata, 0)
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
func (ds *Datastore) SetMicrosoftGraphCredentialInvalid(ctx context.Context, tenantID string, invalid bool) error {
	const stmt = `UPDATE mdm_microsoft_graph_credentials SET credential_invalid = ? WHERE tenant_id = ? AND credential_invalid != ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, invalid, tenantID, invalid); err != nil {
		return ctxerr.Wrap(ctx, err, "update mdm_microsoft_graph_credentials credential_invalid")
	}
	return nil
}

// RecordMicrosoftGraphSyncResult stamps the outcome of a sync pass for a tenant. A nil syncErr records a success and
// clears any previous error; a non-nil one records the message for display alongside the credential. last_synced_at
// means "last successful sync."
func (ds *Datastore) RecordMicrosoftGraphSyncResult(ctx context.Context, tenantID string, syncErr *string) error {
	const stmt = `
		UPDATE mdm_microsoft_graph_credentials
		SET last_synced_at = IF(? IS NULL, NOW(6), last_synced_at), last_sync_error = ?
		WHERE tenant_id = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, syncErr, syncErr, tenantID); err != nil {
		return ctxerr.Wrap(ctx, err, "record microsoft graph sync result")
	}
	return nil
}

const hostAutopilotDeviceColumns = 6

// hostAutopilotDeviceBatchSize is how many devices one ingest transaction handles, which makes it the size of every
// statement inside that transaction. The widest is the 6-column host_autopilot_devices upsert at 6k placeholders, well
// under MySQL's 65535 limit. Chunking happens once, at the entry point; helpers called inside a chunk take the slice as
// given rather than re-batching it, since a second pass at the same size can only ever yield one chunk. A var so tests
// can shrink it and exercise the batch boundary.
var hostAutopilotDeviceBatchSize = 1000

// hostAutopilotDeviceReadBatchSize is how many IDs go into one lookup IN clause. Reads carry a single placeholder per
// row rather than six and take no locks, so they chunk far more coarsely than writes: at 100k devices this is 10
// queries instead of 100. Kept well below MySQL's 65535 placeholder ceiling, above which the IN list also starts to
// cost real parse time for no further round-trip saving. A var so tests can shrink it and exercise the boundary.
var hostAutopilotDeviceReadBatchSize = 10000

func batchUpsertHostAutopilotDevicesDB(ctx context.Context, tx sqlx.ExtContext, devices []*fleet.HostAutopilotDevice) error {
	if len(devices) == 0 {
		return nil
	}

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

	args := make([]any, 0, len(devices)*hostAutopilotDeviceColumns)
	for _, dev := range devices {
		args = append(args, dev.HostID, dev.AutopilotDeviceID, dev.EntraDeviceID, dev.GroupTag, dev.HardwareSerial, dev.TenantID)
	}
	values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(devices)), ",")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host autopilot devices")
	}
	return nil
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

// IngestWindowsAutopilotDevices creates a pending Windows host for every Autopilot device Fleet has no host for yet,
// and stores the Autopilot metadata for every device passed in. HostID on the input is ignored: it is resolved from
// the Autopilot device ID, so the caller does not have to know which devices are new, and two devices sharing a
// hardware serial stay two hosts. A device that already has a host (typically one that has since enrolled) only gets
// its Autopilot metadata refreshed.
func (ds *Datastore) IngestWindowsAutopilotDevices(ctx context.Context, devices []*fleet.HostAutopilotDevice) error {
	if len(devices) == 0 {
		return nil
	}

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config for windows autopilot ingest")
	}
	// mobile_device_management_solutions is unique on (name, server_url) and the name is always Fleet, so the URL
	// alone decides which solution row a host is filed under. Nothing has been reported about these devices yet, so
	// this URL is derived from config rather than observed.
	serverURL, err := microsoft_mdm.ResolveWindowsMDMDiscovery(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolve windows mdm discovery url")
	}

	// Resolve which devices Fleet already has a host for once, up front. This sync is the only production writer of
	// host_autopilot_devices, so nothing can claim a device between this read and the writes below.
	ztdIDs := make([]string, 0, len(devices))
	for _, dev := range devices {
		ztdIDs = append(ztdIDs, dev.AutopilotDeviceID)
	}
	hostIDByZTD, err := hostIDsByAutopilotDeviceIDDB(ctx, ds.reader(ctx), ztdIDs)
	if err != nil {
		return err
	}

	// One row per Fleet deployment, so resolve it once for the whole ingest rather than per chunk. The shared helper
	// reads the replica first and only writes on a miss, which after the first pass is every pass.
	mdmID, err := ds.getOrInsertMDMSolution(ctx, serverURL, fleet.WellKnownMDMFleet)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or insert windows mdm solution")
	}

	// A host created here is placed straight into the Windows enrollment default fleet.
	defaultTeamID, _, err := ds.GetWindowsEnrollmentDefaultFleet(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows enrollment default fleet")
	}

	builtinLabelIDs, err := windowsAutopilotBuiltinLabelIDsDB(ctx, ds.reader(ctx), ds.logger)
	if err != nil {
		return err
	}

	ingest := windowsAutopilotIngest{
		serverURL:       serverURL,
		mdmID:           mdmID,
		defaultTeamID:   defaultTeamID,
		builtinLabelIDs: builtinLabelIDs,
		hostIDByZTD:     hostIDByZTD,
		logger:          ds.logger,
	}

	// One transaction per chunk rather than one for the whole list.
	return common_mysql.BatchProcessSimple(devices, hostAutopilotDeviceBatchSize, func(batch []*fleet.HostAutopilotDevice) error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			return ingestWindowsAutopilotDevicesDB(ctx, tx, ingest, batch)
		})
	})
}

// windowsAutopilotIngest carries the values an ingest resolves once, up front, and every chunk then reuses. They are
// deliberately read outside the per-chunk transaction: none of them can change during a sync.
type windowsAutopilotIngest struct {
	serverURL     string
	mdmID         uint
	defaultTeamID *uint
	// builtinLabelIDs are the labels a pending Windows host joins. Empty when a builtin label has been deleted.
	builtinLabelIDs []uint
	// hostIDByZTD covers the whole device list, not just one chunk.
	hostIDByZTD map[string]uint
	logger      *slog.Logger
}

func ingestWindowsAutopilotDevicesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	ingest windowsAutopilotIngest,
	devices []*fleet.HostAutopilotDevice,
) error {
	// Resolution is keyed on the Autopilot device ID. Windows serials are not unique. Only devices that no Autopilot record already
	// resolves need a serial lookup, so the rest are left out of the query.
	serials := make([]string, 0, len(devices))
	for _, dev := range devices {
		if _, ok := ingest.hostIDByZTD[dev.AutopilotDeviceID]; ok {
			continue
		}
		serials = append(serials, dev.HardwareSerial)
	}
	// Hosts carrying this serial that no Autopilot record has claimed yet. Adopting one covers the machine that was
	// already enrolled in Fleet before it was registered in Autopilot; without it we would create a duplicate host.
	// Hosts already claimed by a different Autopilot record are excluded, which is what keeps two records for one
	// serial on two hosts.
	// Doing the read inside the TX to make sure we have the latest host data and to eliminate races.
	unclaimedBySerial, err := unclaimedHostIDsBySerialDB(ctx, tx, serials)
	if err != nil {
		return err
	}

	// host_autopilot_devices is keyed by host_id. Track device claims in memory to make sure a one-to-one match.
	claimed := make(map[uint]struct{}, len(devices))
	resolved := make([]*fleet.HostAutopilotDevice, 0, len(devices))
	toCreate := make([]*fleet.HostAutopilotDevice, 0, len(devices))
	for _, dev := range devices {
		if hostID, ok := ingest.hostIDByZTD[dev.AutopilotDeviceID]; ok {
			copied := *dev
			copied.HostID = hostID
			claimed[hostID] = struct{}{}
			resolved = append(resolved, &copied)
			continue
		}
		if hostID, ok := popUnclaimedHostID(unclaimedBySerial, dev.HardwareSerial, claimed); ok {
			copied := *dev
			copied.HostID = hostID
			claimed[hostID] = struct{}{}
			resolved = append(resolved, &copied)
			continue
		}
		toCreate = append(toCreate, dev)
	}

	if len(toCreate) > 0 {
		createSerials := make([]string, 0, len(toCreate))
		for _, dev := range toCreate {
			createSerials = append(createSerials, dev.HardwareSerial)
		}
		if err := insertPendingWindowsAutopilotHostsDB(ctx, tx, ingest.defaultTeamID, toCreate); err != nil {
			return err
		}
		// This query returns the host ids of newly created hosts.
		created, err := unclaimedHostIDsBySerialDB(ctx, tx, createSerials)
		if err != nil {
			return err
		}
		newHostIDs := make([]uint, 0, len(toCreate))
		newHosts := make([]fleet.Host, 0, len(toCreate))
		for _, dev := range toCreate {
			hostID, ok := popUnclaimedHostID(created, dev.HardwareSerial, claimed)
			if !ok {
				// Should not happen: we just inserted one host per device. Skip rather than write host_id 0.
				ingest.logger.ErrorContext(ctx, "no host resolved for a just-created autopilot device, skipping it",
					"autopilot_device_id", dev.AutopilotDeviceID, "hardware_serial", dev.HardwareSerial)
				ctxerr.Handle(ctx, ctxerr.New(ctx, "no host resolved for a just-created autopilot device"))
				continue
			}
			copied := *dev
			copied.HostID = hostID
			claimed[hostID] = struct{}{}
			resolved = append(resolved, &copied)
			newHostIDs = append(newHostIDs, hostID)
			// The model is what makes the display name render as "model (serial)"
			newHosts = append(newHosts, fleet.Host{
				ID: hostID, HardwareSerial: copied.HardwareSerial, HardwareModel: copied.HardwareModel,
			})
		}
		if err := upsertWindowsAutopilotHostMDMInfoDB(ctx, tx, ingest.serverURL, ingest.mdmID, newHostIDs); err != nil {
			return err
		}
		if err := upsertWindowsAutopilotHostLabelsDB(ctx, tx, ingest.builtinLabelIDs, newHostIDs); err != nil {
			return err
		}
		// Without a host_display_names row the host is invisible in the UI.
		if err := insertHostDisplayNamesIfAbsent(ctx, tx, newHosts...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert display names for pending autopilot hosts")
		}
	}

	return batchUpsertHostAutopilotDevicesDB(ctx, tx, resolved)
}

// popUnclaimedHostID takes the next host for a serial that no device in this batch has claimed yet, consuming it from
// the candidate list so a later device cannot take the same one.
func popUnclaimedHostID(bySerial map[string][]uint, serial string, claimed map[uint]struct{}) (uint, bool) {
	candidates := bySerial[serial]
	for i, hostID := range candidates {
		if _, taken := claimed[hostID]; taken {
			continue
		}
		bySerial[serial] = candidates[i+1:]
		return hostID, true
	}
	bySerial[serial] = nil
	return 0, false
}

// hostIDsByAutopilotDeviceIDDB maps Autopilot device IDs to the host already carrying that record.
func hostIDsByAutopilotDeviceIDDB(ctx context.Context, q sqlx.QueryerContext, ztdIDs []string) (map[string]uint, error) {
	byID := make(map[string]uint, len(ztdIDs))
	err := common_mysql.BatchProcessSimple(ztdIDs, hostAutopilotDeviceReadBatchSize, func(batch []string) error {
		stmt, args, err := sqlx.In(
			`SELECT host_id, autopilot_device_id FROM host_autopilot_devices
			 WHERE autopilot_device_id IN (?) AND deleted_at IS NULL`, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause for autopilot device ids")
		}
		var rows []struct {
			HostID            uint   `db:"host_id"`
			AutopilotDeviceID string `db:"autopilot_device_id"`
		}
		if err := sqlx.SelectContext(ctx, q, &rows, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "select host ids by autopilot device id")
		}
		for _, row := range rows {
			byID[row.AutopilotDeviceID] = row.HostID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return byID, nil
}

// unclaimedHostIDsBySerialDB returns, per serial, the Windows hosts that no live Autopilot record points at, in id
// order. Scoped to Windows because a macOS host can carry a colliding serial and must never be claimed.
func unclaimedHostIDsBySerialDB(ctx context.Context, tx sqlx.ExtContext, serials []string) (map[string][]uint, error) {
	// Every device in the chunk may already be resolved by its Autopilot device ID, which leaves nothing to look up.
	if len(serials) == 0 {
		return map[string][]uint{}, nil
	}

	stmt, args, err := sqlx.In(`
SELECT h.id, h.hardware_serial
FROM hosts h
LEFT JOIN host_autopilot_devices had ON had.host_id = h.id AND had.deleted_at IS NULL
WHERE h.hardware_serial IN (?) AND h.platform = 'windows' AND had.host_id IS NULL
ORDER BY h.id`, serials)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build IN clause for serials")
	}
	var rows []struct {
		ID             uint   `db:"id"`
		HardwareSerial string `db:"hardware_serial"`
	}
	if err := sqlx.SelectContext(ctx, tx, &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select unclaimed host ids by serial")
	}
	bySerial := make(map[string][]uint, len(rows))
	for _, row := range rows {
		bySerial[row.HardwareSerial] = append(bySerial[row.HardwareSerial], row.ID)
	}
	return bySerial, nil
}

// insertPendingWindowsAutopilotHostsDB creates the bare host rows. The never-timestamp sentinel on last_enrolled_at and
// detail_updated_at marks the host as never having checked in
func insertPendingWindowsAutopilotHostsDB(ctx context.Context, tx sqlx.ExtContext, teamID *uint, devices []*fleet.HostAutopilotDevice) error {
	if len(devices) == 0 {
		return nil
	}

	stmt := `
INSERT INTO hosts (hardware_serial, hardware_model, hardware_vendor, platform, last_enrolled_at, detail_updated_at, osquery_host_id, refetch_requested, team_id)
VALUES %s`

	args := make([]any, 0, len(devices)*4)
	for _, dev := range devices {
		args = append(args, dev.HardwareSerial, dev.HardwareModel, dev.HardwareVendor, teamID)
	}
	values := strings.TrimSuffix(strings.Repeat(
		"(?, ?, ?, 'windows', '"+server.NeverTimestamp+"', '"+server.NeverTimestamp+"', NULL, 1, ?),", len(devices)), ",")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert pending windows autopilot hosts")
	}
	return nil
}

// upsertWindowsAutopilotHostMDMInfoDB marks the hosts as pending: enrolled=0 with installed_from_dep=1 renders as
// "Pending" through the generated host_mdm.enrollment_status column, and flips to "On (automatic)" when the device
// enrolls for real.
func upsertWindowsAutopilotHostMDMInfoDB(ctx context.Context, tx sqlx.ExtContext, serverURL string, mdmID uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	stmt := `
INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, mdm_id, is_server)
VALUES %s
ON DUPLICATE KEY UPDATE
	enrolled = VALUES(enrolled),
	server_url = VALUES(server_url),
	installed_from_dep = VALUES(installed_from_dep),
	mdm_id = VALUES(mdm_id)`

	args := make([]any, 0, len(hostIDs)*3)
	for _, hostID := range hostIDs {
		args = append(args, hostID, serverURL, mdmID)
	}
	values := strings.TrimSuffix(strings.Repeat("(?, 0, ?, 1, ?, 0),", len(hostIDs)), ",")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert windows autopilot host mdm info")
	}
	return nil
}

// windowsAutopilotBuiltinLabelIDsDB resolves the builtin labels a pending Windows host joins, "All Hosts" and
// "MS Windows".
func windowsAutopilotBuiltinLabelIDsDB(ctx context.Context, q sqlx.QueryerContext, logger *slog.Logger) ([]uint, error) {
	var labelIDs []uint
	if err := sqlx.SelectContext(ctx, q, &labelIDs,
		`SELECT id FROM labels WHERE label_type = ? AND name IN (?, ?)`,
		fleet.LabelTypeBuiltIn, fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameWindows); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get builtin labels for windows autopilot hosts")
	}
	if len(labelIDs) != 2 {
		logger.ErrorContext(ctx, "expected 2 builtin labels for pending windows autopilot hosts, skipping label membership",
			"found", len(labelIDs))
		ctxerr.Handle(ctx, ctxerr.New(ctx, "expected 2 builtin labels for pending windows autopilot hosts"))
		return nil, nil
	}
	return labelIDs, nil
}

// upsertWindowsAutopilotHostLabelsDB puts the pending hosts into the builtin labels resolved by
// windowsAutopilotBuiltinLabelIDsDB so they show up in host lists before osquery ever runs on the device.
func upsertWindowsAutopilotHostLabelsDB(ctx context.Context, tx sqlx.ExtContext, labelIDs []uint, hostIDs []uint) error {
	if len(hostIDs) == 0 || len(labelIDs) == 0 {
		return nil
	}

	stmt := `INSERT INTO label_membership (host_id, label_id) VALUES %s ON DUPLICATE KEY UPDATE host_id = host_id`
	args := make([]any, 0, len(hostIDs)*len(labelIDs)*2)
	for _, hostID := range hostIDs {
		for _, labelID := range labelIDs {
			args = append(args, hostID, labelID)
		}
	}
	values := strings.TrimSuffix(strings.Repeat("(?,?),", len(hostIDs)*len(labelIDs)), ",")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert windows autopilot label membership")
	}
	return nil
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
	// Pending state alone is not enough to delete. A device that installs fleetd but never MDM-enrolls keeps
	// enrolled = 0 and installed_from_dep = 1, so deleting on that predicate would destroy a live host and
	// everything hostRefs cleans up with it. Require that the host has never checked in by either route.
	// FOR UPDATE locks the host rows that are about to be deleted.
	stmt, args, err := sqlx.In(`
SELECT h.id
FROM hosts h
JOIN host_mdm hm ON hm.host_id = h.id
WHERE h.id IN (?) AND hm.enrolled = 0 AND hm.installed_from_dep = 1
	AND h.osquery_host_id IS NULL AND h.orbit_node_key IS NULL
FOR UPDATE`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build IN clause for pending autopilot hosts")
	}
	var pendingHostIDs []uint
	if err := sqlx.SelectContext(ctx, tx, &pendingHostIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select pending autopilot hosts")
	}

	// Tombstone every Autopilot row first. We keep the rows for live hosts.
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

// UpdateMicrosoftGraphCredentialInvalidAggregate recomputes MDM.MicrosoftGraphCredentialInvalid from the credentials
// table and saves the app config only when the value actually changed.
func (ds *Datastore) UpdateMicrosoftGraphCredentialInvalidAggregate(ctx context.Context) error {
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
