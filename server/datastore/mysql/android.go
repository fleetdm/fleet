package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewAndroidHost(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
	if !host.IsValid() {
		return nil, ctxerr.New(ctx, "valid Android host is required")
	}
	ds.setTimesToNonZero(host)

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new Android host get app config")
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// We use node_key as a unique identifier for the host table row. It matches: android/{enterpriseSpecificID}.
		stmt := `
		INSERT INTO hosts (
			node_key,
			hostname,
			computer_name,
			platform,
			os_version,
			build,
			memory,
			team_id,
			hardware_serial,
			cpu_type,
			hardware_model,
			hardware_vendor,
			detail_updated_at,
			label_updated_at
		) VALUES (
			:node_key,
			:hostname,
			:computer_name,
			:platform,
			:os_version,
			:build,
			:memory,
			:team_id,
			:hardware_serial,
			:cpu_type,
			:hardware_model,
			:hardware_vendor,
			:detail_updated_at,
			:label_updated_at
		) ON DUPLICATE KEY UPDATE
			hostname = VALUES(hostname),
			computer_name = VALUES(computer_name),
			platform = VALUES(platform),
			os_version = VALUES(os_version),
			build = VALUES(build),
			memory = VALUES(memory),
			team_id = VALUES(team_id),
			hardware_serial = VALUES(hardware_serial),
			cpu_type = VALUES(cpu_type),
			hardware_model = VALUES(hardware_model),
			hardware_vendor = VALUES(hardware_vendor),
			detail_updated_at = VALUES(detail_updated_at),
			label_updated_at = VALUES(label_updated_at)
		`
		result, err := sqlx.NamedExecContext(ctx, tx, stmt, map[string]interface{}{
			"node_key":          host.NodeKey,
			"hostname":          host.Hostname,
			"computer_name":     host.ComputerName,
			"platform":          host.Platform,
			"os_version":        host.OSVersion,
			"build":             host.Build,
			"memory":            host.Memory,
			"team_id":           host.TeamID,
			"hardware_serial":   host.HardwareSerial,
			"cpu_type":          host.CPUType,
			"hardware_model":    host.HardwareModel,
			"hardware_vendor":   host.HardwareVendor,
			"detail_updated_at": host.DetailUpdatedAt,
			"label_updated_at":  host.LabelUpdatedAt,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host")
		}
		id, _ := result.LastInsertId()
		host.Host.ID = uint(id) // nolint:gosec
		host.Device.HostID = host.Host.ID

		err = upsertHostDisplayNames(ctx, tx, *host.Host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host display name")
		}
		err = ds.insertAndroidHostLabelMembershipTx(ctx, tx, host.Host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host label membership")
		}

		// create entry in host_mdm as enrolled (manually), because currently all
		// android hosts are necessarily MDM-enrolled when created.
		if err := upsertAndroidHostMDMInfoDB(ctx, tx, appCfg.ServerSettings.ServerURL, false, true, host.Host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host MDM info")
		}

		host.Device, err = ds.Datastore.CreateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}
		return nil
	})
	return host, err
}

// setTimesToNonZero to avoid issues with MySQL.
func (ds *Datastore) setTimesToNonZero(host *fleet.AndroidHost) {
	if host.DetailUpdatedAt.IsZero() {
		host.DetailUpdatedAt = common_mysql.GetDefaultNonZeroTime()
	}
	if host.LabelUpdatedAt.IsZero() {
		host.LabelUpdatedAt = common_mysql.GetDefaultNonZeroTime()
	}
}

func (ds *Datastore) UpdateAndroidHost(ctx context.Context, host *fleet.AndroidHost, fromEnroll bool) error {
	if !host.IsValid() {
		return ctxerr.New(ctx, "valid Android host is required")
	}
	ds.setTimesToNonZero(host)

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update Android host get app config")
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `
		UPDATE hosts SET
			team_id = :team_id,
			detail_updated_at = :detail_updated_at,
			label_updated_at = :label_updated_at,
			hostname = :hostname,
			computer_name = :computer_name,
			platform = :platform,
			os_version = :os_version,
			build = :build,
			memory = :memory,
			hardware_serial = :hardware_serial,
			cpu_type = :cpu_type,
			hardware_model = :hardware_model,
			hardware_vendor = :hardware_vendor
		WHERE id = :id
		`
		_, err := sqlx.NamedExecContext(ctx, tx, stmt, map[string]interface{}{
			"id":                host.Host.ID,
			"team_id":           host.TeamID,
			"detail_updated_at": host.DetailUpdatedAt,
			"label_updated_at":  host.LabelUpdatedAt,
			"hostname":          host.Hostname,
			"computer_name":     host.ComputerName,
			"platform":          host.Platform,
			"os_version":        host.OSVersion,
			"build":             host.Build,
			"memory":            host.Memory,
			"hardware_serial":   host.HardwareSerial,
			"cpu_type":          host.CPUType,
			"hardware_model":    host.HardwareModel,
			"hardware_vendor":   host.HardwareVendor,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android host")
		}

		if fromEnroll {
			// update host_mdm to set enrolled back to true
			if err := upsertAndroidHostMDMInfoDB(ctx, tx, appCfg.ServerSettings.ServerURL, false, true, host.Host.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "update Android host MDM info")
			}
		}

		err = ds.Datastore.UpdateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android device")
		}
		return nil
	})
	return err
}

func (ds *Datastore) AndroidHostLite(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
	type liteHost struct {
		TeamID *uint `db:"team_id"`
		*android.Device
	}
	stmt := `SELECT
		h.team_id,
		ad.id,
		ad.host_id,
		ad.device_id,
		ad.enterprise_specific_id,
		ad.android_policy_id,
		ad.last_policy_sync_time
		FROM android_devices ad
		JOIN hosts h ON ad.host_id = h.id
		WHERE ad.enterprise_specific_id = ?`
	var host liteHost
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, enterpriseSpecificID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithName(enterpriseSpecificID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting device by enterprise specific ID")
	}
	result := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:     host.Device.HostID,
			TeamID: host.TeamID,
		},
		Device: host.Device,
	}
	result.SetNodeKey(enterpriseSpecificID)
	return result, nil
}

func (ds *Datastore) insertAndroidHostLabelMembershipTx(ctx context.Context, tx sqlx.ExtContext, hostID uint) error {
	// Insert the host in the builtin label memberships, adding them to the "All
	// Hosts" and "Android" labels.
	var labels []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	err := sqlx.SelectContext(ctx, tx, &labels, `SELECT id, name FROM labels WHERE label_type = 1 AND (name = ? OR name = ?)`,
		fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameAndroid)
	switch {
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get builtin labels")
	case len(labels) != 2:
		// Builtin labels can get deleted so it is important that we check that
		// they still exist before we continue.
		// Note that this is the same behavior as for the iOS/iPadOS host labels.
		level.Error(ds.logger).Log("err", fmt.Sprintf("expected 2 builtin labels but got %d", len(labels)))
		return nil
	}

	// We cannot assume IDs on labels, thus we look by name.
	var allHostsLabelID, androidLabelID uint
	for _, label := range labels {
		switch label.Name {
		case fleet.BuiltinLabelNameAllHosts:
			allHostsLabelID = label.ID
		case fleet.BuiltinLabelNameAndroid:
			androidLabelID = label.ID
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?)
		ON DUPLICATE KEY UPDATE host_id = host_id`,
		hostID, allHostsLabelID, hostID, androidLabelID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set label membership")
	}
	return nil
}

// BulkSetAndroidHostsUnenrolled sets all android hosts to unenrolled (for when
// Android MDM is turned off for all Fleet).
func (ds *Datastore) BulkSetAndroidHostsUnenrolled(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
UPDATE host_mdm
	SET server_url = '', mdm_id = NULL, enrolled = 0
	WHERE host_id IN (
		SELECT id FROM hosts WHERE platform = 'android'
	)`)
	return ctxerr.Wrap(ctx, err, "set host_mdm to unenrolled for android")
}

func upsertAndroidHostMDMInfoDB(ctx context.Context, tx sqlx.ExtContext, serverURL string, fromDEP, enrolled bool, hostIDs ...uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO mobile_device_management_solutions (name, server_url) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE server_url = VALUES(server_url)`,
		fleet.WellKnownMDMFleet, serverURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert mdm solution")
	}

	var mdmID int64
	if insertOnDuplicateDidInsertOrUpdate(result) {
		mdmID, _ = result.LastInsertId()
	} else {
		stmt := `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`
		if err := sqlx.GetContext(ctx, tx, &mdmID, stmt, fleet.WellKnownMDMFleet, serverURL); err != nil {
			return ctxerr.Wrap(ctx, err, "query mdm solution id")
		}
	}

	args := []interface{}{}
	parts := []string{}
	for _, id := range hostIDs {
		args = append(args, enrolled, serverURL, fromDEP, mdmID, false, id)
		parts = append(parts, "(?, ?, ?, ?, ?, ?)")
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, is_server, host_id) VALUES %s
		ON DUPLICATE KEY UPDATE enrolled = VALUES(enrolled), server_url = VALUES(server_url), mdm_id = VALUES(mdm_id)`, strings.Join(parts, ",")), args...)

	return ctxerr.Wrap(ctx, err, "upsert host mdm info")
}
