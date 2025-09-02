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
		},
		Device: host.Device,
	}
	result.SetNodeKey(enterpriseSpecificID)
	return result, nil
}

func (ds *Datastore) AndroidHostLiteByHostID(ctx context.Context, hostID uint) (*fleet.AndroidHost, error) {
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
		WHERE ad.host_id = ?`
	var host liteHost
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, hostID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithID(hostID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting android device by host ID")
	}
	result := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:     host.Device.HostID,
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
		macp.profile_uuid, macp.name, h.uuid
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
		macp.profile_uuid, macp.name, h.uuid
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
		macp.profile_uuid, macp.name, h.uuid
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
// applicable profiles, so that it needs to be sent again.
//
// See https://github.com/fleetdm/fleet/issues/32032#issuecomment-3229548389
// for more details on the rationale of that approach.
func (ds *Datastore) ListMDMAndroidProfilesToSend(ctx context.Context) ([]*fleet.MDMAndroidProfilePayload, error) {
	var result []*fleet.MDMAndroidProfilePayload
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		hostsWithChangesStmt := fmt.Sprintf(`
	WITH (%s) AS ds

	SELECT
		DISTINCT ds.host_uuid
	FROM ds
		INNER JOIN android_devices ad
			ON ad.host_id = ds.host_id
		LEFT OUTER JOIN host_mdm_android_profiles hmap
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
	WHERE
	  -- at least one profile is missing from host_mdm_android_profiles
		hmap.host_uuid IS NULL OR
		-- profile was never sent or was updated after sent
		-- TODO(ap): need to make sure we set it to NULL when profile is updated
		hmap.included_in_policy_version IS NULL OR
		-- profile was sent in older policy version than currently applied
		(hmap.included_in_policy_version IS NOT NULL AND ad.applied_policy_id = ds.host_uuid AND
			hmap.included_in_policy_version < COALESCE(ad.applied_policy_version, 0))

	UNION

	SELECT
		DISTINCT hmap.host_uuid
	FROM host_mdm_android_profiles hmap
		INNER JOIN android_devices ad
			ON ad.host_id = ds.host_id
		LEFT OUTER JOIN ds
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
	WHERE
	  -- at least one profile was removed from the set of applicable profiles
		ds.host_uuid IS NULL
`, fmt.Sprintf(androidApplicableProfilesQuery, "TRUE", "TRUE", "TRUE", "TRUE"))

		// NOTE: we explicitly don't ignore profiles to remove based on broken labels,
		// because of how Android profiles are applied vs other platforms (ignoring
		// a broken profile would effectively remove it anyway, and including it so
		// we don't remove it could cause errors applying the rest of the policy if
		// the setting is invalid, which is worse and contrary to the "broken profiles
		// are ignored" general behavior).
		// see https://github.com/fleetdm/fleet/issues/25557#issuecomment-3246496873

		var hostUUIDs []string
		if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, hostsWithChangesStmt); err != nil {
			return ctxerr.Wrap(ctx, err, "list android hosts with profile changes")
		}

		if len(hostUUIDs) == 0 {
			return nil
		}

		// retrieve all the applicable profiles for those hosts
		listHostProfilesStmt := fmt.Sprintf(`
	SELECT
		ds.profile_uuid,
		ds.name as profile_name,
		ds.host_uuid,
		COALESCE(hmap.request_fail_count, 0) as request_fail_count
	FROM ( %s ) AS ds
		LEFT OUTER JOIN host_mdm_android_profiles hmap
			ON hmap.host_uuid = ds.host_uuid AND hmap.profile_uuid = ds.profile_uuid
)`, fmt.Sprintf(androidApplicableProfilesQuery, "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)", "h.uuid IN (?)"))

		query, args, err := sqlx.In(listHostProfilesStmt, hostUUIDs, hostUUIDs, hostUUIDs, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building list android host applicable profiles query")
		}
		if err := sqlx.SelectContext(ctx, tx, &result, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "list android host applicable profiles")
		}
		return nil
	})
	return result, err
}
