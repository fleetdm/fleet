package mysql

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (ds *Datastore) NewGlobalPolicy(queryID uint) (*fleet.Policy, error) {
	res, err := ds.db.Exec(`INSERT INTO policies (query_id) VALUES (?)`, queryID)
	if err != nil {
		return nil, errors.Wrap(err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "getting last id after inserting policy")
	}

	return ds.Policy(uint(lastIdInt64))
}

func (ds *Datastore) Policy(id uint) (*fleet.Policy, error) {
	var policy fleet.Policy
	err := ds.db.Get(
		&policy,
		`SELECT p.*, q.name as query_name FROM policies p JOIN queries q ON (p.query_id=q.id) WHERE p.id=?`,
		id,
	)
	if err != nil {
		return nil, errors.Wrap(err, "getting policy")
	}
	return &policy, nil
}

func (ds *Datastore) RecordPolicyQueryExecutions(host *fleet.Host, results map[uint]*bool, updated time.Time) error {
	// Sort the results to have generated SQL queries ordered to minimize
	// deadlocks. See https://github.com/fleetdm/fleet/issues/1146.
	orderedIDs := make([]uint, 0, len(results))
	for policyID, _ := range results {
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
		`INSERT INTO policy_membership (updated_at, policy_id, host_id, passes)
				VALUES %s ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at), passes = VALUES(passes)`,
		strings.Join(bindvars, ","),
	)

	_, err := ds.db.Exec(query, vals...)
	if err != nil {
		return errors.Wrapf(err, "insert policy_membership (%v)", vals)
	}

	return nil
}

func (ds *Datastore) ListGlobalPolicies() ([]*fleet.Policy, error) {
	var policies []*fleet.Policy
	err := ds.db.Select(
		&policies,
		`SELECT p.*, q.name as query_name FROM policies p JOIN queries q ON (p.query_id=q.id)`,
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
	stmt = ds.db.Rebind(stmt)
	if _, err := ds.db.Exec(stmt, args...); err != nil {
		return nil, errors.Wrap(err, "delete policies")
	}
	return ids, nil
}

func (ds *Datastore) PolicyQueriesForHost(_ *fleet.Host) (map[string]string, error) {
	var rows *sql.Rows
	var err error
	rows, err = ds.db.Query(`SELECT p.id, q.query FROM policies p JOIN queries q ON (p.query_id=q.id)`)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "selecting policies for host")
	}

	defer rows.Close()
	results := map[string]string{}

	for rows.Next() {
		var id, query string

		if err = rows.Scan(&id, &query); err != nil {
			return nil, errors.Wrap(err, "scanning policy queries for host")
		}

		results[id] = query
	}

	return results, nil
}
