package mysql

import (
	"context"
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
	res, err := ds.writer.ExecContext(ctx,
		`INSERT INTO policies (name, query, description, resolution, author_id, platforms) VALUES (?, ?, ?, ?, ?, ?)`,
		args.Name, args.Query, args.Description, args.Resolution, authorID, args.Platform,
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
	return policyDB(ctx, ds.writer, uint(lastIdInt64), nil)
}

func (ds *Datastore) Policy(ctx context.Context, id uint) (*fleet.Policy, error) {
	return policyDB(ctx, ds.reader, id, nil)
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
		fmt.Sprintf(`SELECT p.*,
		    COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
       		(select count(*) from policy_membership where policy_id=p.id and passes=true) as passing_host_count,
       		(select count(*) from policy_membership where policy_id=p.id and passes=false) as failing_host_count
		FROM policies p
		LEFT JOIN users u ON p.author_id = u.id
		WHERE p.id=? AND %s`, teamWhere),
		args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting policy")
	}
	return &policy, nil
}

// SavePolicy updates some fields of the given policy on the datastore.
//
// Currently SavePolicy does not allow updating the team or platform of an existing policy,
// such functionality will be implemented in #3220.
func (ds *Datastore) SavePolicy(ctx context.Context, p *fleet.Policy) error {
	sql := `
		UPDATE policies
			SET name = ?, query = ?, description = ?, resolution = ?
			WHERE id = ?
	`
	result, err := ds.writer.ExecContext(ctx, sql, p.Name, p.Query, p.Description, p.Resolution, p.ID)
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
	return nil
}

func (ds *Datastore) RecordPolicyQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool) error {
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
		`INSERT INTO policy_membership (updated_at, policy_id, host_id, passes)
				VALUES %s ON DUPLICATE KEY UPDATE updated_at=VALUES(updated_at), passes=VALUES(passes)`,
		strings.Join(bindvars, ","),
	)

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, query, vals...)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "insert policy_membership (%v)", vals)
		}

		// if we are deferring host updates, we return at this point and do the change outside of the tx
		if deferredSaveHost {
			return nil
		}

		_, err = tx.ExecContext(ctx, `UPDATE hosts SET policy_updated_at = ? WHERE id=?`, updated, host.ID)
		if err != nil {
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

func (ds *Datastore) ListGlobalPolicies(ctx context.Context) ([]*fleet.Policy, error) {
	return listPoliciesDB(ctx, ds.reader, nil)
}

func listPoliciesDB(ctx context.Context, q sqlx.QueryerContext, teamID *uint) ([]*fleet.Policy, error) {
	teamWhere := "p.team_id is NULL"
	var args []interface{}
	if teamID != nil {
		teamWhere = "p.team_id = ?"
		args = append(args, *teamID)
	}
	var policies []*fleet.Policy
	err := sqlx.SelectContext(
		ctx,
		q,
		&policies,
		fmt.Sprintf(`SELECT p.*,
		    COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
       		(select count(*) from policy_membership where policy_id=p.id and passes=true) as passing_host_count,
       		(select count(*) from policy_membership where policy_id=p.id and passes=false) as failing_host_count
		FROM policies p
		LEFT JOIN users u ON p.author_id = u.id
		WHERE %s`, teamWhere), args...,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing policies")
	}
	return policies, nil
}

func (ds *Datastore) DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error) {
	return deletePolicyDB(ctx, ds.writer, ids, nil)
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
		level.Error(ds.logger).Log("err", fmt.Sprintf("host %d with empty platform", host.ID))
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
	if err := sqlx.SelectContext(ctx, ds.reader, &rows, sql, args...); err != nil {
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
	res, err := ds.writer.ExecContext(ctx,
		`INSERT INTO policies (name, query, description, team_id, resolution, author_id, platforms) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		args.Name, args.Query, args.Description, teamID, args.Resolution, authorID, args.Platform)
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
	return policyDB(ctx, ds.writer, uint(lastIdInt64), &teamID)
}

func (ds *Datastore) ListTeamPolicies(ctx context.Context, teamID uint) ([]*fleet.Policy, error) {
	return listPoliciesDB(ctx, ds.reader, &teamID)
}

func (ds *Datastore) DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
	return deletePolicyDB(ctx, ds.writer, ids, &teamID)
}

func (ds *Datastore) TeamPolicy(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
	return policyDB(ctx, ds.reader, policyID, &teamID)
}

// ApplyPolicySpecs applies the given policy specs, creating new policies and updating the ones that
// already exist (a policy is identified by its name and the team it belongs to).
//
// NOTE: Similar to ApplyQueries, ApplyPolicySpecs will update the author_id of the policies
// that are updated.
//
// Currently ApplyPolicySpecs does not allow updating the team or platform of an existing policy,
// such functionality will be implemented in #3220.
func (ds *Datastore) ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		sql := `
		INSERT INTO policies (
			name,
			query,
			description,
			author_id,
			resolution,
			team_id,
			platforms
		) VALUES ( ?, ?, ?, ?, ?, (SELECT IFNULL(MIN(id), NULL) FROM teams WHERE name = ?), ? )
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			query = VALUES(query),
			description = VALUES(description),
			author_id = VALUES(author_id),
			resolution = VALUES(resolution)
		`
		for _, spec := range specs {
			if _, err := tx.ExecContext(ctx,
				sql, spec.Name, spec.Query, spec.Description, authorID, spec.Resolution, spec.Team, spec.Platform,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "exec ApplyPolicySpecs insert")
			}
		}
		return nil
	})
}

func amountPoliciesDB(db sqlx.Queryer) (int, error) {
	var amount int
	err := sqlx.Get(db, &amount, `SELECT count(*) FROM policies`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}
