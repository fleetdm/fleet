package mysql

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewHost(host *kolide.Host) (*kolide.Host, error) {
	sqlStatement := `
	INSERT INTO hosts (
		detail_update_time,
		node_key,
		host_name,
		uuid,
		platform,
		osquery_version,
		os_version,
		uptime,
		physical_memory
	)
	VALUES( ?,?,?,?,?,?,?,?,? )
	`
	result, err := d.db.Exec(sqlStatement, host.DetailUpdateTime,
		host.NodeKey, host.HostName, host.UUID, host.Platform, host.OsqueryVersion,
		host.OSVersion, host.Uptime, host.PhysicalMemory)
	if err != nil {
		return nil, errors.DatabaseError(err)
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
	if err != nil {
		return err
	}

	return nil
}

func updateNicsForHost(tx *sqlx.Tx, host *kolide.Host) ([]*kolide.NetworkInterface, error) {
	updatedNics := []*kolide.NetworkInterface{}
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
			cpu_logical_cores = ?
		WHERE id = ?
	`

	tx, err := d.db.Beginx()
	if err != nil {
		return errors.DatabaseError(err)
	}

	_, err = tx.Exec(sqlStatement,
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
		host.ID)
	if err != nil {
		tx.Rollback()
		return errors.DatabaseError(err)
	}

	host.NetworkInterfaces, err = updateNicsForHost(tx, host)
	if err != nil {
		tx.Rollback()
		return errors.DatabaseError(err)
	}
	if err = removedUnusedNics(tx, host); err != nil {
		tx.Rollback()
		return errors.DatabaseError(err)
	}

	if needsUpdate := host.ResetPrimaryNetwork(); needsUpdate {
		_, err = tx.Exec(
			"UPDATE hosts SET primary_ip_id = ? WHERE id = ?",
			*host.PrimaryNetworkInterfaceID,
			host.ID,
		)
		if err != nil {
			tx.Rollback()
			return errors.DatabaseError(err)
		}
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return errors.DatabaseError(err)
	}
	return nil
}

// TODO needs test
func (d *Datastore) DeleteHost(host *kolide.Host) error {
	sqlStatement := `
		UPDATE hosts SET
			deleted = TRUE,
			deleted_at = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, d.clock.Now(), host.ID)
	if err != nil {
		return errors.DatabaseError(err)
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
		return nil, errors.DatabaseError(err)
	}

	sqlStatement = `
		SELECT * from network_interfaces
		WHERE host_id = ?
	`
	err = d.db.Select(&host.NetworkInterfaces, sqlStatement, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
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
		return nil, errors.DatabaseError(err)
	}

	// TODO: This should be changed so that hosts and network interfaces
	// are grabbed in a single sql request,
	for _, host := range hosts {
		if err := d.getNetInterfacesForHost(host); err != nil {
			return nil, errors.DatabaseError(err)
		}
	}

	return hosts, nil
}

func (d *Datastore) getNetInterfacesForHost(host *kolide.Host) error {
	sqlStatement := `
		SELECT * FROM network_interfaces
		WHERE host_id = ?
	`
	if err := d.db.Select(&host.NetworkInterfaces, sqlStatement, host.ID); err != nil {
		return err
	}

	return nil
}

// EnrollHost enrolls a host
func (d *Datastore) EnrollHost(uuid, hostname, platform string, nodeKeySize int) (*kolide.Host, error) {
	if uuid == "" {
		return nil, errors.New("missing uuid for host enrollment", "programmer error")
	}
	// REVIEW If a deleted host is enrolled, it is undeleted
	sqlInsert := `
		INSERT INTO hosts (
			detail_update_time,
			node_key,
			host_name,
			uuid,
			platform
		) VALUES (?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			updated_at = VALUES(updated_at),
			detail_update_time = VALUES(detail_update_time),
			node_key = VALUES(node_key),
			host_name = VALUES(host_name),
			platform = VALUES(platform),
			deleted = FALSE
	`
	args := []interface{}{}
	args = append(args, time.Unix(0, 0).Add(24*time.Hour))

	nodeKey, err := kolide.RandomText(nodeKeySize)

	args = append(args, nodeKey)
	args = append(args, hostname)
	args = append(args, uuid)
	args = append(args, platform)

	var result sql.Result

	result, err = d.db.Exec(sqlInsert, args...)

	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	sqlSelect := `
		SELECT * FROM hosts WHERE id = ? LIMIT 1
	`
	host := &kolide.Host{}
	err = d.db.Get(host, sqlSelect, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return host, nil

}

func (d *Datastore) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	sqlStatement := `
		SELECT * FROM hosts
		WHERE node_key = ? AND NOT DELETED LIMIT 1
	`
	host := &kolide.Host{}
	if err := d.db.Get(host, sqlStatement, nodeKey); err != nil {
		switch err {
		case sql.ErrNoRows:
			e := errors.NewFromError(err, http.StatusUnauthorized, "invalid node key")
			e.Extra = map[string]interface{}{"node_invalid": "true"}
			return nil, e
		default:
			return nil, errors.DatabaseError(err)
		}
	}

	return host, nil

}

func (d *Datastore) MarkHostSeen(host *kolide.Host, t time.Time) error {
	sqlStatement := `
		UPDATE hosts SET
			updated_at = ?
		WHERE node_key=?
	`

	_, err := d.db.Exec(sqlStatement, t, host.NodeKey)
	if err != nil {
		return errors.DatabaseError(err)
	}

	host.UpdatedAt = t
	return nil
}

func (d *Datastore) searchHostsWithOmits(query string, omits ...uint) ([]kolide.Host, error) {

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
				MATCH(host_name) AGAINST('%s*' IN BOOLEAN MODE)
			)
		OR
			id IN (
				SELECT host_id
				FROM network_interfaces
				WHERE
				MATCH(ip_address) AGAINST('"%s"' IN BOOLEAN MODE)
			)
		)
		AND NOT deleted
		AND id NOT IN (%s)
		LIMIT 10
	`
	excludeList := ""
	for _, omit := range omits {
		if excludeList != "" {
			excludeList += ", "
		}
		excludeList += strconv.Itoa(int(omit))
	}

	sqlStatement = fmt.Sprintf(sqlStatement, query, query, excludeList)
	hosts := []kolide.Host{}

	if err := d.db.Select(&hosts, sqlStatement); err != nil {
		return nil, errors.DatabaseError(err)
	}

	for i := 0; i < len(hosts); i++ {
		if err := d.getNetInterfacesForHost(&hosts[i]); err != nil {
			return nil, errors.DatabaseError(err)
		}
	}

	return hosts, nil
}

// SearchHosts find hosts by query containing an IP address or a host name. Optionally
// pass a list of IDs to omit from the search
func (d *Datastore) SearchHosts(query string, omit ...uint) ([]kolide.Host, error) {
	if len(omit) > 0 {
		return d.searchHostsWithOmits(query, omit...)
	}

	hostNameQuery := fmt.Sprintf(`%s*`, query)
	ipQuery := fmt.Sprintf(`"%s"`, query)

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
	hosts := []kolide.Host{}

	if err := d.db.Select(&hosts, sqlStatement, hostNameQuery, ipQuery); err != nil {
		return nil, errors.DatabaseError(err)
	}
	// We're not using range to avoid having to copy the hosts slice
	// when we populate the host NewworkInterfaces
	for i := 0; i < len(hosts); i++ {
		if err := d.getNetInterfacesForHost(&hosts[i]); err != nil {
			return nil, errors.DatabaseError(err)
		}
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
		return nil, errors.DatabaseError(err)
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
			return nil, errors.DatabaseError(err)
		}

		results[id] = query

	}

	return results, nil
}
