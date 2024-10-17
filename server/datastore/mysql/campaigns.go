package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewDistributedQueryCampaign(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
	args := []any{camp.QueryID, camp.Status, camp.UserID}

	// for tests, we sometimes provide specific timestamps for CreatedAt, honor
	// those if provided.
	var createdAtField, createdAtPlaceholder string
	if !camp.CreatedAt.IsZero() {
		createdAtField = ", created_at"
		createdAtPlaceholder = ", ?"
		args = append(args, camp.CreatedAt)
	}

	sqlStatement := fmt.Sprintf(`
		INSERT INTO distributed_query_campaigns (
			query_id,
			status,
			user_id
			%s
		)
		VALUES(?,?,?%s)
	`, createdAtField, createdAtPlaceholder)
	result, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting distributed query campaign")
	}

	id, _ := result.LastInsertId()
	camp.ID = uint(id) //nolint:gosec // dismiss G115
	return camp, nil
}

func (ds *Datastore) DistributedQueryCampaign(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
	sql := `
		SELECT * FROM distributed_query_campaigns WHERE id = ?
	`
	campaign := &fleet.DistributedQueryCampaign{}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), campaign, sql, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting distributed query campaign")
	}

	return campaign, nil
}

func (ds *Datastore) SaveDistributedQueryCampaign(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
	sqlStatement := `
		UPDATE distributed_query_campaigns SET
			query_id = ?,
			status = ?,
			user_id = ?
		WHERE id = ?
	`
	result, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, camp.QueryID, camp.Status, camp.UserID, camp.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating distributed query campaign")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected updating distributed query campaign")
	}
	if rowsAffected == 0 {
		return notFound("DistributedQueryCampaign").WithID(camp.ID)
	}

	return nil
}

func (ds *Datastore) DistributedQueryCampaignsForQuery(ctx context.Context, queryID uint) ([]*fleet.DistributedQueryCampaign, error) {
	var campaigns []*fleet.DistributedQueryCampaign
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &campaigns, `SELECT * FROM distributed_query_campaigns WHERE query_id=?`, queryID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting campaigns for query")
	}
	return campaigns, nil
}

func (ds *Datastore) DistributedQueryCampaignTargetIDs(ctx context.Context, id uint) (*fleet.HostTargets, error) {
	sqlStatement := `
		SELECT * FROM distributed_query_campaign_targets WHERE distributed_query_campaign_id = ?
	`
	targets := []fleet.DistributedQueryCampaignTarget{}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &targets, sqlStatement, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select distributed campaign target")
	}

	hostIDs := []uint{}
	labelIDs := []uint{}
	teamIDs := []uint{}
	for _, target := range targets {
		switch target.Type {
		case fleet.TargetHost:
			hostIDs = append(hostIDs, target.TargetID)
		case fleet.TargetLabel:
			labelIDs = append(labelIDs, target.TargetID)
		case fleet.TargetTeam:
			teamIDs = append(teamIDs, target.TargetID)
		default:
			return nil, ctxerr.Errorf(ctx, "invalid target type: %d", target.Type)
		}
	}

	return &fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs, TeamIDs: teamIDs}, nil
}

func (ds *Datastore) NewDistributedQueryCampaignTarget(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
	sqlStatement := `
		INSERT into distributed_query_campaign_targets (
			type,
			distributed_query_campaign_id,
			target_id
		)
		VALUES (?,?,?)
	`
	result, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, target.Type, target.DistributedQueryCampaignID, target.TargetID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert distributed campaign target")
	}

	id, _ := result.LastInsertId()
	target.ID = uint(id) //nolint:gosec // dismiss G115
	return target, nil
}

func (ds *Datastore) GetCompletedCampaigns(ctx context.Context, filter []uint) ([]uint, error) {
	// There is a limit of 65,535 (2^16-1) placeholders in MySQL 5.7
	const batchSize = 65535 - 1
	if len(filter) == 0 {
		return nil, nil
	}

	// We must remove duplicates from the input filter because we process the filter in batches,
	// and that could result in duplicated result IDs
	filter = server.RemoveDuplicatesFromSlice(filter)

	completed := make([]uint, 0, len(filter))
	for i := 0; i < len(filter); i += batchSize {
		end := i + batchSize
		if end > len(filter) {
			end = len(filter)
		}
		batch := filter[i:end]

		query, args, err := sqlx.In(
			`SELECT id
			FROM distributed_query_campaigns
			WHERE status = ?
			AND id IN (?)
		`, fleet.QueryComplete, batch,
		)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "building query for completed campaigns")
		}

		var rows []uint
		// using the writer, so we catch the ones we just marked as completed
		err = sqlx.SelectContext(ctx, ds.writer(ctx), &rows, query, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "selecting completed campaigns")
		}
		completed = append(completed, rows...)
	}
	return completed, nil
}

func (ds *Datastore) CleanupDistributedQueryCampaigns(ctx context.Context, now time.Time) (expired uint, err error) {
	// Expire old waiting/running campaigns
	const sqlStatement = `
		UPDATE distributed_query_campaigns
		SET status = ?
		WHERE (status = ? AND created_at < ?)
		OR (status = ? AND created_at < ?)
	`
	result, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, fleet.QueryComplete,
		fleet.QueryWaiting, now.Add(-1*time.Minute),
		fleet.QueryRunning, now.Add(-24*time.Hour))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "updating distributed query campaign")
	}

	exp, err := result.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "rows affected updating distributed query campaign")
	}
	return uint(exp), nil //nolint:gosec // dismiss G115
}
