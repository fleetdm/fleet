package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
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
			label_updated_at,
			uuid
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
			:label_updated_at,
			:uuid
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
			label_updated_at = VALUES(label_updated_at),
			uuid = VALUES(uuid)
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
			"uuid":              host.UUID,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host")
		}
		id, _ := result.LastInsertId()
		if id == 0 {
			// This was an UPDATE, not an INSERT, so we need to get the host ID
			var hostID uint
			err := sqlx.GetContext(ctx, tx, &hostID, `SELECT id FROM hosts WHERE node_key = ?`, host.NodeKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get host ID after update")
			}
			host.Host.ID = hostID
		} else {
			host.Host.ID = uint(id) // nolint:gosec
		}
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

		host.Device, err = ds.CreateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}

		// insert storage data into host_disks table for API consumption
		// Check != 0 to allow -1 sentinel value for "not supported" to be stored
		if host.Host.GigsTotalDiskSpace != 0 || host.Host.GigsDiskSpaceAvailable != 0 {
			err = ds.SetOrUpdateHostDisksSpace(ctx, host.Host.ID,
				host.Host.GigsDiskSpaceAvailable,
				host.Host.PercentDiskSpaceAvailable,
				host.Host.GigsTotalDiskSpace)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "setting Android host disk space")
			}
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
	if host.PolicyUpdatedAt.IsZero() {
		host.PolicyUpdatedAt = common_mysql.GetDefaultNonZeroTime()
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
			hardware_vendor = :hardware_vendor,
			uuid = :uuid
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
			"uuid":              host.UUID,
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

		err = ds.UpdateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android device")
		}

		// update storage data in host_disks table for API consumption
		// Check != 0 to allow -1 sentinel value for "not supported" to be stored
		if host.Host.GigsTotalDiskSpace != 0 || host.Host.GigsDiskSpaceAvailable != 0 {
			err = ds.SetOrUpdateHostDisksSpace(ctx, host.Host.ID,
				host.Host.GigsDiskSpaceAvailable,
				host.Host.PercentDiskSpaceAvailable,
				host.Host.GigsTotalDiskSpace)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "updating Android host disk space")
			}
		}

		return nil
	})
	return err
}

func (ds *Datastore) AndroidHostLite(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
	type liteHost struct {
		TeamID *uint  `db:"team_id"`
		UUID   string `db:"uuid"`
		*android.Device
	}
	stmt := `SELECT
		h.team_id,
		h.uuid,
		ad.id,
		ad.host_id,
		ad.device_id,
		ad.enterprise_specific_id,
		ad.last_policy_sync_time,
		ad.applied_policy_id,
		ad.applied_policy_version
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
			UUID:   host.UUID,
		},
		Device: host.Device,
	}
	result.SetNodeKey(enterpriseSpecificID)
	return result, nil
}

func (ds *Datastore) AndroidHostLiteByHostUUID(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
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
		ad.last_policy_sync_time,
		ad.applied_policy_id,
		ad.applied_policy_version
	FROM android_devices ad
		JOIN hosts h ON ad.host_id = h.id
	WHERE h.uuid = ?`
	var host liteHost
	switch err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, hostUUID); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithName(hostUUID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting android device by host UUID")
	}
	result := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:     host.Device.HostID,
			UUID:   hostUUID,
			TeamID: host.TeamID,
		},
		Device: host.Device,
	}
	if host.Device.EnterpriseSpecificID != nil {
		result.SetNodeKey(*host.Device.EnterpriseSpecificID)
	}
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
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set host_mdm to unenrolled for android hosts in bulk")
	}
	// Delete all Android custom OS settings for unenrolled hosts.
	// We do this in one query using a JOIN to avoid doing it one host at a time.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_mdm_android_profiles`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete Android custom OS settings for unenrolled hosts in bulk")
	}
	return nil
}

// SetAndroidHostUnenrolled sets a single android host to unenrolled in host_mdm and OS settings records
// associated with it. If the host is not enrolled, it does nothing and returns false.
func (ds *Datastore) SetAndroidHostUnenrolled(ctx context.Context, hostID uint) (bool, error) {
	var rows int64
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		result, err := tx.ExecContext(ctx, `
UPDATE host_mdm
	SET server_url = '', mdm_id = NULL, enrolled = 0
	WHERE host_id = ? AND enrolled = 1`, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "set host_mdm to unenrolled for android host")
		}
		rows, err = result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for set host_mdm unenrolled for android host")
		}
		if rows > 0 {
			var uuid string
			err = sqlx.GetContext(ctx, tx, &uuid, `SELECT uuid FROM hosts WHERE id = ?`, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get host uuid")
			}
			err = ds.deleteMDMOSCustomSettingsForHost(ctx, tx, uuid, "android")
			return ctxerr.Wrap(ctx, err, "delete Android custom OS settings for unenrolled host")
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
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

	// Query host UUIDs to determine personal enrollment status
	// For Android, a non-empty UUID (enterprise_specific_id) indicates a BYOD/personal device
	type hostInfo struct {
		ID   uint   `db:"id"`
		UUID string `db:"uuid"`
	}
	var hosts []hostInfo
	query, args, err := sqlx.In(`SELECT id, uuid FROM hosts WHERE id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build host query")
	}
	if err := sqlx.SelectContext(ctx, tx, &hosts, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "query host UUIDs")
	}

	// Build map of host ID to personal enrollment status
	hostPersonalEnrollment := make(map[uint]bool)
	for _, h := range hosts {
		// Android BYOD devices have a non-empty UUID (enterprise_specific_id)
		hostPersonalEnrollment[h.ID] = h.UUID != ""
	}

	args = []interface{}{}
	parts := []string{}
	for _, id := range hostIDs {
		isPersonalEnrollment := hostPersonalEnrollment[id]
		args = append(args, enrolled, serverURL, fromDEP, mdmID, false, isPersonalEnrollment, id)
		parts = append(parts, "(?, ?, ?, ?, ?, ?, ?)")
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, is_server, is_personal_enrollment, host_id) VALUES %s
		ON DUPLICATE KEY UPDATE enrolled = VALUES(enrolled), server_url = VALUES(server_url), mdm_id = VALUES(mdm_id), is_personal_enrollment = VALUES(is_personal_enrollment)`, strings.Join(parts, ",")), args...)

	return ctxerr.Wrap(ctx, err, "upsert host mdm info")
}

func (ds *Datastore) NewMDMAndroidConfigProfile(ctx context.Context, cp fleet.MDMAndroidConfigProfile) (*fleet.MDMAndroidConfigProfile, error) {
	profileUUID := fleet.MDMAndroidProfileUUIDPrefix + uuid.New().String()
	insertProfileStmt := `
INSERT INTO
    mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json, uploaded_at)
(SELECT ?, ?, ?, ?, CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_windows_configuration_profiles WHERE name = ? AND team_id = ?
	)
)`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertProfileStmt, profileUUID, teamID, cp.Name, cp.RawJSON, cp.Name, teamID, cp.Name, teamID, cp.Name, teamID)
		if err != nil {
			switch {
			case IsDuplicate(err):
				return &existsError{
					ResourceType: "MDMAndroidConfigProfile.Name",
					Identifier:   cp.Name,
					TeamID:       cp.TeamID,
				}
			default:
				return ctxerr.Wrap(ctx, err, "creating new android mdm config profile")
			}
		}

		aff, _ := res.RowsAffected()
		if aff == 0 {
			return &existsError{
				ResourceType: "MDMAndroidConfigProfile.Name",
				Identifier:   cp.Name,
				TeamID:       cp.TeamID,
			}
		}

		labels := make([]fleet.ConfigurationProfileLabel, 0, len(cp.LabelsIncludeAll)+len(cp.LabelsIncludeAny)+len(cp.LabelsExcludeAny))
		for i := range cp.LabelsIncludeAll {
			cp.LabelsIncludeAll[i].ProfileUUID = profileUUID
			cp.LabelsIncludeAll[i].RequireAll = true
			cp.LabelsIncludeAll[i].Exclude = false
			labels = append(labels, cp.LabelsIncludeAll[i])
		}
		for i := range cp.LabelsIncludeAny {
			cp.LabelsIncludeAny[i].ProfileUUID = profileUUID
			cp.LabelsIncludeAny[i].RequireAll = false
			cp.LabelsIncludeAny[i].Exclude = false
			labels = append(labels, cp.LabelsIncludeAny[i])
		}
		for i := range cp.LabelsExcludeAny {
			cp.LabelsExcludeAny[i].ProfileUUID = profileUUID
			cp.LabelsExcludeAny[i].RequireAll = false
			cp.LabelsExcludeAny[i].Exclude = true
			labels = append(labels, cp.LabelsExcludeAny[i])
		}
		var profsWithoutLabel []string
		if len(labels) == 0 {
			profsWithoutLabel = append(profsWithoutLabel, profileUUID)
		}
		if _, err := batchSetProfileLabelAssociationsDB(ctx, tx, labels, profsWithoutLabel, "android"); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting android profile label associations")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &fleet.MDMAndroidConfigProfile{
		ProfileUUID: profileUUID,
		Name:        cp.Name,
		RawJSON:     cp.RawJSON,
		TeamID:      cp.TeamID,
	}, nil
}

func (ds *Datastore) GetMDMAndroidConfigProfile(ctx context.Context, profileUUID string) (*fleet.MDMAndroidConfigProfile, error) {
	stmt := `SELECT profile_uuid, team_id, name, raw_json, auto_increment, created_at, uploaded_at FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`
	var profile fleet.MDMAndroidConfigProfile
	err := sqlx.GetContext(ctx, ds.reader(ctx), &profile, stmt, profileUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("MDMAndroidConfigProfile").WithName(profileUUID)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting android mdm config profile")
	}
	labels, err := ds.listProfileLabelsForProfiles(ctx, nil, nil, []string{profile.ProfileUUID}, nil)
	if err != nil {
		return nil, err
	}
	for _, lbl := range labels {
		switch {
		case lbl.Exclude && lbl.RequireAll:
			// this should never happen so log it for debugging
			level.Warn(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all. Label will be ignored.",
				"profile_uuid", lbl.ProfileUUID,
				"label_name", lbl.LabelName,
			)
		case lbl.Exclude && !lbl.RequireAll:
			profile.LabelsExcludeAny = append(profile.LabelsExcludeAny, lbl)
		case !lbl.Exclude && !lbl.RequireAll:
			profile.LabelsIncludeAny = append(profile.LabelsIncludeAny, lbl)
		default:
			// default include all
			profile.LabelsIncludeAll = append(profile.LabelsIncludeAll, lbl)
		}
	}
	return &profile, nil
}

func (ds *Datastore) DeleteMDMAndroidConfigProfile(ctx context.Context, profileUUID string) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`
		res, err := tx.ExecContext(ctx, stmt, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting android mdm config profile")
		}

		deleted, _ := res.RowsAffected()
		if deleted != 1 {
			return ctxerr.Wrap(ctx, notFound("MDMAndroidConfigProfile").WithName(profileUUID))
		}

		if err := cancelAndroidHostInstallsForDeletedMDMProfiles(ctx, tx, []string{profileUUID}); err != nil {
			return err
		}
		return nil
	})
}

func (ds *Datastore) GetMDMAndroidProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	stmt := `
SELECT
	COUNT(id) AS count,
	%s AS status
FROM
	hosts h
	INNER JOIN host_mdm hmdm ON h.id=hmdm.host_id
	%s
WHERE
	platform = 'android' AND
	hmdm.enrolled = 1 AND
	 %s
GROUP BY
	status HAVING status IS NOT NULL`

	teamFilter := "team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = fmt.Sprintf("team_id = %d", *teamID)
	}

	stmt = fmt.Sprintf(stmt, sqlCaseMDMAndroidStatus(), sqlJoinMDMAndroidProfilesStatus(), teamFilter)

	var dest []struct {
		Count  uint   `db:"count"`
		Status string `db:"status"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt); err != nil {
		return nil, err
	}

	byStatus := make(map[string]uint)
	for _, s := range dest {
		if _, ok := byStatus[s.Status]; ok {
			return nil, fmt.Errorf("duplicate status %s from android profiles summary", s.Status)
		}
		byStatus[s.Status] = s.Count
	}

	var res fleet.MDMProfilesSummary
	for s, c := range byStatus {
		switch fleet.MDMDeliveryStatus(s) {
		case fleet.MDMDeliveryFailed:
			res.Failed = c
		case fleet.MDMDeliveryPending:
			res.Pending = c
		case fleet.MDMDeliveryVerifying:
			res.Verifying = c
		case fleet.MDMDeliveryVerified:
			res.Verified = c
		default:
			return nil, fmt.Errorf("unknown status %s", s)
		}
	}

	return &res, nil
}

// sqlJoinMDMAndroidProfilesStatus returns a SQL snippet that can be used to join a table derived from
// host_mdm_android_profiles (grouped by host_uuid and status) and the hosts table. For each host_uuid,
// it derives a boolean value for each status category. The value will be 1 if the host has any
// profile in the given status category. The snippet assumes the hosts table to be aliased as 'h'.
func sqlJoinMDMAndroidProfilesStatus() string {
	// NOTE: To make this snippet reusable, we're not using sqlx.Named here because it would
	// complicate usage in other queries (e.g., list hosts).
	var (
		failed    = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryFailed))
		pending   = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryPending))
		verifying = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerifying))
		verified  = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerified))
		install   = fmt.Sprintf("'%s'", string(fleet.MDMOperationTypeInstall))
	)
	return `
	LEFT JOIN (
		-- profile statuses grouped by host uuid, boolean value will be 1 if host has any profile with the given status
		SELECT
			host_uuid,
			MAX( IF(status IS NULL OR status = ` + pending + `, 1, 0)) AS android_prof_pending,
			MAX( IF(status = ` + failed + `, 1, 0)) AS android_prof_failed,
			MAX( IF(status = ` + verifying + ` AND operation_type = ` + install + `, 1, 0)) AS android_prof_verifying,
			MAX( IF(status = ` + verified + ` AND operation_type = ` + install + `, 1, 0)) AS android_prof_verified
		FROM
			host_mdm_android_profiles
		GROUP BY
			host_uuid) hmgp ON h.uuid = hmgp.host_uuid
`
}

// sqlCaseMDMAndroidStatus returns a SQL snippet that can be used to determine the status of an Android host
// based on the status of its profiles. It should be used in conjunction with sqlJoinMDMAndroidProfilesStatus
// It assumes the hosts table to be aliased as 'h'
func sqlCaseMDMAndroidStatus() string {
	// NOTE: To make this snippet reusable, we're not using sqlx.Named here because it would
	// complicate usage in other queries (e.g., list hosts).
	var (
		failed    = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryFailed))
		pending   = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryPending))
		verifying = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerifying))
		verified  = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerified))
	)
	return `
	CASE WHEN (android_prof_failed) THEN
		` + failed + `
	WHEN (android_prof_pending) THEN
		` + pending + `
	WHEN (android_prof_verifying) THEN
		` + verifying + `
	WHEN (android_prof_verified) THEN
		` + verified + `
	END
`
}

func (ds *Datastore) NewAndroidPolicyRequest(ctx context.Context, req *fleet.MDMAndroidPolicyRequest) error {
	const stmt = `
	INSERT INTO android_policy_requests
		(request_uuid, request_name, policy_id, payload, status_code, error_details, applied_policy_version, policy_version)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?)
`
	if req.RequestUUID == "" {
		req.RequestUUID = uuid.NewString()
	}

	_, err := ds.writer(ctx).ExecContext(ctx, stmt,
		req.RequestUUID,
		req.RequestName,
		req.PolicyID,
		req.Payload,
		req.StatusCode,
		req.ErrorDetails,
		req.AppliedPolicyVersion,
		req.PolicyVersion,
	)
	return ctxerr.Wrap(ctx, err, "inserting android policy request")
}

func (ds *Datastore) GetAndroidPolicyRequestByUUID(ctx context.Context, requestUUID string) (*fleet.MDMAndroidPolicyRequest, error) {
	const stmt = `
		SELECT
			request_uuid,
			request_name,
			policy_id,
			payload,
			status_code,
			error_details,
			applied_policy_version,
			policy_version
		FROM
			android_policy_requests
		WHERE
			request_uuid = ?
	`

	req := fleet.MDMAndroidPolicyRequest{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &req, stmt, requestUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, common_mysql.NotFound("AndroidPolicyRequest").WithName(requestUUID)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting android policy request")
	}

	return &req, nil
}

const androidApplicableProfilesQuery = `
	-- non label-based profiles
	SELECT
		macp.profile_uuid,
		macp.name,
		h.uuid as host_uuid,
		h.id as host_id,
		0 as count_profile_labels,
		0 as count_non_broken_labels,
		0 as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_android_configuration_profiles macp
			JOIN hosts h
				ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
			JOIN android_devices ad
				ON ad.host_id = h.id
	WHERE
		h.platform = 'android' AND
		NOT EXISTS (
			SELECT 1
			FROM mdm_configuration_profile_labels mcpl
			WHERE mcpl.android_profile_uuid = macp.profile_uuid
		) AND
		( %s )

	UNION

	-- label-based profiles where the host is a member of all the labels (include-all).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		macp.profile_uuid,
		macp.name,
		h.uuid as host_uuid,
		h.id as host_id,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_android_configuration_profiles macp
			JOIN hosts h
				ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
			JOIN android_devices ad
				ON ad.host_id = h.id
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.android_profile_uuid = macp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 1
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'android' AND
		( %s )
	GROUP BY
		macp.profile_uuid, macp.name, h.uuid, h.id
	HAVING
		count_profile_labels > 0 AND count_host_labels = count_profile_labels

	UNION

	-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
	-- explicitly ignore profiles with broken excluded labels so that they are never applied,
	-- and ignore profiles that depend on labels created _after_ the label_updated_at timestamp
	-- of the host (because we don't have results for that label yet, the host may or may not be
	-- a member).
	SELECT
		macp.profile_uuid,
		macp.name,
		h.uuid as host_uuid,
		h.id as host_id,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		-- this helps avoid the case where the host is not a member of a label
		-- just because it hasn't reported results for that label yet.
		SUM(CASE WHEN lbl.created_at IS NOT NULL AND h.label_updated_at >= lbl.created_at THEN 1 ELSE 0 END) as count_host_updated_after_labels
	FROM
		mdm_android_configuration_profiles macp
			JOIN hosts h
				ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
			JOIN android_devices ad
				ON ad.host_id = h.id
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.android_profile_uuid = macp.profile_uuid AND mcpl.exclude = 1 AND mcpl.require_all = 0
			LEFT OUTER JOIN labels lbl
				ON lbl.id = mcpl.label_id
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'android' AND
		( %s )
	GROUP BY
		macp.profile_uuid, macp.name, h.uuid, h.id
	HAVING
		-- considers only the profiles with labels, without any broken label, with results reported after all labels were
		-- created and with the host not in any label
		count_profile_labels > 0 AND count_profile_labels = count_non_broken_labels AND
		count_profile_labels = count_host_updated_after_labels AND count_host_labels = 0

	UNION

	-- label-based profiles where the host is a member of any of the labels (include-any).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		macp.profile_uuid,
		macp.name,
		h.uuid as host_uuid,
		h.id as host_id,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_android_configuration_profiles macp
			JOIN hosts h
				ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
			JOIN android_devices ad
				ON ad.host_id = h.id
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.android_profile_uuid = macp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 0
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'android' AND
		( %s )
	GROUP BY
		macp.profile_uuid, macp.name, h.uuid, h.id
	HAVING
		count_profile_labels > 0 AND count_host_labels >= 1
`

// ListMDMAndroidProfilesToSend is the android platform equivalent to
// ListMDMAppleProfilesToInstall/Remove and
// ListMDMWindowsProfilesToInstall/Remove. It plays a similar role but is quite
// different in implementation since Android profiles are fundamentally
// different from those platforms - the "configuration profiles" are just
// fragments of the full (and unique per host) "Android policy" to apply to a
// host, so as soon as there is a change to the set of profiles to apply to a
// host, the full list of applicable profiles are needed to generate the
// resulting policy.
//
// That is, profiles are not applied individually, but as a merged set. For
// that reason, there is no "to install" and "to remove", as removing a profile
// is just not including it in the merged set of profiles to apply.
//
// So with that in mind, what this method does is return the full set of
// applicable profiles for each host that has a change in that set of
// applicable profiles, so that it needs to be sent again, along with the list
// of of previously-applied profiles that need to be removed (which is just
// "not merging them" in the Android policy, but until this policy is fully
// applied on the host we need to mark those profiles as pending removal).
//
// See https://github.com/fleetdm/fleet/issues/32032#issuecomment-3229548389
// for more details on the rationale of that approach.
func (ds *Datastore) ListMDMAndroidProfilesToSend(ctx context.Context) ([]*fleet.MDMAndroidProfilePayload, []*fleet.MDMAndroidProfilePayload, error) {
	var toApplyProfiles, toRemoveProfiles []*fleet.MDMAndroidProfilePayload
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		hostsWithChangesStmt := fmt.Sprintf(`
	WITH ds AS ( %s )

	SELECT
		DISTINCT ds.host_uuid
	FROM ds
		INNER JOIN android_devices ad
			ON ad.host_id = ds.host_id
		INNER JOIN host_mdm hmdm
			ON ad.host_id=hmdm.host_id
		LEFT OUTER JOIN host_mdm_android_profiles hmap
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
	WHERE
	  -- host is enrolled
	    hmdm.enrolled = 1 AND
		(
		-- at least one profile is missing from host_mdm_android_profiles
			hmap.host_uuid IS NULL OR
			-- profile was never sent or was updated after sent
			-- TODO(ap): need to make sure we set it to NULL when profile is updated
			( hmap.included_in_policy_version IS NULL AND COALESCE(hmap.status, '') <> ? ) OR
			hmap.status IS NULL OR
			-- profile was sent in older policy version than currently applied
			(hmap.included_in_policy_version IS NOT NULL AND ad.applied_policy_id = ds.host_uuid AND
				hmap.included_in_policy_version < COALESCE(ad.applied_policy_version, 0))
		)

	UNION

	SELECT
		DISTINCT hmap.host_uuid
	FROM host_mdm_android_profiles hmap
		INNER JOIN hosts h
			ON h.uuid = hmap.host_uuid
		INNER JOIN android_devices ad
			ON ad.host_id = h.id
		INNER JOIN host_mdm hmdm
			ON hmdm.host_id = h.id
		LEFT OUTER JOIN ds
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
	WHERE
	  -- at least one profile was removed from the set of applicable profiles
	    hmdm.enrolled = 1 AND
		ds.host_uuid IS NULL AND
		-- and it is not in pending remove status (in which case it was processed)
		( hmap.operation_type != ? OR COALESCE(hmap.status, '') <> ? )
`, fmt.Sprintf(androidApplicableProfilesQuery, "TRUE", "TRUE", "TRUE", "TRUE"))

		// NOTE: we explicitly don't "ignore" profiles to remove based on broken labels,
		// because of how Android profiles are applied vs other platforms (ignoring
		// a broken profile would effectively remove it anyway, and including it so
		// we don't remove it could cause errors applying the rest of the policy if
		// the setting is invalid, which is worse and contrary to the "broken profiles
		// are ignored" general behavior).
		//
		// So unlike for Apple/Windows, for Android we effectively remove broken
		// profiles from the host.
		//
		// see https://github.com/fleetdm/fleet/issues/25557#issuecomment-3246496873

		var hostUUIDs []string
		if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, hostsWithChangesStmt,
			fleet.MDMDeliveryFailed, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending); err != nil {
			return ctxerr.Wrap(ctx, err, "list android hosts with profile changes")
		}

		if len(hostUUIDs) == 0 {
			return nil
		}

		// retrieve all the applicable profiles for those hosts
		listToInstallProfilesStmt := fmt.Sprintf(`
	SELECT
		ds.profile_uuid,
		ds.name as profile_name,
		ds.host_uuid,
		COALESCE(hmap.request_fail_count, 0) as request_fail_count
	FROM ( %s ) ds
		LEFT OUTER JOIN host_mdm_android_profiles hmap
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
`, fmt.Sprintf(androidApplicableProfilesQuery, "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)"))

		query, args, err := sqlx.In(listToInstallProfilesStmt, hostUUIDs, hostUUIDs, hostUUIDs, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building list android host applicable profiles query")
		}
		if err := sqlx.SelectContext(ctx, tx, &toApplyProfiles, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "list android host applicable profiles")
		}

		listToRemoveProfilesStmt := fmt.Sprintf(`
	SELECT
		hmap.profile_uuid,
		hmap.profile_name,
		hmap.host_uuid,
		hmap.request_fail_count
	FROM ( %s ) ds
		RIGHT OUTER JOIN host_mdm_android_profiles hmap
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
	WHERE
		hmap.host_uuid IN (?) AND
		ds.host_uuid IS NULL
`, fmt.Sprintf(androidApplicableProfilesQuery, "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)"))

		query, args, err = sqlx.In(listToRemoveProfilesStmt, hostUUIDs, hostUUIDs, hostUUIDs, hostUUIDs, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building list android host to remove profiles query")
		}
		if err := sqlx.SelectContext(ctx, tx, &toRemoveProfiles, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "list android host to remove profiles")
		}

		return nil
	})
	return toApplyProfiles, toRemoveProfiles, err
}

func (ds *Datastore) GetMDMAndroidProfilesContents(ctx context.Context, uuids []string) (map[string]json.RawMessage, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt := `
		SELECT profile_uuid, raw_json
		FROM mdm_android_configuration_profiles
		WHERE profile_uuid IN (?)
	`
	query, args, err := sqlx.In(stmt, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building get android profiles contents query")
	}

	rows, err := ds.reader(ctx).QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "querying android profiles contents")
	}
	defer rows.Close()

	results := make(map[string]json.RawMessage, len(uuids))
	for rows.Next() {
		var (
			uid     string
			rawJSON json.RawMessage
		)
		if err := rows.Scan(&uid, &rawJSON); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning android profile content")
		}
		results[uid] = rawJSON
	}

	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterating android profiles contents")
	}
	return results, nil
}

func (ds *Datastore) BulkUpsertMDMAndroidHostProfiles(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload) error {
	if len(payload) == 0 {
		return nil
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`
			INSERT INTO host_mdm_android_profiles (
				host_uuid,
				status,
				operation_type,
				detail,
				profile_uuid,
				profile_name,
				policy_request_uuid,
				device_request_uuid,
				request_fail_count,
				included_in_policy_version
			)
			VALUES %s
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type),
				detail = VALUES(detail),
				profile_name = VALUES(profile_name),
				policy_request_uuid = VALUES(policy_request_uuid),
				device_request_uuid = VALUES(device_request_uuid),
				request_fail_count = VALUES(request_fail_count),
				included_in_policy_version = VALUES(included_in_policy_version)
`, strings.TrimSuffix(valuePart, ","),
		)

		// Taken from BulkUpsertMDMAppleHostProfiles: We need to run with retry
		// due to deadlocks. The INSERT/ON DUPLICATE KEY UPDATE pattern is prone
		// to deadlocks when multiple threads are modifying nearby rows. That's
		// because this statement uses gap locks. When two transactions acquire
		// the same gap lock, they may deadlock. Two simultaneous transactions
		// may happen when cron job runs and the user is updating via the UI at
		// the same time.
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
		return err
	}

	generateValueArgs := func(p *fleet.MDMAndroidProfilePayload) (string, []any) {
		valuePart := "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?),"
		args := []any{
			p.HostUUID, p.Status, p.OperationType,
			p.Detail, p.ProfileUUID, p.ProfileName,
			p.PolicyRequestUUID, p.DeviceRequestUUID, p.RequestFailCount,
			p.IncludedInPolicyVersion,
		}
		return valuePart, args
	}

	const defaultBatchSize = 1000 // number of parameters is this times number of placeholders
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	if err := batchProcessDB(payload, batchSize, generateValueArgs, executeUpsertBatch); err != nil {
		return err
	}

	return nil
}

func (ds *Datastore) GetHostMDMAndroidProfiles(ctx context.Context, hostUUID string) ([]fleet.HostMDMAndroidProfile, error) {
	// TODO(AP): confirm whether we should be hiding any profile names for Android like we do
	// for other platforms
	stmt := fmt.Sprintf(`
SELECT
	profile_uuid,
	profile_name AS name,
	-- internally, a NULL status implies that the cron needs to pick up
	-- this profile, for the user that difference doesn't exist, the
	-- profile is effectively pending. This is consistent with all our
	-- aggregation functions.
	COALESCE(status, '%s') AS status,
	COALESCE(operation_type, '') AS operation_type,
	COALESCE(detail, '') AS detail
FROM
	host_mdm_android_profiles
WHERE
host_uuid = ? AND NOT (operation_type = '%s' AND COALESCE(status, '%s') IN('%s', '%s'))`,
		fleet.MDMDeliveryPending,
		fleet.MDMOperationTypeRemove,
		fleet.MDMDeliveryPending,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
	)
	args := []interface{}{hostUUID}

	var profiles []fleet.HostMDMAndroidProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, args...); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (ds *Datastore) ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx context.Context, hostUUID string, policyVersion int64) ([]*fleet.MDMAndroidProfilePayload, error) {
	const stmt = `
		SELECT profile_uuid, host_uuid, status, operation_type, detail, profile_name, policy_request_uuid, device_request_uuid, request_fail_count, included_in_policy_version
		FROM host_mdm_android_profiles
		WHERE host_uuid = ? AND included_in_policy_version <= ? AND status = ? AND operation_type = ?
	`

	var profiles []*fleet.MDMAndroidProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, hostUUID, policyVersion, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing host MDM Android profiles pending install")
	}

	return profiles, nil
}

func (ds *Datastore) BulkDeleteMDMAndroidHostProfiles(ctx context.Context, hostUUID string, policyVersionId int64) error {
	stmt := `
		DELETE FROM host_mdm_android_profiles
		WHERE host_uuid = ? AND included_in_policy_version <= ? AND operation_type = ? AND status IN (?)
	`

	stmt, args, err := sqlx.In(stmt, hostUUID, policyVersionId, fleet.MDMOperationTypeRemove, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryPending, fleet.MDMDeliveryFailed})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to delete host MDM Android profiles")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host MDM Android profiles")
	}
	return nil
}

func (ds *Datastore) deleteAllAndroidProfiles(ctx context.Context, tx sqlx.ExtContext, tmID *uint) (int, error) {
	var teamID uint
	if tmID != nil {
		teamID = *tmID
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM mdm_android_configuration_profiles WHERE team_id = ?`, teamID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "deleting all android profiles for team")
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "getting rows affected when deleting all android profiles for team")
	}
	return int(rows), nil
}

func (ds *Datastore) batchSetMDMAndroidProfiles(
	ctx context.Context,
	tx sqlx.ExtContext,
	tmID *uint,
	profiles []*fleet.MDMAndroidConfigProfile,
) (updatedDB bool, err error) {
	if len(profiles) == 0 {
		rowsAffected, err := ds.deleteAllAndroidProfiles(ctx, tx, tmID)
		if err != nil {
			return false, err
		}

		return rowsAffected > 0, nil
	}

	// Select and delete profiles that are not incoming so we can cancel the install.
	const loadToBeDeletedProfilesNotInList = `
SELECT
  profile_uuid
FROM
  mdm_android_configuration_profiles
WHERE
  team_id = ? AND
  name NOT IN (?)
`

	// use a profile team id of 0 if no-team
	var profileTeamID uint
	if tmID != nil {
		profileTeamID = *tmID
	}

	// Create list of names from profiles
	incomingNames := make([]string, len(profiles))
	incomingUUIDS := make([]string, len(profiles))
	for i, p := range profiles {
		incomingNames[i] = p.Name
		incomingUUIDS[i] = p.ProfileUUID
	}

	var (
		stmt                string
		args                []interface{}
		deletedProfileUUIDs []string
	)
	stmt, args, err = sqlx.In(loadToBeDeletedProfilesNotInList, profileTeamID, incomingNames)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "build query to load profiles to be deleted")
	}
	if err := sqlx.SelectContext(ctx, tx, &deletedProfileUUIDs, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "load profiles to be deleted")
	}

	// Delete profiles that are not incoming
	const deleteProfilesNotInList = `
DELETE FROM
  mdm_android_configuration_profiles
WHERE
  profile_uuid IN (?)
`
	if len(deletedProfileUUIDs) > 0 {
		var result sql.Result
		stmt, args, err = sqlx.In(deleteProfilesNotInList, deletedProfileUUIDs)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "build query to delete profiles")
		}
		if result, err = tx.ExecContext(ctx, stmt, args...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "delete profiles")
		}

		if result != nil {
			rows, _ := result.RowsAffected()
			updatedDB = rows > 0
		}

		if err := cancelAndroidHostInstallsForDeletedMDMProfiles(ctx, tx, deletedProfileUUIDs); err != nil {
			return false, ctxerr.Wrap(ctx, err, "cancel android host installs for deleted profiles")
		}
	}

	// Insert or update incoming profiles
	const insertNewOrEditedProfile = `
	INSERT INTO mdm_android_configuration_profiles (
		profile_uuid,
		team_id,
		name,
		raw_json,
		uploaded_at
	) VALUES (CONCAT('` + fleet.MDMAndroidProfileUUIDPrefix + `', CONVERT(uuid() USING utf8mb4)), ?, ?, ?, CURRENT_TIMESTAMP(6))
	ON DUPLICATE KEY UPDATE
		raw_json = VALUES(raw_json),
		name = VALUES(name),
		uploaded_at = IF(raw_json = VALUES(raw_json) AND name = VALUES(name), uploaded_at, CURRENT_TIMESTAMP(6))
`
	for _, p := range profiles {
		var res sql.Result
		if res, err = tx.ExecContext(ctx, insertNewOrEditedProfile, profileTeamID, p.Name, p.RawJSON); err != nil {
			return false, ctxerr.Wrap(ctx, err, "insert or update profile")
		}

		if insertOnDuplicateDidInsertOrUpdate(res) {
			updatedDB = true
		}
	}

	const updateIncludedInPolicyVersionStmt = `
	UPDATE
		host_mdm_android_profiles
	SET
		included_in_policy_version = NULL,
		detail = NULL,
		policy_request_uuid = NULL,
		device_request_uuid = NULL,
		status = NULL,
		request_fail_count = 0
	WHERE
		profile_uuid IN (SELECT profile_uuid FROM mdm_android_configuration_profiles WHERE name IN (?))
	`
	stmt, args, err = sqlx.In(updateIncludedInPolicyVersionStmt, incomingNames)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "build query to update included in policy version")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "update included in policy version")
	}

	var mappedIncomingProfiles []*BatchSetAssociationIncomingProfile
	for _, p := range profiles {
		mappedIncomingProfiles = append(mappedIncomingProfiles, &BatchSetAssociationIncomingProfile{
			Name:             p.Name,
			ProfileUUID:      p.ProfileUUID,
			LabelsIncludeAll: p.LabelsIncludeAll,
			LabelsIncludeAny: p.LabelsIncludeAny,
			LabelsExcludeAny: p.LabelsExcludeAny,
		})
	}

	didUpdateLabels, err := ds.batchSetLabelAndVariableAssociations(ctx, tx, "android", tmID, mappedIncomingProfiles, nil)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "setting labels and variable associations")
	}

	return updatedDB || didUpdateLabels, nil
}

func cancelAndroidHostInstallsForDeletedMDMProfiles(ctx context.Context, tx sqlx.ExtContext, profileUUIDs []string) error {
	// For Android profiles, we can safely delete the rows where STATUS is null and operation is install.
	// For any other profiles, we update the operating to remove and set the STATUS to null
	// to let it be picked up by the reconciler.

	if len(profileUUIDs) == 0 {
		return nil
	}

	const delStmt = `
	DELETE FROM
		host_mdm_android_profiles
	WHERE
		profile_uuid IN (?) AND
		status IS NULL AND
		operation_type = ?`

	stmt, args, err := sqlx.In(delStmt, profileUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query to cancel android host installs for deleted profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "cancel android host installs for deleted profiles")
	}

	const updateStmt = `
	UPDATE
		host_mdm_android_profiles
	SET
		status = NULL,
		operation_type = ?
	WHERE
		profile_uuid IN (?) AND
		status IS NOT NULL AND
		operation_type = ?`

	stmt, args, err = sqlx.In(updateStmt, fleet.MDMOperationTypeRemove, profileUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query to update android host installs for deleted profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "update android host installs for deleted profiles")
	}

	return nil
}

// For android we set the status to NIL
func (ds *Datastore) bulkSetPendingMDMAndroidHostProfilesDB(
	ctx context.Context,
	hostUUIDs []string,
) (updatedDB bool, err error) {
	if len(hostUUIDs) == 0 {
		return false, nil
	}

	profilesToInstall, profilesToRemove, err := ds.ListMDMAndroidProfilesToSend(ctx)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "list android profiles to send")
	}

	if len(profilesToInstall) == 0 && len(profilesToRemove) == 0 {
		return false, nil
	}

	var profilesToUpsert []*fleet.MDMAndroidProfilePayload
	for setIndex, profiles := range [][]*fleet.MDMAndroidProfilePayload{profilesToInstall, profilesToRemove} {
		operationType := fleet.MDMOperationTypeInstall
		if setIndex == 1 {
			operationType = fleet.MDMOperationTypeRemove
		}

		for _, p := range profiles {
			profilesToUpsert = append(profilesToUpsert, &fleet.MDMAndroidProfilePayload{
				ProfileUUID:             p.ProfileUUID,
				ProfileName:             p.ProfileName,
				HostUUID:                p.HostUUID,
				OperationType:           operationType,
				Status:                  nil,
				Detail:                  "",
				PolicyRequestUUID:       nil,
				DeviceRequestUUID:       nil,
				RequestFailCount:        0,
				IncludedInPolicyVersion: nil,
			})
		}
	}

	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, profilesToUpsert)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "bulk upsert android host profiles")
	}

	return true, nil
}

// ListAndroidEnrolledDevicesForReconcile returns Android devices currently marked as enrolled in Fleet.
func (ds *Datastore) ListAndroidEnrolledDevicesForReconcile(ctx context.Context) ([]*android.Device, error) {
	var devices []*android.Device
	stmt := `SELECT
		ad.id,
		ad.host_id,
		ad.device_id,
		ad.enterprise_specific_id,
		ad.last_policy_sync_time,
		ad.applied_policy_id,
		ad.applied_policy_version
	FROM android_devices ad
	JOIN host_mdm hm ON hm.host_id = ad.host_id AND hm.enrolled = 1
	JOIN hosts h ON h.id = ad.host_id AND h.platform = 'android'`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &devices, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list enrolled android devices for reconcile")
	}
	return devices, nil
}
