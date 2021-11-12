package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (ds *Datastore) NewGlobalPolicy(ctx context.Context, authorID, queryID uint, name, query, description, resolution string) (*fleet.Policy, error) {
	if queryID != 0 {
		q, err := ds.Query(ctx, queryID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching query from id")
		}
		name = q.Name
		query = q.Query
		description = q.Description
	}
	res, err := ds.writer.ExecContext(ctx,
		`INSERT INTO policies (name, query, description, resolution, author_id) VALUES (?, ?, ?, ?, ?)`,
		name, query, description, resolution, authorID,
	)
	switch {
	case err == nil:
		// OK
	case isDuplicate(err):
		return nil, alreadyExists("Policy", name)
	default:
		return nil, errors.Wrap(err, "inserting new policy")
	}

	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "getting last id after inserting policy")
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
		return nil, errors.Wrap(err, "getting policy")
	}
	return &policy, nil
}

// SavePolicy updates some fields of the given policy on the datastore.
func (ds *Datastore) SavePolicy(ctx context.Context, p *fleet.Policy) error {
	sql := `
		UPDATE policies
			SET name = ?, query = ?, description = ?, resolution = ?
			WHERE id = ?
	`
	result, err := ds.writer.ExecContext(ctx, sql, p.Name, p.Query, p.Description, p.Resolution, p.ID)
	if err != nil {
		return errors.Wrap(err, "updating policy")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating policy")
	}
	if rows == 0 {
		return notFound("Policy").WithID(p.ID)
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
		`INSERT INTO policy_membership_history (updated_at, policy_id, host_id, passes)
				VALUES %s`,
		strings.Join(bindvars, ","),
	)

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, query, vals...)
		if err != nil {
			return errors.Wrapf(err, "insert policy_membership (%v)", vals)
		}

		// if we are deferring host updates, we return at this point and do the change outside of the tx
		if deferredSaveHost {
			return nil
		}

		_, err = tx.ExecContext(ctx, `UPDATE hosts SET policy_updated_at = ? WHERE id=?`, updated, host.ID)
		if err != nil {
			return errors.Wrap(err, "updating hosts policy updated at")
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
		return nil, errors.Wrap(err, "listing policies")
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
		return nil, errors.Wrap(err, "IN for DELETE FROM policies")
	}
	stmt = q.Rebind(stmt)

	teamWhere := "TRUE"
	if teamID != nil {
		teamWhere = "team_id = ?"
		args = append(args, *teamID)
	}

	if _, err := q.ExecContext(ctx, fmt.Sprintf(stmt, teamWhere), args...); err != nil {
		return nil, errors.Wrap(err, "delete policies")
	}
	return ids, nil
}

func (ds *Datastore) PolicyQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, error) {
	var globalRows, teamRows []struct {
		Id    string `db:"id"`
		Query string `db:"query"`
	}
	err := sqlx.SelectContext(
		ctx,
		ds.reader,
		&globalRows,
		`SELECT p.id, p.query FROM policies p WHERE team_id is NULL`,
	)
	if err != nil {
		return nil, errors.Wrap(err, "selecting policies for host")
	}

	results := map[string]string{}

	if host.TeamID != nil {
		err := sqlx.SelectContext(
			ctx,
			ds.reader,
			&teamRows,
			`SELECT p.id, p.query FROM policies p WHERE team_id = ?`,
			*host.TeamID,
		)
		if err != nil {
			return nil, errors.Wrap(err, "selecting policies for host in team")
		}
	}

	for _, row := range globalRows {
		results[row.Id] = row.Query
	}

	for _, row := range teamRows {
		results[row.Id] = row.Query
	}

	return results, nil
}

func (ds *Datastore) NewTeamPolicy(ctx context.Context, authorID, teamID, queryID uint, name, query, description, resolution string) (*fleet.Policy, error) {
	if queryID != 0 {
		q, err := ds.Query(ctx, queryID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching query from id")
		}
		name = q.Name
		query = q.Query
		description = q.Description
	}
	res, err := ds.writer.ExecContext(ctx,
		`INSERT INTO policies (name, query, description, team_id, resolution, author_id) VALUES (?, ?, ?, ?, ?, ?)`,
		name, query, description, teamID, resolution, authorID)
	if err != nil {
		return nil, errors.Wrap(err, "inserting new team policy")
	}
	lastIdInt64, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "getting last id after inserting policy")
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
func (ds *Datastore) ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		sql := `
		INSERT INTO policies (
			name,
			query,
			description,
			author_id,
			resolution,
			team_id
		) VALUES ( ?, ?, ?, ?, ?, (SELECT IFNULL(MIN(id), NULL) FROM teams WHERE name = ?) )
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			query = VALUES(query),
			description = VALUES(description),
			author_id = VALUES(author_id),
			resolution = VALUES(resolution),
			team_id = VALUES(team_id)
		`
		for _, spec := range specs {
			if spec.Name == "" {
				return errors.New("policy name must not be empty")
			}
			if spec.Query == "" {
				return errors.New("policy query must not be empty")
			}
			_, err := tx.ExecContext(ctx, sql, spec.Name, spec.Query, spec.Description, authorID, spec.Resolution, spec.Team)
			if err != nil {
				return errors.Wrap(err, "exec ApplyPolicySpecs insert")
			}
		}
		return nil
	})
}
