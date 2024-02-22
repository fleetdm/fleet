package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

const policyCols = `
	p.id, p.team_id, p.resolution, p.name, p.query, p.description,
	p.author_id, p.platforms, p.created_at, p.updated_at, p.critical
`

var policySearchColumns = []string{"p.name"}

func (ds *Datastore) NewGlobalPolicy(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
	if args.QueryID != nil {
		q, err := ds.Query(ctx, *args.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetching query from id")
		}
		args.Name = q.Name
		args.Query = q.Query
		args.Description = q.Description
	}
	res, err := ds.writer(ctx).ExecContext(ctx,
		fmt.Sprintf(
			`INSERT INTO policies (name, query, description, resolution, author_id, platforms, critical, checksum) VALUES (?, ?, ?, ?, ?, ?, ?, %s)`,
			policiesChecksumComputedColumn(),
		),
		args.Name, args.Query, args.Description, args.Resolution, authorID, args.Platform, args.Critical,
	)
	switch {
	case err == nil:
		// OK
	case isDuplicate(err):
		return nil, ctxerr.Wrap(ctx, alreadyExists("Policy", args.Name))
	default:
		return nil, ctxerr.Wrap(ctx, err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last id after inserting policy")
	}
	return policyDB(ctx, ds.writer(ctx), uint(lastIdInt64), nil)
}

func policiesChecksumComputedColumn() string {
	// concatenate with separator \x00
	return ` UNHEX(
		MD5(
			CONCAT_WS(CHAR(0),
				COALESCE(team_id, ''),
				name
			)
		)
	) `
}

func (ds *Datastore) Policy(ctx context.Context, id uint) (*fleet.Policy, error) {
	return policyDB(ctx, ds.reader(ctx), id, nil)
}

func (ds *Datastore) PolicyByName(ctx context.Context, name string) (*fleet.Policy, error) {
	var policy fleet.Policy
	err := sqlx.GetContext(ctx, ds.reader(ctx), &policy,
		fmt.Sprint(`SELECT `+policyCols+`,
		COALESCE(u.name, '<deleted>') AS author_name,
		COALESCE(u.email, '') AS author_email,
		ps.updated_at as host_count_updated_at,
		COALESCE(ps.passing_host_count, 0) as passing_host_count,
		COALESCE(ps.failing_host_count, 0) as failing_host_count
		FROM policies p
		LEFT JOIN users u ON p.author_id = u.id
		LEFT JOIN policy_stats ps ON p.id = ps.policy_id
			AND ((p.team_id IS NULL AND ps.inherited_team_id = 0)
				OR (p.team_id IS NOT NULL AND ps.inherited_team_id = p.team_id))
		WHERE p.name=?`), name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Policy").WithName(name))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting policy")
	}
	return &policy, nil
}

func policyDB(ctx context.Context, q sqlx.QueryerContext, id uint, teamID *uint) (*fleet.Policy, error) {
	teamWhere := "TRUE"
	args := []interface{}{id}
	if teamID != nil {
		teamWhere = "team_id = ?"
		args = append(args, *teamID)
	}

	var policy fleet.Policy
	err := sqlx.GetContext(ctx, q, &policy,
		fmt.Sprintf(`
		SELECT %s,
		    COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
			ps.updated_at as host_count_updated_at,
			COALESCE(ps.passing_host_count, 0) as passing_host_count,
			COALESCE(ps.failing_host_count, 0) as failing_host_count
		FROM policies p
		LEFT JOIN users u ON p.author_id = u.id
		LEFT JOIN policy_stats ps ON p.id = ps.policy_id
		AND ((p.team_id IS NULL AND ps.inherited_team_id = 0)
			OR (p.team_id IS NOT NULL AND ps.inherited_team_id = p.team_id))
		WHERE p.id=? AND %s`, policyCols, teamWhere),
		args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Policy").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting policy")
	}
	return &policy, nil
}

// SavePolicy updates some fields of the given policy on the datastore.
//
// Currently SavePolicy does not allow updating the team of an existing policy.
func (ds *Datastore) SavePolicy(ctx context.Context, p *fleet.Policy, shouldRemoveAllPolicyMemberships bool) error {
	sql := `
		UPDATE policies
			SET name = ?, query = ?, description = ?, resolution = ?, platforms = ?, critical = ?, checksum = ` + policiesChecksumComputedColumn() + `
			WHERE id = ?
	`
	result, err := ds.writer(ctx).ExecContext(ctx, sql, p.Name, p.Query, p.Description, p.Resolution, p.Platform, p.Critical, p.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating policy")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected updating policy")
	}
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("Policy").WithID(p.ID))
	}

	if shouldRemoveAllPolicyMemberships {
		return ds.cleanupPolicyMembershipForPolicy(ctx, p.ID)
	}
	return cleanupPolicyMembershipOnPolicyUpdate(ctx, ds.writer(ctx), p.ID, p.Platform)
}

// FlippingPoliciesForHost fetches previous policy membership results and returns:
//   - a list of "new" failing policies; "new" here means those that fail on their first
//     run, and those that were passing on the previous run and are failing on the incoming execution.
//   - a list of "new" passing policies; "new" here means those that failed on a previous
//     run and are passing now.
//
// "Failure" here means the policy query executed successfully but didn't return any rows,
// so policies that did not execute (incomingResults with nil bool) are ignored.
//
// NOTES(lucas):
//   - If a policy has been deleted (also deleted on `policy_membership` via cascade)
//     and osquery agents bring in new failing results from them then those will be returned here
//     (in newFailing or newPassing).
//   - Similar in case a host was deleted.
//
// Trying to filter those out here would make this operation more expensive (fetch policies from the
// `policies` table and querying the `hosts` table).
func (ds *Datastore) FlippingPoliciesForHost(
	ctx context.Context,
	hostID uint,
	incomingResults map[uint]*bool,
) (newFailing []uint, newPassing []uint, err error) {
	orderedIDs := make([]uint, 0, len(incomingResults))
	filteredIncomingResults := filterNotExecuted(incomingResults)
	for policyID := range filteredIncomingResults {
		orderedIDs = append(orderedIDs, policyID)
	}
	if len(orderedIDs) == 0 {
		return nil, nil, nil
	}
	// Sort the results to have generated SQL queries ordered to minimize deadlocks (see #1146).
	sort.Slice(orderedIDs, func(i, j int) bool {
		return orderedIDs[i] < orderedIDs[j]
	})
	// By using `passes IS NOT NULL` we filter out those policies that never executed properly.
	selectQuery := `SELECT policy_id, passes FROM policy_membership
		WHERE host_id = ? AND policy_id IN (?) AND passes IS NOT NULL`
	var fetchedPolicyResults []struct {
		ID     uint `db:"policy_id"`
		Passes bool `db:"passes"`
	}
	selectQuery, args, err := sqlx.In(selectQuery, hostID, orderedIDs)
	if err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "build select policy_membership query")
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &fetchedPolicyResults, selectQuery, args...); err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "select policy_membership")
	}
	prevPolicyResults := make(map[uint]bool)
	for _, result := range fetchedPolicyResults {
		prevPolicyResults[result.ID] = result.Passes
	}
	newFailing, newPassing = flipping(prevPolicyResults, filteredIncomingResults)
	return newFailing, newPassing, nil
}

func flipping(prevResults map[uint]bool, incomingResults map[uint]bool) (newFailing, newPassing []uint) {
	for policyID, incomingPasses := range incomingResults {
		prevPasses, ok := prevResults[policyID]
		if !ok { // first run
			if !incomingPasses {
				newFailing = append(newFailing, policyID)
			}
		} else { // it run previously
			if !prevPasses && incomingPasses {
				newPassing = append(newPassing, policyID)
			} else if prevPasses && !incomingPasses {
				newFailing = append(newFailing, policyID)
			}
		}
	}
	return newFailing, newPassing
}

func filterNotExecuted(results map[uint]*bool) map[uint]bool {
	filtered := make(map[uint]bool)
	for id, result := range results {
		if result != nil {
			filtered[id] = *result
		}
	}
	return filtered
}

func (ds *Datastore) RecordPolicyQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool) error {
	vals := []interface{}{}
	bindvars := []string{}
	if len(results) > 0 {
		// Sort the results to have generated SQL queries ordered to minimize
		// deadlocks. See https://github.com/fleetdm/fleet/issues/1146.
		orderedIDs := make([]uint, 0, len(results))
		for policyID := range results {
			orderedIDs = append(orderedIDs, policyID)
		}
		sort.Slice(orderedIDs, func(i, j int) bool { return orderedIDs[i] < orderedIDs[j] })

		// Loop through results, collecting which labels we need to insert/update
		for _, policyID := range orderedIDs {
			matches := results[policyID]
			bindvars = append(bindvars, "(?,?,?,?)")
			vals = append(vals, updated, policyID, host.ID, matches)
		}
	}

	// NOTE: the insert of policy membership that follows must be kept in sync
	// with the async implementation in AsyncBatchInsertPolicyMembership, and the
	// update of the policy_updated_at timestamp in sync with the
	// AsyncBatchUpdatePolicyTimestamp method (that is, their processing must be
	// semantically equivalent, even though here it processes a single host and
	// in async mode it processes a batch of hosts).

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if len(results) > 0 {
			query := fmt.Sprintf(
				`INSERT INTO policy_membership (updated_at, policy_id, host_id, passes)
				VALUES %s ON DUPLICATE KEY UPDATE updated_at=VALUES(updated_at), passes=VALUES(passes)`,
				strings.Join(bindvars, ","),
			)
			_, err := tx.ExecContext(ctx, query, vals...)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "insert policy_membership (%v)", vals)
			}
		}

		// if we are deferring host updates, we return at this point and do the change outside of the tx
		if deferredSaveHost {
			return nil
		}

		if _, err := tx.ExecContext(ctx, `UPDATE hosts SET policy_updated_at = ? WHERE id=?`, updated, host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "updating hosts policy updated at")
		}
		return nil
	})
	if err != nil {
		return err
	}

	if deferredSaveHost {
		errCh := make(chan error, 1)
		defer close(errCh)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ds.writeCh <- itemToWrite{
			ctx:   ctx,
			errCh: errCh,
			item: hostXUpdatedAt{
				hostID:    host.ID,
				updatedAt: updated,
				what:      "policy_updated_at",
			},
		}:
			return <-errCh
		}
	}
	return nil
}

func (ds *Datastore) ListGlobalPolicies(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) {
	return listPoliciesDB(ctx, ds.reader(ctx), nil, opts)
}

// returns the list of policies associated with the provided teamID, or the
// global policies if teamID is nil. The pass/fail host counts are the totals
// regardless of hosts' team if countsForTeamID is nil, or the totals just for
// hosts that belong to the provided countsForTeamID if it is not nil.
func listPoliciesDB(ctx context.Context, q sqlx.QueryerContext, teamID *uint, opts fleet.ListOptions) ([]*fleet.Policy, error) {
	var args []interface{}

	query := `
		SELECT ` + policyCols + `,
			COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
			ps.updated_at as host_count_updated_at,
			COALESCE(ps.passing_host_count, 0) AS passing_host_count,
			COALESCE(ps.failing_host_count, 0) AS failing_host_count
		FROM policies p
		LEFT JOIN users u ON p.author_id = u.id
		LEFT JOIN policy_stats ps ON p.id = ps.policy_id AND ps.inherited_team_id = 0
	`

	if teamID != nil {
		query += " WHERE team_id = ?"
		args = append(args, *teamID)
	} else {
		query += " WHERE team_id IS NULL"
	}

	query, args = searchLike(query, args, opts.MatchQuery, policySearchColumns...)
	query, args = appendListOptionsWithCursorToSQL(query, args, &opts)

	var policies []*fleet.Policy
	err := sqlx.SelectContext(ctx, q, &policies, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing policies")
	}

	return policies, nil
}

// getInheritedPoliciesForTeam returns the list of global policies with the
// passing and failing host counts for the provided teamID
func getInheritedPoliciesForTeam(ctx context.Context, q sqlx.QueryerContext, TeamID uint, opts fleet.ListOptions) ([]*fleet.Policy, error) {
	var args []interface{}

	query := `
        SELECT 
            ` + policyCols + `,
			COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
			ps.updated_at as host_count_updated_at,
            COALESCE(ps.passing_host_count, 0) as passing_host_count,
            COALESCE(ps.failing_host_count, 0) as failing_host_count
        FROM policies p
        LEFT JOIN users u ON p.author_id = u.id
        LEFT JOIN policy_stats ps ON p.id = ps.policy_id AND ps.inherited_team_id = ?
        WHERE p.team_id IS NULL
    `

	args = append(args, TeamID)

	query, args = searchLike(query, args, opts.MatchQuery, policySearchColumns...)
	query, _ = appendListOptionsToSQL(query, &opts)

	var policies []*fleet.Policy
	err := sqlx.SelectContext(ctx, q, &policies, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing inherited policies")
	}

	return policies, nil
}

// CountPolicies returns the total number of team policies.
// If teamID is nil, it returns the total number of global policies.
func (ds *Datastore) CountPolicies(ctx context.Context, teamID *uint, matchQuery string) (int, error) {
	var (
		query string
		args  []interface{}
		count int
	)

	if teamID == nil {
		query = `SELECT count(*) FROM policies p WHERE team_id IS NULL`
	} else {
		query = `SELECT count(*) FROM policies p WHERE team_id = ?`
		args = append(args, *teamID)
	}

	query, args = searchLike(query, args, matchQuery, policySearchColumns...)

	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting policies")
	}

	return count, nil
}

func (ds *Datastore) PoliciesByID(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
	sql := `SELECT ` + policyCols + `,
	  COALESCE(u.name, '<deleted>') AS author_name,
	  COALESCE(u.email, '') AS author_email,
	  ps.updated_at as host_count_updated_at,
	  COALESCE(ps.passing_host_count, 0) as passing_host_count,
	  COALESCE(ps.failing_host_count, 0) as failing_host_count
	  FROM policies p
	  LEFT JOIN users u ON p.author_id = u.id
	  LEFT JOIN policy_stats ps ON p.id = ps.policy_id
	  	AND ((p.team_id IS NULL AND ps.inherited_team_id = 0)
				OR (p.team_id IS NOT NULL AND ps.inherited_team_id = p.team_id))
	  WHERE p.id IN (?)`
	query, args, err := sqlx.In(sql, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get policies by ID")
	}

	var policies []*fleet.Policy
	err = sqlx.SelectContext(
		ctx,
		ds.reader(ctx),
		&policies,
		query, args...,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting policies by ID")
	}

	policiesByID := make(map[uint]*fleet.Policy, len(ids))
	for _, p := range policies {
		policiesByID[p.ID] = p
	}
	for _, id := range ids {
		if policiesByID[id] == nil {
			return nil, ctxerr.Wrap(ctx, notFound("Policy").WithID(id))
		}
	}

	return policiesByID, nil
}

func (ds *Datastore) DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error) {
	return deletePolicyDB(ctx, ds.writer(ctx), ids, nil)
}

func deletePolicyDB(ctx context.Context, q sqlx.ExtContext, ids []uint, teamID *uint) ([]uint, error) {
	stmt := `DELETE FROM policies WHERE id IN (?) AND %s`
	stmt, args, err := sqlx.In(stmt, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "IN for DELETE FROM policies")
	}
	stmt = q.Rebind(stmt)

	teamWhere := "TRUE"
	if teamID != nil {
		teamWhere = "team_id = ?"
		args = append(args, *teamID)
	}

	if _, err := q.ExecContext(ctx, fmt.Sprintf(stmt, teamWhere), args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "delete policies")
	}
	return ids, nil
}

// PolicyQueriesForHost returns the policy queries that are to be executed on the given host.
func (ds *Datastore) PolicyQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, error) {
	var rows []struct {
		ID    string `db:"id"`
		Query string `db:"query"`
	}
	if host.FleetPlatform() == "" {
		// We log to help troubleshooting in case this happens, as the host
		// won't be receiving any policies targeted for specific platforms.
		level.Error(ds.logger).Log("err", "unrecognized platform", "hostID", host.ID, "platform", host.Platform) //nolint:errcheck
	}
	q := dialect.From("policies").Select(
		goqu.I("id"),
		goqu.I("query"),
	).Where(
		goqu.And(
			goqu.Or(
				goqu.I("platforms").Eq(""),
				goqu.L("FIND_IN_SET(?, ?)",
					host.FleetPlatform(),
					goqu.I("platforms"),
				).Neq(0),
			),
			goqu.Or(
				goqu.I("team_id").IsNull(),        // global policies
				goqu.I("team_id").Eq(host.TeamID), // team policies
			),
		),
	)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting policies sql build")
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting policies for host")
	}
	results := make(map[string]string)
	for _, row := range rows {
		results[row.ID] = row.Query
	}
	return results, nil
}

func (ds *Datastore) NewTeamPolicy(ctx context.Context, teamID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
	if args.QueryID != nil {
		q, err := ds.Query(ctx, *args.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetching query from id")
		}
		args.Name = q.Name
		args.Query = q.Query
		args.Description = q.Description
	}
	res, err := ds.writer(ctx).ExecContext(ctx,
		fmt.Sprintf(
			`INSERT INTO policies (name, query, description, team_id, resolution, author_id, platforms, critical, checksum) VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s)`,
			policiesChecksumComputedColumn(),
		),
		args.Name, args.Query, args.Description, teamID, args.Resolution, authorID, args.Platform, args.Critical)
	switch {
	case err == nil:
		// OK
	case isDuplicate(err):
		return nil, ctxerr.Wrap(ctx, alreadyExists("Policy", args.Name))
	default:
		return nil, ctxerr.Wrap(ctx, err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last id after inserting policy")
	}
	return policyDB(ctx, ds.writer(ctx), uint(lastIdInt64), &teamID)
}

func (ds *Datastore) ListTeamPolicies(ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions) (teamPolicies, inheritedPolicies []*fleet.Policy, err error) {
	teamPolicies, err = listPoliciesDB(ctx, ds.reader(ctx), &teamID, opts)
	if err != nil {
		return nil, nil, err
	}
	// get inherited (global) policies with counts of hosts for that team
	inheritedPolicies, err = getInheritedPoliciesForTeam(ctx, ds.reader(ctx), teamID, iopts)
	if err != nil {
		return nil, nil, err
	}
	return teamPolicies, inheritedPolicies, err
}

func (ds *Datastore) DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
	return deletePolicyDB(ctx, ds.writer(ctx), ids, &teamID)
}

func (ds *Datastore) TeamPolicy(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
	return policyDB(ctx, ds.reader(ctx), policyID, &teamID)
}

// ApplyPolicySpecs applies the given policy specs, creating new policies and updating the ones that
// already exist (a policy is identified by its name).
//
// NOTE: Similar to ApplyQueries, ApplyPolicySpecs will update the author_id of the policies
// that are updated.
//
// Currently ApplyPolicySpecs does not allow updating the team of an existing policy.
func (ds *Datastore) ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		query := fmt.Sprintf(
			`
		INSERT INTO policies (
			name,
			query,
			description,
			author_id,
			resolution,
			team_id,
			platforms,
			critical,
			checksum
		) VALUES ( ?, ?, ?, ?, ?, (SELECT IFNULL(MIN(id), NULL) FROM teams WHERE name = ?), ?, ?, %s)
		ON DUPLICATE KEY UPDATE
			query = VALUES(query),
			description = VALUES(description),
			author_id = VALUES(author_id),
			resolution = VALUES(resolution),
			platforms = VALUES(platforms),
			critical = VALUES(critical)
		`, policiesChecksumComputedColumn(),
		)
		for _, spec := range specs {

			res, err := tx.ExecContext(ctx,
				query, spec.Name, spec.Query, spec.Description, authorID, spec.Resolution, spec.Team, spec.Platform, spec.Critical,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "exec ApplyPolicySpecs insert")
			}

			if insertOnDuplicateDidUpdate(res) {
				// when the upsert results in an UPDATE that *did* change some values,
				// it returns the updated ID as last inserted id.
				if lastID, _ := res.LastInsertId(); lastID > 0 {
					if err := cleanupPolicyMembershipOnPolicyUpdate(ctx, tx, uint(lastID), spec.Platform); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func amountPoliciesDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `SELECT count(*) FROM policies`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

// AsyncBatchInsertPolicyMembership inserts into the policy_membership table
// the batch of policy membership results.
func (ds *Datastore) AsyncBatchInsertPolicyMembership(ctx context.Context, batch []fleet.PolicyMembershipResult) error {
	// NOTE: this is tested via the server/service/async package tests.

	// INSERT IGNORE, to avoid failing if policy / host does not exist (as this
	// runs asynchronously, they could get deleted in between the data being
	// received and being upserted).
	sql := `INSERT IGNORE INTO policy_membership (policy_id, host_id, passes) VALUES `
	sql += strings.Repeat(`(?, ?, ?),`, len(batch))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at), passes = VALUES(passes)`

	vals := make([]interface{}, 0, len(batch)*3)
	for _, tup := range batch {
		vals = append(vals, tup.PolicyID, tup.HostID, tup.Passes)
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		return ctxerr.Wrap(ctx, err, "insert into policy_membership")
	})
}

// AsyncBatchUpdatePolicyTimestamp updates the hosts' policy_updated_at timestamp
// for the batch of host ids provided.
func (ds *Datastore) AsyncBatchUpdatePolicyTimestamp(ctx context.Context, ids []uint, ts time.Time) error {
	// NOTE: this is tested via the server/service/async package tests.

	sql := `
		UPDATE
		  hosts
		SET
		  policy_updated_at = ?
		WHERE
		  id IN (?)`
	query, args, err := sqlx.In(sql, ts, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to update hosts.policy_updated_at")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return ctxerr.Wrap(ctx, err, "update hosts.policy_updated_at")
	})
}

func deleteAllPolicyMemberships(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	query, args, err := sqlx.In(`DELETE FROM policy_membership WHERE host_id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to delete policies")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec delete policies")
	}
	return nil
}

func cleanupPolicyMembershipOnTeamChange(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	// hosts can only be in one team, so if there's a policy that has a team id and a result from one of our hosts
	// it can only be from the previous team they are being transferred from
	query, args, err := sqlx.In(`DELETE FROM policy_membership
					WHERE policy_id IN (SELECT id FROM policies WHERE team_id IS NOT NULL) AND host_id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clean old policy memberships sqlx in")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec clean old policy memberships")
	}
	return nil
}

func cleanupPolicyMembershipOnPolicyUpdate(ctx context.Context, db sqlx.ExecerContext, policyID uint, platforms string) error {
	if platforms == "" {
		// all platforms allowed, nothing to clean up
		return nil
	}

	delStmt := `
	DELETE
	  pm
	FROM
	  policy_membership pm
	LEFT JOIN
	  hosts h
	ON
	  pm.host_id = h.id
	WHERE
	  pm.policy_id = ? AND
	  ( h.id IS NULL OR
		FIND_IN_SET(h.platform, ?) = 0 )`

	var expandedPlatforms []string
	splitPlatforms := strings.Split(platforms, ",")
	for _, platform := range splitPlatforms {
		expandedPlatforms = append(expandedPlatforms, fleet.ExpandPlatform(strings.TrimSpace(platform))...)
	}
	_, err := db.ExecContext(ctx, delStmt, policyID, strings.Join(expandedPlatforms, ","))
	return ctxerr.Wrap(ctx, err, "cleanup policy membership")
}

// cleanupPolicyMembership is similar to cleanupPolicyMembershipOnPolicyUpdate but without the platform constraints.
// Used when we want to remove all policy membership.
func (ds *Datastore) cleanupPolicyMembershipForPolicy(ctx context.Context, policyID uint) error {
	// delete all policy memberships for the policy
	delStmt := `
		DELETE
			pm
		FROM
			policy_membership pm
		LEFT JOIN
			hosts h
		ON
			pm.host_id = h.id
		WHERE
			pm.policy_id = ?
	`

	_, err := ds.writer(ctx).ExecContext(ctx, delStmt, policyID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup policy membership")
	}

	// delete all policy stats for the policy
	// wrapping in a retry to avoid deadlocks with the cleanups_then_aggregation cron job
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM policy_stats WHERE policy_id = ?`, policyID)
		return err
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup policy stats")
	}

	return nil
}

// CleanupPolicyMembership deletes the host's membership from policies that
// have been updated recently if those hosts don't meet the policy's criteria
// anymore (e.g. if the policy's platforms has been updated from "any" - the
// empty string - to "windows", this would delete that policy's membership rows
// for any non-windows host).
func (ds *Datastore) CleanupPolicyMembership(ctx context.Context, now time.Time) error {
	const (
		recentlyUpdatedPoliciesInterval = 24 * time.Hour

		// Using `p.created_at < p.updated.at` to ignore newly created.
		updatedPoliciesStmt = `
			SELECT
				p.id,
				p.platforms
			FROM
				policies p
			WHERE
				p.updated_at >= DATE_SUB(?, INTERVAL ? SECOND) AND
				p.created_at < p.updated_at`

		deleteMembershipStmt = `
			DELETE
				pm
			FROM
				policy_membership pm
			INNER JOIN
				hosts h
			ON
				pm.host_id = h.id
			WHERE
				pm.policy_id = ? AND
				FIND_IN_SET(h.platform, ?) = 0`
	)

	var pols []*fleet.Policy
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &pols, updatedPoliciesStmt, now, int(recentlyUpdatedPoliciesInterval.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "select recently updated policies")
	}

	for _, pol := range pols {
		if pol.Platform == "" {
			continue
		}

		var expandedPlatforms []string
		splitPlatforms := strings.Split(pol.Platform, ",")
		for _, platform := range splitPlatforms {
			expandedPlatforms = append(expandedPlatforms, fleet.ExpandPlatform(strings.TrimSpace(platform))...)
		}

		if _, err := ds.writer(ctx).ExecContext(ctx, deleteMembershipStmt, pol.ID, strings.Join(expandedPlatforms, ",")); err != nil {
			return ctxerr.Wrapf(ctx, err, "delete outdated hosts membership for policy: %d; platforms: %v", pol.ID, expandedPlatforms)
		}
	}

	return nil
}

func (ds *Datastore) UpdatePolicyFailureCountsForHosts(ctx context.Context, hosts []*fleet.Host) ([]*fleet.Host, error) {
	if len(hosts) == 0 {
		return hosts, nil
	}

	// Get policy failure counts for each host
	hostIDs := make([]uint, 0, len(hosts))

	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	query, args, err := sqlx.In(`
		SELECT
			pm.host_id,
			COUNT(*) AS failing_policy_count
		FROM
			policy_membership pm
		WHERE
			pm.passes = 0 AND
			pm.host_id IN (?)
		GROUP BY
			pm.host_id
	`, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build policy failure count query")
	}

	var policyFailureCounts []struct {
		HostID             uint `db:"host_id"`
		FailingPolicyCount int  `db:"failing_policy_count"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policyFailureCounts, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policy failure counts for hosts")
	}

	// Map policy failure counts to hosts
	hostIDToPolicyFailureCounts := make(map[uint]int)
	for _, policyFailureCount := range policyFailureCounts {
		hostIDToPolicyFailureCounts[policyFailureCount.HostID] = policyFailureCount.FailingPolicyCount
	}

	for _, host := range hosts {
		host.TotalIssuesCount = hostIDToPolicyFailureCounts[host.ID]
		host.FailingPoliciesCount = hostIDToPolicyFailureCounts[host.ID]
	}

	return hosts, nil
}

// PolicyViolationDays is a structure used for aggregate counts of policy violation days.
type PolicyViolationDays struct {
	// FailingHostCount is an aggregate count of actual policy violations days. One actual policy
	// violation day is added for each policy that a host is failing at the time of the count.
	FailingHostCount uint `json:"failing_host_count" db:"failing_host_count"`
	// TotalHostCount is an aggregate count of possible policy violations days. One possible policy
	// violation day is added for each policy that a host is a member of at the time of the count.
	TotalHostCount uint `json:"total_host_count" db:"total_host_count"`
}

func (ds *Datastore) IncrementPolicyViolationDays(ctx context.Context) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return incrementViolationDaysDB(ctx, tx)
	})
}

func (ds *Datastore) IncreasePolicyAutomationIteration(ctx context.Context, policyID uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO policy_automation_iterations (policy_id, iteration) VALUES (?,1)
			ON DUPLICATE KEY UPDATE iteration = iteration + 1;
		`, policyID)
		return err
	})
}

// OutdatedAutomationBatch returns a batch of hosts that had a failing policy.
func (ds *Datastore) OutdatedAutomationBatch(ctx context.Context) ([]fleet.PolicyFailure, error) {
	var failures []fleet.PolicyFailure
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		failures = failures[:0] // In case of retry (from withRetryTxx) empty the list of failures.
		var hostIDs []uint

		rows, err := tx.QueryContext(ctx, `
			SELECT ai.policy_id, pm.host_id, h.hostname, h.computer_name
				FROM policy_automation_iterations ai
				JOIN policy_membership pm ON pm.policy_id = ai.policy_id
					AND (pm.automation_iteration < ai.iteration
						   OR pm.automation_iteration IS NULL)
				JOIN hosts h ON pm.host_id = h.id
				WHERE NOT pm.passes
				LIMIT 1000
				FOR UPDATE;
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var f fleet.PolicyFailure
			if err := rows.Scan(&f.PolicyID, &f.Host.ID, &f.Host.Hostname, &f.Host.DisplayName); err != nil {
				return err
			}
			failures = append(failures, f)
			hostIDs = append(hostIDs, f.Host.ID)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(hostIDs) == 0 {
			return nil
		}
		query := `
			UPDATE policy_membership pm SET pm.automation_iteration = (
				SELECT ai.iteration
				FROM policy_automation_iterations ai
				WHERE pm.policy_id = ai.policy_id
		   ) WHERE pm.host_id IN (?);`
		query, args, err := sqlx.In(query, hostIDs)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return failures, nil
}

func incrementViolationDaysDB(ctx context.Context, tx sqlx.ExtContext) error {
	const (
		statsID        = 0
		globalStats    = true
		statsType      = aggregatedStatsTypePolicyViolationsDays
		updateInterval = 24 * time.Hour
	)

	var prevFailing uint
	var prevTotal uint
	var shouldIncrement bool

	// get current count of policy violation days from `aggregated_stats``
	selectStmt := `
		SELECT
			json_value,
			created_at,
			updated_at
		FROM
			aggregated_stats
		WHERE
			id = ? AND global_stats = ? AND type = ?`
	dest := struct {
		CreatedAt time.Time       `json:"created_at" db:"created_at"`
		UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
		StatsJSON json.RawMessage `json:"json_value" db:"json_value"`
	}{}

	err := sqlx.GetContext(ctx, tx, &dest, selectStmt, statsID, globalStats, statsType)
	switch {
	case err == sql.ErrNoRows:
		// no previous counts exists so initialize counts as zero and proceed to increment
		prevFailing = 0
		prevTotal = 0
		shouldIncrement = true
	case err != nil:
		return ctxerr.Wrap(ctx, err, "selecting policy violation days aggregated stats")
	default:
		// increment previous counts if interval has elapsed
		var prevStats PolicyViolationDays
		if err := json.Unmarshal(dest.StatsJSON, &prevStats); err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshal policy violation counts")
		}
		prevFailing = prevStats.FailingHostCount
		prevTotal = prevStats.TotalHostCount
		shouldIncrement = time.Now().After(dest.UpdatedAt.Add(updateInterval))
	}

	if !shouldIncrement {
		return nil
	}

	// increment count of policy violation days by total number of failing records from
	// `policy_membership`
	var newCounts PolicyViolationDays
	if err := sqlx.GetContext(ctx, tx, &newCounts, `
		 SELECT	(select count(*) from policy_membership where passes=0) as failing_host_count,
	   		(select count(*) from policy_membership) as total_host_count`,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "count policy violation days")
	}
	newCounts.FailingHostCount = prevFailing + newCounts.FailingHostCount
	newCounts.TotalHostCount = prevTotal + newCounts.TotalHostCount
	statsJSON, err := json.Marshal(newCounts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal policy violation counts")
	}

	// upsert `aggregated_stats` with new count
	upsertStmt := `
		INSERT INTO
			aggregated_stats (id, global_stats, type, json_value)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			json_value = VALUES(json_value)`
	if _, err := tx.ExecContext(ctx, upsertStmt, statsID, globalStats, statsType, statsJSON); err != nil {
		return ctxerr.Wrap(ctx, err, "update policy violation days aggregated stats")
	}

	return nil
}

func (ds *Datastore) InitializePolicyViolationDays(ctx context.Context) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return initializePolicyViolationDaysDB(ctx, tx)
	})
}

func initializePolicyViolationDaysDB(ctx context.Context, tx sqlx.ExtContext) error {
	const (
		statsID     = 0
		globalStats = true
		statsType   = aggregatedStatsTypePolicyViolationsDays
	)

	statsJSON, err := json.Marshal(PolicyViolationDays{})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal policy violation counts")
	}

	stmt := `
		INSERT INTO
			aggregated_stats (id, global_stats, type, json_value)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			json_value = VALUES(json_value),
			created_at = CURRENT_TIMESTAMP`
	if _, err := tx.ExecContext(ctx, stmt, statsID, globalStats, statsType, statsJSON); err != nil {
		return ctxerr.Wrap(ctx, err, "initialize policy violation days aggregated stats")
	}

	return nil
}

func amountPolicyViolationDaysDB(ctx context.Context, tx sqlx.QueryerContext) (int, int, error) {
	const (
		statsID     = 0
		globalStats = true
		statsType   = aggregatedStatsTypePolicyViolationsDays
	)
	var statsJSON json.RawMessage
	if err := sqlx.GetContext(ctx, tx, &statsJSON, `
		SELECT
			json_value
		FROM
			aggregated_stats
		WHERE
			id = ? AND global_stats = ? AND type = ?
	`, statsID, globalStats, statsType); err != nil {
		return 0, 0, err
	}

	var counts PolicyViolationDays
	if err := json.Unmarshal(statsJSON, &counts); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "unmarshal policy violation counts")
	}

	return int(counts.FailingHostCount), int(counts.TotalHostCount), nil
}

func (ds *Datastore) UpdateHostPolicyCounts(ctx context.Context) error {
	// NOTE these queries are duplicated in the below migration.  Updates
	// to these queries should be reflected there as well.
	// https://github.com/fleetdm/fleet/blob/main/server/datastore/mysql/migrations/tables/20231215122713_InsertPolicyStatsData.go#L12

	// Update Counts for Inherited Global Policies for each Team
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		SELECT
			p.id,
			t.id AS inherited_team_id,
			(
				SELECT COUNT(*) 
				FROM policy_membership pm 
				INNER JOIN hosts h ON pm.host_id = h.id 
				WHERE pm.policy_id = p.id AND pm.passes = true AND h.team_id = t.id
			) AS passing_host_count,
			(
				SELECT COUNT(*) 
				FROM policy_membership pm 
				INNER JOIN hosts h ON pm.host_id = h.id 
				WHERE pm.policy_id = p.id AND pm.passes = false AND h.team_id = t.id
			) AS failing_host_count
		FROM policies p
		CROSS JOIN teams t
		WHERE p.team_id IS NULL
		GROUP BY p.id, t.id
		ON DUPLICATE KEY UPDATE 
			updated_at = NOW(),
			passing_host_count = VALUES(passing_host_count),
			failing_host_count = VALUES(failing_host_count);
    `)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host policy counts for inherited global policies")
	}

	// Update Counts for Global and Team Policies
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		SELECT
			p.id,
			0 AS inherited_team_id, -- using 0 to represent global scope
			COALESCE(SUM(IF(pm.passes IS NULL, 0, pm.passes = 1)), 0), 
			COALESCE(SUM(IF(pm.passes IS NULL, 0, pm.passes = 0)), 0)
		FROM policies p
		LEFT JOIN policy_membership pm ON p.id = pm.policy_id
		GROUP BY p.id
		ON DUPLICATE KEY UPDATE 
			updated_at = NOW(),
			passing_host_count = VALUES(passing_host_count),
			failing_host_count = VALUES(failing_host_count);
    `)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host policy counts for global and team policies")
	}

	return nil
}
