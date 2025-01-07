package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

const policyCols = `
	p.id, p.team_id, p.resolution, p.name, p.query, p.description,
	p.author_id, p.platforms, p.created_at, p.updated_at, p.critical,
	p.calendar_events_enabled, p.software_installer_id, p.script_id
`

var (
	errSoftwareTitleIDOnGlobalPolicy = errors.New("install software title id can be only be set on team policies")
	errScriptIDOnGlobalPolicy        = errors.New("run script id can only be set on team or \"no team\" policies")
)

var policySearchColumns = []string{"p.name"}

func (ds *Datastore) NewGlobalPolicy(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
	if args.SoftwareInstallerID != nil {
		return nil, ctxerr.Wrap(ctx, errSoftwareTitleIDOnGlobalPolicy, "create policy")
	}
	if args.ScriptID != nil {
		return nil, ctxerr.Wrap(ctx, errScriptIDOnGlobalPolicy, "create policy")
	}
	if args.QueryID != nil {
		q, err := ds.Query(ctx, *args.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetching query from id")
		}
		args.Name = q.Name
		args.Query = q.Query
		args.Description = q.Description
	}
	// We must normalize the name for full Unicode support (Unicode equivalence).
	nameUnicode := norm.NFC.String(args.Name)
	res, err := ds.writer(ctx).ExecContext(ctx,
		fmt.Sprintf(
			`INSERT INTO policies (name, query, description, resolution, author_id, platforms, critical, checksum) VALUES (?, ?, ?, ?, ?, ?, ?, %s)`,
			policiesChecksumComputedColumn(),
		),
		nameUnicode, args.Query, args.Description, args.Resolution, authorID, args.Platform, args.Critical,
	)
	switch {
	case err == nil:
		// OK
	case IsDuplicate(err):
		return nil, ctxerr.Wrap(ctx, alreadyExists("Policy", nameUnicode))
	default:
		return nil, ctxerr.Wrap(ctx, err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last id after inserting policy")
	}
	return policyDB(ctx, ds.writer(ctx), uint(lastIdInt64), nil) //nolint:gosec // dismiss G115
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
		AND ((p.team_id IS NULL AND ps.inherited_team_id IS NULL) OR (p.team_id IS NOT NULL))
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

func (ds *Datastore) PolicyLite(ctx context.Context, id uint) (*fleet.PolicyLite, error) {
	var policy fleet.PolicyLite
	err := sqlx.GetContext(
		ctx, ds.reader(ctx), &policy,
		`SELECT id, name, description, resolution FROM policies WHERE id=?`, id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("Policy").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting policy")
	}
	return &policy, nil
}

// SavePolicy updates some fields of the given policy on the datastore.
//
// Currently, SavePolicy does not allow updating the team of an existing policy.
func (ds *Datastore) SavePolicy(ctx context.Context, p *fleet.Policy, shouldRemoveAllPolicyMemberships bool, removePolicyStats bool) error {
	if p.TeamID == nil && p.SoftwareInstallerID != nil {
		return ctxerr.Wrap(ctx, errSoftwareTitleIDOnGlobalPolicy, "save policy")
	}
	if p.TeamID == nil && p.ScriptID != nil {
		return ctxerr.Wrap(ctx, errScriptIDOnGlobalPolicy, "save policy")
	}

	if p.TeamID != nil {
		if err := assertTeamMatches(ctx, ds.writer(ctx), *p.TeamID, p.SoftwareInstallerID, p.ScriptID); err != nil {
			return ctxerr.Wrap(ctx, err, "save policy")
		}
	}

	// We must normalize the name for full Unicode support (Unicode equivalence).
	p.Name = norm.NFC.String(p.Name)
	sql := `
		UPDATE policies
			SET name = ?, query = ?, description = ?, resolution = ?, platforms = ?, critical = ?, calendar_events_enabled = ?, software_installer_id = ?, script_id = ?, checksum = ` + policiesChecksumComputedColumn() + `
			WHERE id = ?
	`
	result, err := ds.writer(ctx).ExecContext(
		ctx, sql, p.Name, p.Query, p.Description, p.Resolution, p.Platform, p.Critical, p.CalendarEventsEnabled, p.SoftwareInstallerID, p.ScriptID, p.ID,
	)
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

	return cleanupPolicy(
		ctx, ds.reader(ctx), ds.writer(ctx), p.ID, p.Platform, shouldRemoveAllPolicyMemberships, removePolicyStats, ds.logger,
	)
}

var (
	errMismatchedInstallerTeam = &fleet.BadRequestError{Message: "software installer is associated with a different team"}
	errMismatchedScriptTeam    = &fleet.BadRequestError{Message: "script is associated with a different team"}
)

func assertTeamMatches(ctx context.Context, db sqlx.QueryerContext, teamID uint, softwareInstallerID *uint, scriptID *uint) error {
	if softwareInstallerID != nil {
		var softwareInstallerTeamID uint
		err := sqlx.GetContext(ctx, db, &softwareInstallerTeamID, "SELECT global_or_team_id FROM software_installers WHERE id = ?", softwareInstallerID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &fleet.BadRequestError{Message: "A software installer with the supplied ID does not exist"}
			}
			return err
		} else if softwareInstallerTeamID != teamID {
			return errMismatchedInstallerTeam
		}
	}

	if scriptID != nil {
		var scriptTeamID uint
		err := sqlx.GetContext(ctx, db, &scriptTeamID, "SELECT global_or_team_id FROM scripts WHERE id = ?", scriptID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &fleet.BadRequestError{Message: "A script with the supplied ID does not exist"}
			}
			return err
		} else if scriptTeamID != teamID {
			return errMismatchedScriptTeam
		}
	}

	return nil
}

func cleanupPolicy(
	ctx context.Context, queryerContext sqlx.QueryerContext, extContext sqlx.ExtContext, policyID uint, policyPlatform string,
	shouldRemoveAllPolicyMemberships bool,
	removePolicyStats bool, logger kitlog.Logger,
) error {
	var err error
	if shouldRemoveAllPolicyMemberships {
		err = cleanupPolicyMembershipForPolicy(ctx, queryerContext, extContext, policyID)
	} else {
		err = cleanupPolicyMembershipOnPolicyUpdate(ctx, queryerContext, extContext, policyID, policyPlatform)
	}
	if err != nil {
		return err
	}
	if removePolicyStats {
		// delete all policy stats for the policy
		fn := func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, `DELETE FROM policy_stats WHERE policy_id = ?`, policyID)
			return err
		}
		if _, isDB := extContext.(*sqlx.DB); isDB {
			// wrapping in a retry to avoid deadlocks with the cleanups_then_aggregation cron job
			err = common_mysql.WithRetryTxx(ctx, extContext.(*sqlx.DB), fn, logger)
		} else {
			err = fn(extContext)
		}
		if err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup policy stats")
		}
	}
	return nil
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

	// ds.UpdateHostIssuesFailingPolicies should be executed even if len(results) == 0
	// because this means the host is configured to run no policies and we would like
	// to cleanup the counts (if any).
	if err := ds.UpdateHostIssuesFailingPolicies(ctx, []uint{host.ID}); err != nil {
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

func (ds *Datastore) ClearAutoInstallPolicyStatusForHosts(ctx context.Context, installerID uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	stmt := `
UPDATE
	policies p
	JOIN policy_membership pm ON pm.policy_id = p.id
SET
	passes = NULL
WHERE
	p.software_installer_id = ?
	AND pm.host_id IN (?)
		`

	stmt, args, err := sqlx.In(stmt, installerID, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement for clearing auto install policy status")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "clearing auto install policy status")
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
		LEFT JOIN policy_stats ps ON p.id = ps.policy_id AND ps.inherited_team_id IS NULL
	`

	if teamID != nil {
		query += " WHERE team_id = ?"
		args = append(args, *teamID)
	} else {
		query += " WHERE team_id IS NULL"
	}

	// We must normalize the name for full Unicode support (Unicode equivalence).
	match := norm.NFC.String(opts.MatchQuery)
	query, args = searchLike(query, args, match, policySearchColumns...)
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
func getInheritedPoliciesForTeam(ctx context.Context, q sqlx.QueryerContext, teamID uint, opts fleet.ListOptions) ([]*fleet.Policy, error) {
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

	args = append(args, teamID)

	// We must normalize the name for full Unicode support (Unicode equivalence).
	match := norm.NFC.String(opts.MatchQuery)
	query, args = searchLike(query, args, match, policySearchColumns...)
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

	// We must normalize the name for full Unicode support (Unicode equivalence).
	match := norm.NFC.String(matchQuery)
	query, args = searchLike(query, args, match, policySearchColumns...)

	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting policies")
	}

	return count, nil
}

func (ds *Datastore) CountMergedTeamPolicies(ctx context.Context, teamID uint, matchQuery string) (int, error) {
	var args []interface{}

	query := `SELECT count(*) FROM policies p WHERE (p.team_id = ? OR p.team_id IS NULL)`
	args = append(args, teamID)

	// We must normalize the name for full Unicode support (Unicode equivalence).
	match := norm.NFC.String(matchQuery)
	query, args = searchLike(query, args, match, policySearchColumns...)

	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting merged team policies")
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
	  	AND ((p.team_id IS NULL AND ps.inherited_team_id IS NULL) OR (p.team_id IS NOT NULL))
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
	for _, id := range ids {
		if err := ds.deletePendingSoftwareInstallsForPolicy(ctx, nil, id); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending software installs for policy")
		}
		if err := ds.deletePendingHostScriptExecutionsForPolicy(ctx, nil, id); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending host script executions for policy")
		}
	}

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
	if host.FleetPlatform() == "" {
		// We log to help troubleshooting in case this happens, as the host
		// won't be receiving any policies targeted for specific platforms.
		level.Error(ds.logger).Log("err", "unrecognized platform", "hostID", host.ID, "platform", host.Platform) //nolint:errcheck
	}
	const stmt = `
		SELECT id, query
		FROM policies
		WHERE
			-- team_id == NULL are global policies that apply to all hosts
			-- team_id == 0 are policies that apply to hosts in "No team"
			-- team_id > 0 are policies that apply to hosts in teams
			(team_id IS NULL OR team_id = COALESCE(?, 0)) AND
			(platforms = '' OR FIND_IN_SET(?, platforms))`
	var rows []struct {
		ID    string `db:"id"`
		Query string `db:"query"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, host.TeamID, host.FleetPlatform()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting policies for host")
	}
	results := make(map[string]string)
	for _, row := range rows {
		results[row.ID] = row.Query
	}
	return results, nil
}

func (ds *Datastore) NewTeamPolicy(ctx context.Context, teamID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
	return newTeamPolicy(ctx, ds.writer(ctx), teamID, authorID, args)
}

func newTeamPolicy(ctx context.Context, db sqlx.ExtContext, teamID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
	if args.QueryID != nil {
		q, err := query(ctx, db, *args.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetching query from id")
		}
		args.Name = q.Name
		args.Query = q.Query
		args.Description = q.Description
	}
	// Check team exists.
	if teamID > 0 {
		var ok bool
		err := sqlx.GetContext(ctx, db, &ok, `SELECT COUNT(*) = 1 FROM teams WHERE id = ?`, teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team id")
		}
		if !ok {
			return nil, ctxerr.Wrap(ctx, notFound("Team").WithID(teamID), "get team id")
		}

	}
	// We must normalize the name for full Unicode support (Unicode equivalence).
	nameUnicode := norm.NFC.String(args.Name)

	if err := assertTeamMatches(ctx, db, teamID, args.SoftwareInstallerID, args.ScriptID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create team policy")
	}

	res, err := db.ExecContext(ctx,
		fmt.Sprintf(
			`INSERT INTO policies (name, query, description, team_id, resolution, author_id, platforms, critical, calendar_events_enabled, software_installer_id, script_id, checksum) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s)`,
			policiesChecksumComputedColumn(),
		),
		nameUnicode, args.Query, args.Description, teamID, args.Resolution, authorID, args.Platform, args.Critical,
		args.CalendarEventsEnabled, args.SoftwareInstallerID, args.ScriptID,
	)
	switch {
	case err == nil:
		// OK
	case IsDuplicate(err):
		return nil, ctxerr.Wrap(ctx, alreadyExists("Policy", nameUnicode))
	default:
		return nil, ctxerr.Wrap(ctx, err, "inserting new policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last id after inserting policy")
	}
	return policyDB(ctx, db, uint(lastIdInt64), &teamID) //nolint:gosec // dismiss G115
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

func (ds *Datastore) ListMergedTeamPolicies(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.Policy, error) {
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
		LEFT JOIN policy_stats ps ON p.id = ps.policy_id
		AND (p.team_id IS NOT NULL OR ps.inherited_team_id = ?)
		WHERE (p.team_id = ? OR p.team_id IS NULL)
    `

	args = append(args, teamID, teamID)

	// We must normalize the name for full Unicode support (Unicode equivalence).
	match := norm.NFC.String(opts.MatchQuery)
	query, args = searchLike(query, args, match, policySearchColumns...)
	query, _ = appendListOptionsToSQL(query, &opts)

	var policies []*fleet.Policy
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing merged team policies")
	}

	return policies, nil
}

func (ds *Datastore) DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
	for _, id := range ids {
		if err := ds.deletePendingSoftwareInstallsForPolicy(ctx, &teamID, id); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending software installs for policy")
		}
		if err := ds.deletePendingHostScriptExecutionsForPolicy(ctx, &teamID, id); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending host script executions for policy")
		}
	}

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
// Currently, ApplyPolicySpecs does not allow updating the team of an existing policy.
func (ds *Datastore) ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
	// Use the same DB for all operations in this method for performance
	queryerContext := ds.writer(ctx)

	// Preprocess specs and group them by team
	teamNameToID := make(map[string]*uint, 1)
	teamIDToPolicies := make(map[*uint][]*fleet.PolicySpec, 1)
	softwareInstallerIDs := make(map[*uint]map[uint]*uint) // teamID -> titleID -> softwareInstallerID

	// Get the team IDs
	for _, spec := range specs {
		// We must normalize the name for full Unicode support (Unicode equivalence).
		spec.Name = norm.NFC.String(spec.Name)
		spec.Team = norm.NFC.String(spec.Team)
		teamID, ok := teamNameToID[spec.Team]
		if !ok {
			if spec.Team != "" {
				if spec.Team == "No team" {
					teamID = ptr.Uint(0)
				} else {
					var tmID uint
					err := sqlx.GetContext(ctx, queryerContext, &tmID, `SELECT id FROM teams WHERE name = ?`, spec.Team)
					if err != nil {
						if errors.Is(err, sql.ErrNoRows) {
							return ctxerr.Wrap(ctx, notFound("Team").WithName(spec.Team), "get team id")
						}
						return ctxerr.Wrap(ctx, err, "get team id")
					}
					teamID = &tmID
				}
			}
			teamNameToID[spec.Team] = teamID
		}
		teamIDToPolicies[teamID] = append(teamIDToPolicies[teamID], spec)
	}

	// Get software installer ids from software title ids.
	for _, spec := range specs {
		if spec.SoftwareTitleID == nil || *spec.SoftwareTitleID == 0 {
			continue
		}
		if spec.Team == "" {
			return ctxerr.Wrap(ctx, errSoftwareTitleIDOnGlobalPolicy, "create policy from spec")
		}
		var softwareInstallerID uint
		err := sqlx.GetContext(ctx, queryerContext, &softwareInstallerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id = ?`,
			teamNameToID[spec.Team], spec.SoftwareTitleID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("SoftwareInstaller").WithID(*spec.SoftwareTitleID), "get software installer id")
			}
			return ctxerr.Wrap(ctx, err, "get software installer id")
		}
		if len(softwareInstallerIDs[teamNameToID[spec.Team]]) == 0 {
			softwareInstallerIDs[teamNameToID[spec.Team]] = make(map[uint]*uint)
		}
		softwareInstallerIDs[teamNameToID[spec.Team]][*spec.SoftwareTitleID] = &softwareInstallerID
	}

	// Get the query and platforms of the current policies so that we can check if query or platform changed later, if needed
	type policyLite struct {
		Name                string `db:"name"`
		Query               string `db:"query"`
		Platforms           string `db:"platforms"`
		SoftwareInstallerID *uint  `db:"software_installer_id"`
		ScriptID            *uint  `db:"script_id"`
	}
	teamIDToPoliciesByName := make(map[*uint]map[string]policyLite, len(teamIDToPolicies))
	for teamID, teamPolicySpecs := range teamIDToPolicies {
		teamIDToPoliciesByName[teamID] = make(map[string]policyLite, len(teamPolicySpecs))
		policyNames := make([]string, 0, len(teamPolicySpecs))
		for _, spec := range teamPolicySpecs {
			policyNames = append(policyNames, spec.Name)
		}

		var query string
		var args []interface{}
		var err error
		if teamID == nil {
			query, args, err = sqlx.In("SELECT name, query, platforms, software_installer_id, script_id FROM policies WHERE team_id IS NULL AND name IN (?)", policyNames)
		} else {
			query, args, err = sqlx.In(
				"SELECT name, query, platforms, software_installer_id, script_id FROM policies WHERE team_id = ? AND name IN (?)", *teamID, policyNames,
			)
		}
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building query to get policies by name")
		}
		policies := make([]policyLite, 0, len(teamPolicySpecs))
		err = sqlx.SelectContext(ctx, queryerContext, &policies, query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting policies by name")
		}
		for _, p := range policies {
			teamIDToPoliciesByName[teamID][p.Name] = p
		}
	}

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
			calendar_events_enabled,
			software_installer_id,
		    script_id,
			checksum
		) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s)
		ON DUPLICATE KEY UPDATE
			query = VALUES(query),
			description = VALUES(description),
			author_id = VALUES(author_id),
			resolution = VALUES(resolution),
			platforms = VALUES(platforms),
			critical = VALUES(critical),
			calendar_events_enabled = VALUES(calendar_events_enabled),
			software_installer_id = VALUES(software_installer_id),
			script_id = VALUES(script_id)
		`, policiesChecksumComputedColumn(),
		)
		for teamID, teamPolicySpecs := range teamIDToPolicies {
			for _, spec := range teamPolicySpecs {
				var softwareInstallerID *uint
				if spec.SoftwareTitleID != nil {
					softwareInstallerID = softwareInstallerIDs[teamNameToID[spec.Team]][*spec.SoftwareTitleID]
				}
				scriptID := spec.ScriptID
				if spec.ScriptID != nil && *spec.ScriptID == 0 {
					scriptID = nil
				}

				res, err := tx.ExecContext(
					ctx,
					query, spec.Name, spec.Query, spec.Description, authorID, spec.Resolution, teamID, spec.Platform, spec.Critical,
					spec.CalendarEventsEnabled, softwareInstallerID, scriptID,
				)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "exec ApplyPolicySpecs insert")
				}

				if insertOnDuplicateDidInsertOrUpdate(res) {
					// when the upsert results in an UPDATE that *did* change some values,
					// it returns the updated ID as last inserted id.
					if lastID, _ := res.LastInsertId(); lastID > 0 {
						var (
							shouldRemoveAllPolicyMemberships bool
							removePolicyStats                bool
						)
						// Figure out if the query, platform or software installer changed.
						var softwareInstallerID *uint
						if spec.SoftwareTitleID != nil {
							softwareInstallerID = softwareInstallerIDs[teamID][*spec.SoftwareTitleID]
						}
						if prev, ok := teamIDToPoliciesByName[teamID][spec.Name]; ok {
							switch {
							case prev.Query != spec.Query:
								shouldRemoveAllPolicyMemberships = true
								removePolicyStats = true
							case teamID != nil &&
								((prev.SoftwareInstallerID == nil && spec.SoftwareTitleID != nil) ||
									(prev.SoftwareInstallerID != nil && softwareInstallerID != nil && *prev.SoftwareInstallerID != *softwareInstallerID)):
								shouldRemoveAllPolicyMemberships = true
								removePolicyStats = true
							case teamID != nil &&
								((prev.ScriptID == nil && spec.ScriptID != nil) ||
									(prev.ScriptID != nil && spec.ScriptID != nil && *prev.ScriptID != *spec.ScriptID)):
								shouldRemoveAllPolicyMemberships = true
								removePolicyStats = true
							case prev.Platforms != spec.Platform:
								removePolicyStats = true
							}
						}
						if err = cleanupPolicy(
							ctx, tx, tx, uint(lastID), spec.Platform, shouldRemoveAllPolicyMemberships, //nolint:gosec // dismiss G115
							removePolicyStats, ds.logger,
						); err != nil {
							return err
						}
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
	hostIDs := make([]uint, 0, len(batch))
	for _, tup := range batch {
		vals = append(vals, tup.PolicyID, tup.HostID, tup.Passes)
		hostIDs = append(hostIDs, tup.HostID)
	}
	err := ds.withRetryTxx(
		ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, sql, vals...)
			return ctxerr.Wrap(ctx, err, "insert into policy_membership")
		})
	if err != nil {
		return err
	}
	if err = ds.UpdateHostIssuesFailingPolicies(ctx, hostIDs); err != nil {
		return err
	}
	return nil
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
	// This method is currently only called for 1 host at a time, so it is not a performance concern.
	if err = updateHostIssuesFailingPolicies(ctx, tx, hostIDs); err != nil {
		return err
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
	// This method is currently called for a batch of hosts. Performance should be monitored. If performance becomes a concern,
	// we can reduce batch size or move this method outside the transaction.
	if err = updateHostIssuesFailingPolicies(ctx, tx, hostIDs); err != nil {
		return err
	}
	return nil
}

func cleanupQueryResultsOnTeamChange(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	// Similar to cleanupPolicyMembershipOnTeamChange, hosts can belong to one team only, so we just delete all
	// the query results of the hosts that belong to queries that are not global.
	const cleanupQuery = `
		DELETE FROM query_results
		WHERE query_id IN (SELECT id FROM queries WHERE team_id IS NOT NULL) AND host_id IN (?)`
	query, args, err := sqlx.In(cleanupQuery, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build cleanup query results query")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec cleanup query results query")
	}
	return nil
}

func cleanupPolicyMembershipOnPolicyUpdate(
	ctx context.Context, queryerContext sqlx.QueryerContext, db sqlx.ExecerContext, policyID uint, platforms string,
) error {
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

	selectStmt := `
	SELECT DISTINCT
	  h.id
	FROM
	  policy_membership pm
	INNER JOIN
	  hosts h
	ON
	  pm.host_id = h.id
	WHERE
	  pm.policy_id = ? AND
	  FIND_IN_SET(h.platform, ?) = 0`

	var expandedPlatforms []string
	splitPlatforms := strings.Split(platforms, ",")
	for _, platform := range splitPlatforms {
		expandedPlatforms = append(expandedPlatforms, fleet.ExpandPlatform(strings.TrimSpace(platform))...)
	}

	// Find the impacted host IDs, so we can update their host issues entries
	var hostIDs []uint
	err := sqlx.SelectContext(ctx, queryerContext, &hostIDs, selectStmt, policyID, strings.Join(expandedPlatforms, ","))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "select hosts to cleanup policy membership")
	}

	_, err = db.ExecContext(ctx, delStmt, policyID, strings.Join(expandedPlatforms, ","))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup policy membership")
	}

	// Update host issues entries. This method is rarely called, so performance should not be a concern.
	if err = updateHostIssuesFailingPolicies(ctx, db, hostIDs); err != nil {
		return err
	}

	return nil
}

// cleanupPolicyMembership is similar to cleanupPolicyMembershipOnPolicyUpdate but without the platform constraints.
// Used when we want to remove all policy membership.
func cleanupPolicyMembershipForPolicy(
	ctx context.Context, queryerContext sqlx.QueryerContext, exec sqlx.ExecerContext, policyID uint,
) error {
	selectStmt := `
	SELECT DISTINCT
	  h.id
	FROM
	  policy_membership pm
	INNER JOIN
	  hosts h
	ON
	  pm.host_id = h.id
	WHERE
	  pm.policy_id = ?`

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

	// Find the impacted host IDs, so we can update their host issues entries
	var hostIDs []uint
	err := sqlx.SelectContext(ctx, queryerContext, &hostIDs, selectStmt, policyID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "select hosts to cleanup policy membership for policy")
	}

	_, err = exec.ExecContext(ctx, delStmt, policyID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup policy membership")
	}

	// Update host issues entries. This method is rarely called, so performance should not be a concern.
	if err = updateHostIssuesFailingPolicies(ctx, exec, hostIDs); err != nil {
		return err
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
	)

	var pols []*fleet.Policy
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &pols, updatedPoliciesStmt, now, int(recentlyUpdatedPoliciesInterval.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "select recently updated policies")
	}

	for _, pol := range pols {
		if err := cleanupPolicyMembershipOnPolicyUpdate(ctx, ds.reader(ctx), ds.writer(ctx), pol.ID, pol.Platform); err != nil {
			return ctxerr.Wrapf(ctx, err, "delete outdated hosts membership for policy: %d; platforms: %v", pol.ID, pol.Platform)
		}
	}

	return nil
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

	return int(counts.FailingHostCount), int(counts.TotalHostCount), nil //nolint:gosec // dismiss G115
}

func (ds *Datastore) UpdateHostPolicyCounts(ctx context.Context) error {
	// NOTE these queries are duplicated in the below migration.  Updates
	// to these queries should be reflected there as well.
	// https://github.com/fleetdm/fleet/blob/main/server/datastore/mysql/migrations/tables/20231215122713_InsertPolicyStatsData.go#L12
	// This implementation should be functionally equivalent to the migration.

	// Update Counts for Inherited Global Policies for each Team
	// The original implementation that used INSERT ... SELECT (SELECT COUNT(*)) ... caused performance issues.
	// Given 50 global policies, 10 teams, and 10,000 hosts per team, the INSERT query took 30-60 seconds to complete.
	// Since it was an INSERT query, it blocked other hosts from updating their policy results in policy_membership.

	// Now, we separate the INSERT from the SELECT, since SELECT by itself does not block other hosts from updating their policy results.
	// In addition, we process one global policy at a time, which reduces the time to complete the SELECT query to <2 seconds, and limits the memory usage.
	// We are not using a transaction to reduce locks. This means that INSERT may fail if the policy was deleted by a parallel process.
	// Also, the INSERT may overwrite a clearing of the stats. This is acceptable, since these are very rare cases. We log and proceed in that case.

	db := ds.writer(ctx)

	// Inherited policies are only relevant for teams, so we check whether we have teams
	var hasTeams bool
	err := sqlx.GetContext(ctx, db, &hasTeams, `SELECT 1 FROM teams`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No teams, so no inherited policies
			hasTeams = false
		} else {
			return ctxerr.Wrap(ctx, err, "count teams")
		}
	}

	if hasTeams {
		globalPolicies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list global policies")
		}
		type policyStat struct {
			PolicyID         uint `db:"policy_id"`
			InheritedTeamID  uint `db:"inherited_team_id"`
			PassingHostCount uint `db:"passing_host_count"`
			FailingHostCount uint `db:"failing_host_count"`
		}
		var policyStats []policyStat
		for _, policy := range globalPolicies {
			selectStmt := `SELECT
				p.id as policy_id,
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
			WHERE p.team_id IS NULL AND p.id = ?
			GROUP BY t.id, p.id`
			err = sqlx.SelectContext(ctx, db, &policyStats, selectStmt, policy.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// Policy or team was deleted by a parallel process. We proceed.
					level.Error(ds.logger).Log(
						"msg", "policy not found for inherited global policies. Was policy or team(s) deleted?", "policy_id", policy.ID,
					)
					continue
				}
				return ctxerr.Wrap(ctx, err, "select policy counts for inherited global policies")
			}

			noTeamStmt := `SELECT
				p.id as policy_id,
				0 AS inherited_team_id, -- 0 means "No team"
				(
					SELECT COUNT(*)
					FROM policy_membership pm
					INNER JOIN hosts h ON pm.host_id = h.id
					WHERE pm.policy_id = p.id AND pm.passes = true AND h.team_id IS NULL
				) AS passing_host_count,
				(
					SELECT COUNT(*)
					FROM policy_membership pm
					INNER JOIN hosts h ON pm.host_id = h.id
					WHERE pm.policy_id = p.id AND pm.passes = false AND h.team_id IS NULL
				) AS failing_host_count
			FROM policies p
			WHERE p.team_id IS NULL AND p.id = ?`
			var noTeamPolicyStats []policyStat
			err = sqlx.SelectContext(ctx, db, &noTeamPolicyStats, noTeamStmt, policy.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// Policy was deleted by a parallel process. We proceed.
					level.Error(ds.logger).Log(
						"msg", "'No team' policy not found for inherited global policies. Was policy deleted?", "policy_id", policy.ID,
					)
					continue
				}
				return ctxerr.Wrap(ctx, err, "select policy counts for inherited global policies for 'no team' policies")
			}
			policyStats = append(policyStats, noTeamPolicyStats...)

			insertStmt := `INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
			VALUES (:policy_id, :inherited_team_id, :passing_host_count, :failing_host_count)
			ON DUPLICATE KEY UPDATE
				updated_at = NOW(),
				passing_host_count = VALUES(passing_host_count),
				failing_host_count = VALUES(failing_host_count)`
			_, err = sqlx.NamedExecContext(ctx, db, insertStmt, policyStats)
			if err != nil {
				// INSERT may fail due to rare race conditions. We log and proceed.
				level.Error(ds.logger).Log(
					"msg", "insert policy stats for inherited global policies. Was policy deleted?", "policy_id", policy.ID, "err", err,
				)
			}
		}
	}

	// Update Counts for Global and Team Policies
	// The performance of this query is linear with the number of policies.
	_, err = db.ExecContext(
		ctx, `
		INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		SELECT
			p.id,
			NULL AS inherited_team_id, -- using NULL to represent global scope
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

func (ds *Datastore) GetCalendarPolicies(ctx context.Context, teamID uint) ([]fleet.PolicyCalendarData, error) {
	query := `SELECT id, name FROM policies WHERE team_id = ? AND calendar_events_enabled;`
	var policies []fleet.PolicyCalendarData
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get calendar policies")
	}
	return policies, nil
}

func (ds *Datastore) GetPoliciesWithAssociatedInstaller(ctx context.Context, teamID uint, policyIDs []uint) ([]fleet.PolicySoftwareInstallerData, error) {
	if len(policyIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, software_installer_id FROM policies WHERE team_id = ? AND software_installer_id IS NOT NULL AND id IN (?);`
	query, args, err := sqlx.In(query, teamID, policyIDs)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "build sqlx.In for get policies with associated installer")
	}
	var policies []fleet.PolicySoftwareInstallerData
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies with associated installer")
	}
	return policies, nil
}

func (ds *Datastore) GetPoliciesWithAssociatedScript(ctx context.Context, teamID uint, policyIDs []uint) ([]fleet.PolicyScriptData, error) {
	if len(policyIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, script_id FROM policies WHERE team_id = ? AND script_id IS NOT NULL AND id IN (?);`
	query, args, err := sqlx.In(query, teamID, policyIDs)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "build sqlx.In for get policies with associated script")
	}
	var policies []fleet.PolicyScriptData
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies with associated script")
	}
	return policies, nil
}

func (ds *Datastore) GetTeamHostsPolicyMemberships(
	ctx context.Context,
	domain string,
	teamID uint,
	policyIDs []uint,
	hostID *uint,
) ([]fleet.HostPolicyMembershipData, error) {
	query := `
	SELECT
		COALESCE(sh.email, '') AS email,
		COALESCE(pm.passing, 1) AS passing,
		COALESCE(pm.failing_policy_ids, '') AS failing_policy_ids,
		h.id AS host_id,
		COALESCE(hdn.display_name, '') AS host_display_name,
		h.hardware_serial AS host_hardware_serial
	FROM hosts h
	LEFT JOIN (
		SELECT host_id, 0 AS passing, GROUP_CONCAT(policy_id) AS failing_policy_ids
		FROM policy_membership
		WHERE policy_id IN (?) AND passes = 0
		GROUP BY host_id
	) pm ON h.id = pm.host_id
	LEFT JOIN (
		SELECT host_id, MIN(email) AS email
		FROM host_emails
		JOIN hosts ON host_emails.host_id=hosts.id
		WHERE email LIKE CONCAT('%@', ?) AND team_id = ?
		GROUP BY host_id
	) sh ON h.id = sh.host_id
	LEFT JOIN host_display_names hdn ON h.id = hdn.host_id
	LEFT JOIN host_calendar_events hce ON h.id = hce.host_id
	WHERE h.team_id = ? AND ((pm.passing IS NOT NULL AND NOT pm.passing) OR (COALESCE(pm.passing, 1) AND hce.host_id IS NOT NULL))
`

	query, args, err := sqlx.In(query, policyIDs, domain, teamID, teamID)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "build select get team hosts policy memberships query")
	}
	if hostID != nil {
		query += ` AND h.id = ?`
		args = append(args, *hostID)
	}
	var hosts []fleet.HostPolicyMembershipData
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing policies")
	}

	return hosts, nil
}

// GetPoliciesBySoftwareTitleID returns the policies that are associated with a set of software titles.
func (ds *Datastore) getPoliciesBySoftwareTitleIDs(
	ctx context.Context,
	softwareTitleIDs []uint,
	teamID *uint,
) ([]fleet.AutomaticInstallPolicy, error) {
	if len(softwareTitleIDs) == 0 {
		return nil, nil
	}

	query := `
	SELECT
		p.id AS id,
		p.name AS name,
		st.id AS software_title_id
	FROM policies p
	JOIN software_installers si ON p.software_installer_id = si.id
	JOIN software_titles st ON si.title_id = st.id
	WHERE st.id IN (?) AND p.team_id = ?
`

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	query, args, err := sqlx.In(query, softwareTitleIDs, tmID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build select get policies by software id query")
	}

	var policies []fleet.AutomaticInstallPolicy
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies by software installer id")
	}
	return policies, nil
}
