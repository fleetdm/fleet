package mysql

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

// NewQuery creates a Query
func (d *Datastore) NewQuery(query *kolide.Query) (*kolide.Query, error) {

	sql := `
		INSERT INTO queries  ( name, description, query,
			snapshot, differential, platform, version, ` + "`interval`" + `)
		VALUES ( ?, ?, ?, ?, ?, ?, ?, ? )
	`

	result, err := d.db.Exec(sql, query.Name, query.Description, query.Query, query.Snapshot,
		query.Differential, query.Platform, query.Version, query.Interval)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	query.ID = uint(id)
	return query, nil
}

// SaveQuery saves changes to a Query.
func (d *Datastore) SaveQuery(q *kolide.Query) error {
	sql := `
		UPDATE queries
			SET name = ?, description = ?, query = ?, ` + "`interval`" + ` = ?, snapshot = ?,
			 	differential = ?, platform = ?, version = ?
			WHERE id = ? AND NOT deleted
	`
	_, err := d.db.Exec(sql, q.Name, q.Description, q.Query, q.Interval,
		q.Snapshot, q.Differential, q.Platform, q.Version, q.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// DeleteQuery soft deletes Query identified by Query.ID
func (d *Datastore) DeleteQuery(query *kolide.Query) error {
	query.MarkDeleted(d.clock.Now())
	sql := `
		UPDATE queries
			SET deleted_at = ?, deleted = ?
			WHERE id = ?
	`
	_, err := d.db.Exec(sql, query.DeletedAt, true, query.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// Query returns a single Query identified by id, if such
// exists
func (d *Datastore) Query(id uint) (*kolide.Query, error) {
	sql := `
		SELECT * FROM queries WHERE id = ? AND NOT deleted
	`
	query := &kolide.Query{}
	if err := d.db.Get(query, sql, id); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return query, nil
}

// ListQueries returns a list of queries with sort order and results limit
// determined by passed in kolide.ListOptions
func (d *Datastore) ListQueries(opt kolide.ListOptions) ([]*kolide.Query, error) {
	sql := `
		SELECT * FROM queries WHERE NOT deleted
	`
	sql = appendListOptionsToSQL(sql, opt)
	results := []*kolide.Query{}

	if err := d.db.Select(&results, sql); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return results, nil

}

func (d *Datastore) SaveDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) error {

	sqlStatement := `
		UPDATE distributed_query_campaigns SET
			query_id = ?,
			max_duration = ?,
			status = ?,
			user_id = ?
		WHERE id = ?
		AND NOT deleted
	`
	_, err := d.db.Exec(sqlStatement, camp.QueryID, camp.MaxDuration,
		camp.Status, camp.UserID, camp.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) NewDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) (*kolide.DistributedQueryCampaign, error) {

	sqlStatement := `
		INSERT INTO distributed_query_campaigns (
			query_id,
			max_duration,
			status,
			user_id
		)
		VALUES(?,?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, camp.QueryID, camp.MaxDuration, camp.Status, camp.UserID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	camp.ID = uint(id)
	return camp, nil
}

func (d *Datastore) NewDistributedQueryCampaignTarget(target *kolide.DistributedQueryCampaignTarget) (*kolide.DistributedQueryCampaignTarget, error) {
	sqlStatement := `
		INSERT into distributed_query_campaign_targets (
			type,
			distributed_query_campaign_id,
			target_id
		)
		VALUES (?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, target.Type, target.DistributedQueryCampaignID, target.TargetID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	target.ID = uint(id)
	return target, nil
}

func (d *Datastore) NewDistributedQueryExecution(exec *kolide.DistributedQueryExecution) (*kolide.DistributedQueryExecution, error) {
	sqlStatement := `
		INSERT INTO distributed_query_executions (
			host_id,
			distributed_query_campaign_id,
			status,
			error,
			execution_duration
		) VALUES (?,?,?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, exec.HostID, exec.DistributedQueryCampaignID,
		exec.Status, exec.Error, exec.ExecutionDuration)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	exec.ID = uint(id)

	return exec, nil
}
