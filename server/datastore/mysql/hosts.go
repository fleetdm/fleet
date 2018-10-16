package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewHost(host *kolide.Host) (*kolide.Host, error) {
	sqlStatement := `
	INSERT INTO hosts (
		osquery_host_id,
		detail_update_time,
		node_key,
		host_name,
		uuid,
		platform,
		osquery_version,
		os_version,
		uptime,
		physical_memory,
		seen_time
	)
	VALUES( ?,?,?,?,?,?,?,?,?,?,? )
	`
	result, err := d.db.Exec(sqlStatement, host.OsqueryHostID, host.DetailUpdateTime,
		host.NodeKey, host.HostName, host.UUID, host.Platform, host.OsqueryVersion,
		host.OSVersion, host.Uptime, host.PhysicalMemory, host.SeenTime)
	if err != nil {
		return nil, errors.Wrap(err, "new host")
	}
	id, _ := result.LastInsertId()
	host.ID = uint(id)
	return host, nil
}

func removedUnusedNics(tx *sqlx.Tx, host *kolide.Host) error {
	if len(host.NetworkInterfaces) == 0 {
		_, err := tx.Exec(`DELETE FROM network_interfaces WHERE host_id = ?`, host.ID)
		return err
	}
	// Remove nics not associated with host
	sqlStatement := fmt.Sprintf(`
			DELETE FROM network_interfaces
			WHERE host_id = %d AND id NOT IN (?)
		`, host.ID)

	list := []uint{}
	for _, nic := range host.NetworkInterfaces {
		list = append(list, nic.ID)
	}

	sql, args, err := sqlx.In(sqlStatement, list)
	if err != nil {
		return err
	}

	sql = tx.Rebind(sql)
	_, err = tx.Exec(sql, args...)
	return err
}

func updateNicsForHost(tx *sqlx.Tx, host *kolide.Host) ([]*kolide.NetworkInterface, error) {
	updatedNics := []*kolide.NetworkInterface{}
	// id = LAST_INSERT_ID(id) is a fix for the lastinsertid not being set
	// properly. See comments in https://goo.gl/cwWRXd.
	sqlStatement := `
	 	INSERT INTO network_interfaces (
			host_id,
			mac,
			ip_address,
			broadcast,
			ibytes,
			interface,
			ipackets,
			last_change,
			mask,
			metric,
			mtu,
			obytes,
			ierrors,
			oerrors,
			opackets,
			point_to_point,
			type
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			mac = VALUES(mac),
			broadcast = VALUES(broadcast),
			ibytes = VALUES(ibytes),
			ipackets = VALUES(ipackets),
			last_change = VALUES(last_change),
			mask = VALUES(mask),
			metric = VALUES(metric),
			mtu = VALUES(mtu),
			obytes = VALUES(obytes),
			ierrors = VALUES(ierrors),
			oerrors = VALUES(oerrors),
			opackets = VALUES(opackets),
			point_to_point = VALUES(point_to_point),
			type = VALUES(type)
	 `
	for _, nic := range host.NetworkInterfaces {
		nic.HostID = host.ID
		result, err := tx.Exec(sqlStatement,
			nic.HostID,
			nic.MAC,
			nic.IPAddress,
			nic.Broadcast,
			nic.IBytes,
			nic.Interface,
			nic.IPackets,
			nic.LastChange,
			nic.Mask,
			nic.Metric,
			nic.MTU,
			nic.OBytes,
			nic.IErrors,
			nic.OErrors,
			nic.OPackets,
			nic.PointToPoint,
			nic.Type,
		)

		if err != nil {
			return nil, err
		}
		nicID, _ := result.LastInsertId()
		// if row was updated there is no LastInsertID
		if nicID != 0 {
			nic.ID = uint(nicID)
		}
		updatedNics = append(updatedNics, nic)
	}

	return updatedNics, nil
}

// TODO needs test
func (d *Datastore) SaveHost(host *kolide.Host) error {
	sqlStatement := `
		UPDATE hosts SET
			detail_update_time = ?,
			node_key = ?,
			host_name = ?,
			uuid = ?,
			platform = ?,
			osquery_version = ?,
			os_version = ?,
			uptime = ?,
			physical_memory = ?,
			cpu_type = ?,
			cpu_subtype = ?,
			cpu_brand = ?,
			cpu_physical_cores = ?,
			hardware_vendor = ?,
			hardware_model = ?,
			hardware_version = ?,
			hardware_serial = ?,
			computer_name = ?,
			primary_ip_id = ?,
			build = ?,
			platform_like = ?,
			code_name = ?,
			cpu_logical_cores = ?,
			seen_time = ?,
			distributed_interval = ?,
			config_tls_refresh = ?,
			logger_tls_period = ?
		WHERE id = ?
	`

	tx, err := d.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "creating transaction")
	}

	results, err := tx.Exec(sqlStatement,
		host.DetailUpdateTime,
		host.NodeKey,
		host.HostName,
		host.UUID,
		host.Platform,
		host.OsqueryVersion,
		host.OSVersion,
		host.Uptime,
		host.PhysicalMemory,
		host.CPUType,
		host.CPUSubtype,
		host.CPUBrand,
		host.CPUPhysicalCores,
		host.HardwareVendor,
		host.HardwareModel,
		host.HardwareVersion,
		host.HardwareSerial,
		host.ComputerName,
		host.PrimaryNetworkInterfaceID,
		host.Build,
		host.PlatformLike,
		host.CodeName,
		host.CPULogicalCores,
		host.SeenTime,
		host.DistributedInterval,
		host.ConfigTLSRefresh,
		host.LoggerTLSPeriod,
		host.ID)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "executing main SQL statement")
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "rows affected updating host")
	}
	if rowsAffected == 0 {
		tx.Rollback()
		return notFound("Host").WithID(host.ID)
	}

	host.NetworkInterfaces, err = updateNicsForHost(tx, host)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "updating nics")
	}

	if err = removedUnusedNics(tx, host); err != nil {
		tx.Rollback()
		return errors.Wrap(err, "removing unused nics")
	}

	if needsUpdate := host.ResetPrimaryNetwork(); needsUpdate {
		results, err = tx.Exec(
			"UPDATE hosts SET primary_ip_id = ? WHERE id = ?",
			host.PrimaryNetworkInterfaceID,
			host.ID,
		)

		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "resetting primary network")
		}
		rowsAffected, err = results.RowsAffected()
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "rows affected resetting primary network")
		}
		if rowsAffected == 0 {
			tx.Rollback()
			return notFound("Host").WithID(host.ID)
		}
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return errors.Wrap(err, "committing transaction")
	}
	return nil
}

func (d *Datastore) DeleteHost(hid uint) error {
	_, err := d.db.Exec("DELETE FROM hosts WHERE id = ?", hid)
	if err != nil {
		return errors.Wrapf(err, "deleting host with id %d", hid)
	}
	return nil
}

// TODO needs test
func (d *Datastore) Host(id uint) (*kolide.Host, error) {
	sqlStatement := `
		SELECT * FROM hosts
		WHERE id = ? AND NOT deleted LIMIT 1
	`
	host := &kolide.Host{}
	err := d.db.Get(host, sqlStatement, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting host by id")
	}

	if err := d.getNetInterfacesForHost(host); err != nil {
		return nil, err
	}

	return host, nil

}

func (d *Datastore) ListHosts(opt kolide.ListOptions) ([]*kolide.Host, error) {
	sqlStatement := `
		SELECT * FROM hosts
		WHERE NOT deleted
	`
	sqlStatement = appendListOptionsToSQL(sqlStatement, opt)
	hosts := []*kolide.Host{}
	if err := d.db.Select(&hosts, sqlStatement); err != nil {
		return nil, errors.Wrap(err, "list hosts")
	}

	if opt.PerPage == 0 || (opt.Page == 0 && uint(len(hosts)) < opt.PerPage) {
		// If all hosts, we can use the optimized network interface retrieval function
		if err := d.getNetInterfacesForAllHosts(hosts); err != nil {
			return nil, err
		}

	} else {
		if err := d.getNetInterfacesForHosts(hosts); err != nil {
			return nil, err
		}
	}

	return hosts, nil
}

func (d *Datastore) GenerateHostStatusStatistics(now time.Time) (online, offline, mia, new uint, e error) {
	// The logic in this function should remain synchronized with
	// host.Status and CountHostsInTargets

	sqlStatement := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? AND DATE_ADD(seen_time, INTERVAL 30 DAY) >= ? THEN 1 ELSE 0 END), 0) offline,
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
			COALESCE(SUM(CASE WHEN DATE_ADD(created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new
		FROM hosts
		LIMIT 1;
	`, kolide.OnlineIntervalBuffer, kolide.OnlineIntervalBuffer)

	counts := struct {
		MIA     uint `db:"mia"`
		Offline uint `db:"offline"`
		Online  uint `db:"online"`
		New     uint `db:"new"`
	}{}
	err := d.db.Get(&counts, sqlStatement, now, now, now, now, now)
	if err != nil && err != sql.ErrNoRows {
		e = errors.Wrap(err, "generating host statistics")
		return
	}

	mia = counts.MIA
	offline = counts.Offline
	online = counts.Online
	new = counts.New
	return online, offline, mia, new, nil
}

// Optimized network interface fetch for sets of hosts.  Instead of looping
// through hosts and doing a select for each host to get nics, we get all
// nics at once, so 2 db calls, and then assign nics to hosts here.
func (d *Datastore) getNetInterfacesForHosts(hosts []*kolide.Host) error {
	if len(hosts) == 0 {
		return nil
	}

	sqlStatement := `
		SELECT *
		FROM network_interfaces
		WHERE host_id IN (:hosts)
		ORDER BY host_id ASC
	`
	hostIDs := make([]uint, len(hosts))

	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	arg := map[string]interface{}{
		"hosts": hostIDs,
	}
	query, args, err := sqlx.Named(sqlStatement, arg)
	if err != nil {
		return errors.Wrap(err, "select nics for hosts, named query")
	}

	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return errors.Wrap(err, "select nics for hosts, in query")
	}

	query = d.db.Rebind(query)
	nics := []*kolide.NetworkInterface{}
	err = d.db.Select(&nics, query, args...)
	if err != nil {
		return errors.Wrap(err, "select nics for hosts, rebound query")
	}

	for _, host := range hosts {
		for i := 0; i < len(nics); i++ {
			if host.ID == nics[i].HostID {
				host.NetworkInterfaces = append(host.NetworkInterfaces, nics[i])
			}
		}
	}

	return nil
}

// When we know we're loading the network interfaces for all hosts, we can skip
// the IN clause and load them all. This allows us to load net interfaces
// without error for larger sets of hosts.
func (d *Datastore) getNetInterfacesForAllHosts(hosts []*kolide.Host) error {
	if len(hosts) == 0 {
		return nil
	}

	sqlStatement := `
		SELECT *
		FROM network_interfaces
		ORDER BY host_id ASC
	`
	nics := []*kolide.NetworkInterface{}
	err := d.db.Select(&nics, sqlStatement)
	if err != nil {
		return errors.Wrap(err, "select nics for all hosts")
	}

	for _, host := range hosts {
		for i := 0; i < len(nics); i++ {
			if host.ID == nics[i].HostID {
				host.NetworkInterfaces = append(host.NetworkInterfaces, nics[i])
			}
		}
	}

	return nil
}

func (d *Datastore) getNetInterfacesForHost(host *kolide.Host) error {
	sqlStatement := `
		SELECT * FROM network_interfaces
		WHERE host_id = ?
	`
	return d.db.Select(&host.NetworkInterfaces, sqlStatement, host.ID)
}

// EnrollHost enrolls a host
func (d *Datastore) EnrollHost(osqueryHostID string, nodeKeySize int) (*kolide.Host, error) {
	if osqueryHostID == "" {
		return nil, fmt.Errorf("missing osquery host identifier")
	}

	detailUpdateTime := time.Unix(0, 0).Add(24 * time.Hour)
	nodeKey, err := kolide.RandomText(nodeKeySize)
	if err != nil {
		return nil, errors.Wrap(err, "generating random text")
	}

	sqlInsert := `
		INSERT INTO hosts (
			detail_update_time,
			osquery_host_id,
			seen_time,
			node_key
		) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			node_key = VALUES(node_key),
			deleted = FALSE
	`

	var result sql.Result

	result, err = d.db.Exec(sqlInsert, detailUpdateTime, osqueryHostID, time.Now().UTC(), nodeKey)

	if err != nil {
		return nil, errors.Wrap(err, "inserting")
	}

	id, _ := result.LastInsertId()
	sqlSelect := `
		SELECT * FROM hosts WHERE id = ? LIMIT 1
	`
	host := &kolide.Host{}
	err = d.db.Get(host, sqlSelect, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting the host to return")
	}

	return host, nil

}

func (d *Datastore) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	sqlStatement := `
		SELECT *
		FROM hosts
		WHERE node_key = ? AND NOT deleted
		LIMIT 1
	`

	host := &kolide.Host{}
	if err := d.db.Get(host, sqlStatement, nodeKey); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, notFound("Host")
		default:
			return nil, errors.New("finding host")
		}
	}

	if err := d.getNetInterfacesForHost(host); err != nil {
		return nil, errors.Wrap(err, "getting interfaces")
	}

	return host, nil
}

func (d *Datastore) MarkHostSeen(host *kolide.Host, t time.Time) error {
	sqlStatement := `
		UPDATE hosts SET
			seen_time = ?
		WHERE node_key=?
	`

	_, err := d.db.Exec(sqlStatement, t, host.NodeKey)
	if err != nil {
		return errors.Wrap(err, "marking host seen")
	}

	host.UpdatedAt = t
	return nil
}

func (d *Datastore) searchHostsWithOmits(query string, omit ...uint) ([]*kolide.Host, error) {
	hostnameQuery := query
	if len(hostnameQuery) > 0 {
		hostnameQuery += "*"
	}

	ipQuery := `"` + query + `"`

	sqlStatement :=
		`
		SELECT DISTINCT *
		FROM hosts
		WHERE
		(
			id IN (
				SELECT id
				FROM hosts
				WHERE
				MATCH(host_name) AGAINST(? IN BOOLEAN MODE)
			)
		OR
			id IN (
				SELECT host_id
				FROM network_interfaces
				WHERE
				MATCH(ip_address) AGAINST(? IN BOOLEAN MODE)
			)
		)
		AND NOT deleted
		AND id NOT IN (?)
		LIMIT 10
	`

	sql, args, err := sqlx.In(sqlStatement, hostnameQuery, ipQuery, omit)
	if err != nil {
		return nil, errors.Wrap(err, "searching hosts")
	}
	sql = d.db.Rebind(sql)

	hosts := []*kolide.Host{}

	err = d.db.Select(&hosts, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "searching hosts rebound")
	}

	if err := d.getNetInterfacesForHosts(hosts); err != nil {
		return nil, errors.Wrap(err, "getting network interfaces for hosts")
	}

	return hosts, nil
}

func (d *Datastore) searchHostsDefault(omit ...uint) ([]*kolide.Host, error) {
	sqlStatement := `
	SELECT * FROM hosts
	WHERE NOT deleted
	AND id NOT IN (?)
	ORDER BY seen_time DESC
	LIMIT 5
	`

	var in interface{}
	{
		// use -1 if there are no values to omit.
		//Avoids empty args error for `sqlx.In`
		in = omit
		if len(omit) == 0 {
			in = -1
		}
	}

	var hosts []*kolide.Host
	sql, args, err := sqlx.In(sqlStatement, in)
	if err != nil {
		return nil, errors.Wrap(err, "searching default hosts")
	}
	sql = d.db.Rebind(sql)
	err = d.db.Select(&hosts, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "searching default hosts rebound")
	}
	if err := d.getNetInterfacesForHosts(hosts); err != nil {
		return nil, errors.Wrap(err, "getting network interfaces for default search hosts")
	}
	return hosts, nil
}

// SearchHosts find hosts by query containing an IP address or a host name. Optionally
// pass a list of IDs to omit from the search
func (d *Datastore) SearchHosts(query string, omit ...uint) ([]*kolide.Host, error) {
	if query == "" {
		return d.searchHostsDefault(omit...)
	}
	if len(omit) > 0 {
		return d.searchHostsWithOmits(query, omit...)
	}

	hostnameQuery := query
	hostnameQuery += "*"

	// Needs quotes to avoid each . marking a word boundary
	ipQuery := `"` + query + `"`

	sqlStatement :=
		`
		SELECT DISTINCT *
		FROM hosts
		WHERE
		(
			id IN (
				SELECT id
				FROM hosts
				WHERE
				MATCH(host_name) AGAINST(? IN BOOLEAN MODE)
			)
		OR
			id IN (
				SELECT host_id
				FROM network_interfaces
				WHERE
				MATCH(ip_address) AGAINST(? IN BOOLEAN MODE)
			)
		)
		AND NOT deleted
		LIMIT 10
	`
	hosts := []*kolide.Host{}

	if err := d.db.Select(&hosts, sqlStatement, hostnameQuery, ipQuery); err != nil {
		return nil, errors.Wrap(err, "searching hosts")
	}

	if err := d.getNetInterfacesForHosts(hosts); err != nil {
		return nil, errors.Wrap(err, "getting interfaces")
	}

	return hosts, nil

}

func (d *Datastore) DistributedQueriesForHost(host *kolide.Host) (map[uint]string, error) {
	sqlStatement := `
		SELECT DISTINCT dqc.id, q.query
		FROM distributed_query_campaigns dqc
		JOIN distributed_query_campaign_targets dqct
		    ON (dqc.id = dqct.distributed_query_campaign_id)
		LEFT JOIN label_query_executions lqe
		    ON (dqct.type = ? AND dqct.target_id = lqe.label_id AND lqe.matches)
		LEFT JOIN hosts h
		    ON ((dqct.type = ? AND lqe.host_id = h.id) OR (dqct.type = ? AND dqct.target_id = h.id))
		LEFT JOIN distributed_query_executions dqe
		    ON (h.id = dqe.host_id AND dqc.id = dqe.distributed_query_campaign_id)
		JOIN queries q
		    ON (dqc.query_id = q.id)
		WHERE dqe.status IS NULL AND dqc.status = ? AND h.id = ?
			AND NOT q.deleted
			AND NOT dqc.deleted
 `
	rows, err := d.db.Query(sqlStatement, kolide.TargetLabel, kolide.TargetLabel,
		kolide.TargetHost, kolide.QueryRunning, host.ID)
	if err != nil {
		return nil, errors.Wrap(err, "finding distributed queries for host")
	}
	defer rows.Close()

	results := map[uint]string{}

	for rows.Next() {
		var (
			id    uint
			query string
		)
		err = rows.Scan(&id, &query)
		if err != nil {
			return nil, errors.Wrap(err, "scanning query results")
		}

		results[id] = query

	}

	return results, nil
}

func (d *Datastore) HostIDsByName(hostnames []string) ([]uint, error) {
	sqlStatement := `
		SELECT id FROM hosts
		WHERE host_name IN (?)
	`

	sql, args, err := sqlx.In(sqlStatement, hostnames)
	if err != nil {
		return nil, errors.Wrap(err, "building query to get host IDs")
	}

	var hostIDs []uint
	if err := d.db.Select(&hostIDs, sql, args...); err != nil {
		return nil, errors.Wrap(err, "get host IDs")
	}

	return hostIDs, nil

}
