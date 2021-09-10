package mysql

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (ds *Datastore) NewGlobalPolicy(queryID uint) (*fleet.Policy, error) {
	res, err := ds.writer.Exec(`INSERT INTO policies (query_id) VALUES (?)`, queryID)
	if err != nil {
		return nil, errors.Wrap(err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "getting last id after inserting policy")
	}

	return policyDB(ds.writer, uint(lastIdInt64))
}

func (ds *Datastore) Policy(id uint) (*fleet.Policy, error) {
	return policyDB(ds.reader, id)
}

func policyDB(q sqlx.Queryer, id uint) (*fleet.Policy, error) {
	var policy fleet.Policy
	err := sqlx.Get(q, &policy,
		`SELECT
       		p.*,
       		q.name as query_name,
       		(select count(*) from policy_membership where policy_id=p.id and passes=true) as passing_host_count,
       		(select count(*) from policy_membership where policy_id=p.id and passes=false) as failing_host_count
		FROM policies p JOIN queries q ON (p.query_id=q.id) WHERE p.id=?`,
		id)
	if err != nil {
		return nil, errors.Wrap(err, "getting policy")
	}
	return &policy, nil
}

func (ds *Datastore) RecordPolicyQueryExecutions(host *fleet.Host, results map[uint]*bool, updated time.Time) error {
	// Sort the results to have generated SQL queries ordered to minimize
	// deadlocks. See https://github.com/fleetdm/fleet/issues/1146.
	orderedIDs := make([]uint, 0, len(results))
	for policyID := range results {
		orderedIDs = append(orderedIDs, policyID)
	}
	sort.Slice(orderedIDs, func(i, j int) bool { return orderedIDs[i] < orderedIDs[j] })

	// Loop through results, collecting which labels we need to insert/update
	vals := []interface{}{}
	bindvars := []string{}
	for _, policyID := range orderedIDs {
		matches := results[policyID]
		bindvars = append(bindvars, "(?,?,?,?)")
		vals = append(vals, updated, policyID, host.ID, matches)
	}

	query := fmt.Sprintf(
		`INSERT INTO policy_membership_history (updated_at, policy_id, host_id, passes)
				VALUES %s`,
		strings.Join(bindvars, ","),
	)

	_, err := ds.writer.Exec(query, vals...)
	if err != nil {
		return errors.Wrapf(err, "insert policy_membership (%v)", vals)
	}

	return nil
}

func (ds *Datastore) ListGlobalPolicies() ([]*fleet.Policy, error) {
	var policies []*fleet.Policy
	err := ds.reader.Select(
		&policies,
		`SELECT
       		p.*,
       		q.name as query_name,
       		(select count(*) from policy_membership where policy_id=p.id and passes=true) as passing_host_count,
       		(select count(*) from policy_membership where policy_id=p.id and passes=false) as failing_host_count
		FROM policies p JOIN queries q ON (p.query_id=q.id)`,
	)
	if err != nil {
		return nil, errors.Wrap(err, "listing policies")
	}
	return policies, nil
}

func (ds *Datastore) DeleteGlobalPolicies(ids []uint) ([]uint, error) {
	stmt := `DELETE FROM policies WHERE id IN (?)`
	stmt, args, err := sqlx.In(stmt, ids)
	if err != nil {
		return nil, errors.Wrap(err, "IN for DELETE FROM policies")
	}
	stmt = ds.writer.Rebind(stmt)
	if _, err := ds.writer.Exec(stmt, args...); err != nil {
		return nil, errors.Wrap(err, "delete policies")
	}
	return ids, nil
}

func (ds *Datastore) PolicyQueriesForHost(_ *fleet.Host) (map[string]string, error) {
	var rows []struct {
		Id    string `db:"id"`
		Query string `db:"query"`
	}
	err := ds.reader.Select(&rows, `SELECT p.id, q.query FROM policies p JOIN queries q ON (p.query_id=q.id)`)
	if err != nil {
		return nil, errors.Wrap(err, "selecting policies for host")
	}

	results := map[string]string{}

	for _, row := range rows {
		results[row.Id] = row.Query
	}

	return results, nil
}
