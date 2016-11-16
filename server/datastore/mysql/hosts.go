package mysql

import (
	"database/sql"
	"net/http"
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
		physical_memory,
		primary_mac,
		primary_ip
	)
	VALUES( ?,?,?,?,?,?,?,?,?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, host.DetailUpdateTime,
		host.NodeKey, host.HostName, host.UUID, host.Platform, host.OsqueryVersion,
		host.OSVersion, host.Uptime, host.PhysicalMemory, host.PrimaryMAC, host.PrimaryIP)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	id, _ := result.LastInsertId()
	host.ID = uint(id)
	return host, nil
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
			primary_mac = ?,
			primary_ip = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, host.DetailUpdateTime, host.NodeKey,
		host.HostName, host.UUID, host.Platform, host.OsqueryVersion,
		host.OSVersion, host.Uptime, host.PhysicalMemory, host.PrimaryMAC,
		host.PrimaryIP, host.ID)
	if err != nil {
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

	return host, nil

}

// TODO needs test
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

	return hosts, nil
}

// EnrollHost enrolls a host
func (d *Datastore) EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*kolide.Host, error) {
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
			platform,
			primary_ip
		) VALUES (?, ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			updated_at = VALUES(updated_at),
			detail_update_time = VALUES(detail_update_time),
			node_key = VALUES(node_key),
			host_name = VALUES(host_name),
			platform = VALUES(platform),
			primary_ip = VALUES(primary_ip),
			deleted = FALSE
	`
	args := []interface{}{}
	args = append(args, time.Unix(0, 0).Add(24*time.Hour))

	nodeKey, err := kolide.RandomText(nodeKeySize)

	args = append(args, nodeKey)
	args = append(args, hostname)
	args = append(args, uuid)
	args = append(args, platform)
	args = append(args, ip)

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

func (d *Datastore) MarkHostSeen(*kolide.Host, time.Time) error {
	panic("not implemented")
}

func (d *Datastore) searchHostsWithOmits(query string, omits ...uint) ([]kolide.Host, error) {
	// The reason that string cocantenation is used to include query as opposed to a
	// bindvar is that sqlx.In has a bug such that, if you have any bindvars other
	// than those in the IN clause, sqlx.In returns an empty sql statement.
	// I've submitted an issue https://github.com/jmoiron/sqlx/issues/260 about this
	sqlStatement := `
		SELECT *
		FROM hosts
		WHERE MATCH(host_name, primary_ip)
		AGAINST('` + query + "*" + `' IN BOOLEAN MODE)
		AND NOT deleted
		AND id NOT IN (?)
		LIMIT 10
	`

	sql, args, err := sqlx.In(sqlStatement, omits)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	sql = d.db.Rebind(sql)

	hosts := []kolide.Host{}

	if err = d.db.Select(&hosts, sql, args...); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return hosts, nil
}

// SearchHosts find hosts by query containing an IP address or a host name. Optionally
// pass a list of IDs to omit from the search
func (d *Datastore) SearchHosts(query string, omit ...uint) ([]kolide.Host, error) {
	if len(omit) > 0 {
		return d.searchHostsWithOmits(query, omit...)
	}

	sqlStatement := `
		SELECT * FROM hosts
		WHERE MATCH(host_name, primary_ip)
		AGAINST(? IN BOOLEAN MODE)
		AND not deleted
		LIMIT 10
	`
	hosts := []kolide.Host{}

	if err := d.db.Select(&hosts, sqlStatement, query); err != nil {
		return nil, errors.DatabaseError(err)
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
