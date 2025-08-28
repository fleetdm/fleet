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
	"github.com/fleetdm/fleet/v4/server/mdm"
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

		host.Device, err = ds.CreateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}

		// insert storage data into host_disks table for API consumption
		if host.Host.GigsTotalDiskSpace > 0 || host.Host.GigsDiskSpaceAvailable > 0 {
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

		err = ds.UpdateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android device")
		}

		// update storage data in host_disks table for API consumption
		if host.Host.GigsTotalDiskSpace > 0 || host.Host.GigsDiskSpaceAvailable > 0 {
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
	stmt := `DELETE FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting android mdm config profile")
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMAndroidConfigProfile").WithName(profileUUID))
	}

	return nil
}

func (ds *Datastore) GetMDMAndroidProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	counts, err := getMDMAndroidStatusCountsDB(ctx, ds, teamID)
	if err != nil {
		return nil, err
	}

	var res fleet.MDMProfilesSummary
	for _, c := range counts {
		switch c.Status {
		case "failed":
			res.Failed = c.Count
		case "pending":
			res.Pending += c.Count
		case "verifying":
			res.Verifying = c.Count
		case "verified":
			res.Verified = c.Count
		case "":
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("counted %d android hosts on team %v with mdm turned on but no profiles", c.Count, teamID))
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unexpected mdm android status count: status=%s, count=%d", c.Status, c.Count))
		}
	}

	return &res, nil
}

func getMDMAndroidStatusCountsDB(ctx context.Context, ds *Datastore, teamID *uint) ([]statusCounts, error) {
	var args []interface{}
	subqueryFailed, subqueryFailedArgs, err := subqueryHostsMDMAndroidProfilesStatusFailed()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMAndroidProfilesStatusFailed")
	}
	args = append(args, subqueryFailedArgs...)
	subqueryPending, subqueryPendingArgs, err := subqueryHostsMDMAndroidProfilesStatusPending()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMAndroidProfilesStatusPending")
	}
	args = append(args, subqueryPendingArgs...)
	subqueryVerifying, subqueryVeryingingArgs, err := subqueryHostsMDMAndroidProfilesStatusVerifying()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMAndroidProfilesStatusVerifying")
	}
	args = append(args, subqueryVeryingingArgs...)
	subqueryVerified, subqueryVerifiedArgs, err := subqueryHostsMDMAndroidProfilesStatusVerified()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMAndroidProfilesStatusVerified")
	}
	args = append(args, subqueryVerifiedArgs...)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	stmt := fmt.Sprintf(`
SELECT
    CASE
        WHEN EXISTS (%s) THEN
            'failed'
        WHEN EXISTS (%s) THEN
            'pending'
        WHEN EXISTS (%s) THEN
            'verifying'
        WHEN EXISTS (%s) THEN
            'verified'
        ELSE
            ''
    END AS final_status,
    SUM(1) AS count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
WHERE
    h.platform = 'android' AND
    hmdm.enrolled = 1 AND
    %s
GROUP BY
    final_status`,
		subqueryFailed,
		subqueryPending,
		subqueryVerifying,
		subqueryVerified,
		teamFilter,
	)

	var counts []statusCounts
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func subqueryHostsMDMAndroidProfilesStatusFailed() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_android_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.status = ?
                AND hmap.profile_name NOT IN(?)`
	args := []interface{}{
		fleet.MDMDeliveryFailed,
		mdm.ListFleetReservedAndroidProfileNames(),
	}

	return sqlx.In(sql, args...)
}

func subqueryHostsMDMAndroidProfilesStatusPending() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_android_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND (hmap.status IS NULL OR hmap.status = ?)
				AND hmap.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_android_profiles hmap2
                    WHERE (h.uuid = hmap2.host_uuid
                        AND hmap2.status = ?
                        AND hmap2.profile_name NOT IN(?)))`
	args := []interface{}{
		fleet.MDMDeliveryPending,
		mdm.ListFleetReservedAndroidProfileNames(),
		fleet.MDMDeliveryFailed,
		mdm.ListFleetReservedAndroidProfileNames(),
	}
	return sqlx.In(sql, args...)
}

func subqueryHostsMDMAndroidProfilesStatusVerifying() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_android_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.operation_type = ?
                AND hmap.status = ?
                AND hmap.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_android_profiles hmap2
                    WHERE (h.uuid = hmap2.host_uuid
                        AND hmap2.operation_type = ?
                        AND hmap2.profile_name NOT IN(?)
                        AND(hmap2.status IS NULL
                            OR hmap2.status NOT IN(?))))`

	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerifying,
		mdm.ListFleetReservedAndroidProfileNames(),
		fleet.MDMOperationTypeInstall,
		mdm.ListFleetReservedAndroidProfileNames(),
		[]interface{}{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerified},
	}
	return sqlx.In(sql, args...)
}

func subqueryHostsMDMAndroidProfilesStatusVerified() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_android_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.operation_type = ?
                AND hmap.status = ?
                AND hmap.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_android_profiles hmap2
                    WHERE (h.uuid = hmap2.host_uuid
                        AND hmap2.operation_type = ?
                        AND hmap2.profile_name NOT IN(?)
                        AND(hmap2.status IS NULL
                            OR hmap2.status != ?)))`
	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerified,
		mdm.ListFleetReservedAndroidProfileNames(),
		fleet.MDMOperationTypeInstall,
		mdm.ListFleetReservedAndroidProfileNames(),
		fleet.MDMDeliveryVerified,
	}
	return sqlx.In(sql, args...)
}
