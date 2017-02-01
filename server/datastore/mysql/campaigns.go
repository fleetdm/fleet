package mysql

import (
	"fmt"
	"time"

	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) (*kolide.DistributedQueryCampaign, error) {

	sqlStatement := `
		INSERT INTO distributed_query_campaigns (
			query_id,
			status,
			user_id
		)
		VALUES(?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, camp.QueryID, camp.Status, camp.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "inserting distributed query campaign")
	}

	id, _ := result.LastInsertId()
	camp.ID = uint(id)
	return camp, nil
}

func (d *Datastore) DistributedQueryCampaign(id uint) (*kolide.DistributedQueryCampaign, error) {
	sql := `
		SELECT * FROM distributed_query_campaigns WHERE id = ? AND NOT deleted
	`
	campaign := &kolide.DistributedQueryCampaign{}
	if err := d.db.Get(campaign, sql, id); err != nil {
		return nil, errors.Wrap(err, "selecting distributed query campaign")
	}

	return campaign, nil
}

func (d *Datastore) SaveDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) error {
	sqlStatement := `
		UPDATE distributed_query_campaigns SET
			query_id = ?,
			status = ?,
			user_id = ?
		WHERE id = ?
		AND NOT deleted
	`
	_, err := d.db.Exec(sqlStatement, camp.QueryID, camp.Status, camp.UserID, camp.ID)
	if err != nil {
		return errors.Wrap(err, "updating distributed query campaign")
	}

	return nil
}

func (d *Datastore) DistributedQueryCampaignTargetIDs(id uint) (hostIDs []uint, labelIDs []uint, err error) {
	sqlStatement := `
		SELECT * FROM distributed_query_campaign_targets WHERE distributed_query_campaign_id = ?
	`
	targets := []kolide.DistributedQueryCampaignTarget{}

	if err = d.db.Select(&targets, sqlStatement, id); err != nil {
		return nil, nil, errors.Wrap(err, "selecting distributed campaign target")
	}

	hostIDs = []uint{}
	labelIDs = []uint{}
	for _, target := range targets {
		if target.Type == kolide.TargetHost {
			hostIDs = append(hostIDs, target.TargetID)
		} else if target.Type == kolide.TargetLabel {
			labelIDs = append(labelIDs, target.TargetID)
		} else {
			return []uint{}, []uint{}, fmt.Errorf("invalid target type: %d", target.Type)
		}
	}

	return hostIDs, labelIDs, nil
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
		return nil, errors.Wrap(err, "insert distributed campaign target")
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
		return nil, errors.Wrap(err, "insert distributed campaign target")
	}

	id, _ := result.LastInsertId()
	exec.ID = uint(id)

	return exec, nil
}

func (d *Datastore) CleanupDistributedQueryCampaigns(now time.Time) (expired uint, deleted uint, err error) {
	// First expire old waiting and running campaigns
	sqlStatement := `
		UPDATE distributed_query_campaigns
		SET status = ?
		WHERE (status = ? AND created_at < ?)
		OR (status = ? AND created_at < ?)
	`
	result, err := d.db.Exec(sqlStatement, kolide.QueryComplete,
		kolide.QueryWaiting, now.Add(-1*time.Minute),
		kolide.QueryRunning, now.Add(-24*time.Hour))
	if err != nil {
		return expired, deleted, errors.Wrap(err, "updating distributed query campaign")
	}

	exp, err := result.RowsAffected()
	if err != nil {
		return expired, deleted, errors.Wrap(err, "rows effected updating distributed query campaign")
	}
	expired = uint(exp)

	// Now delete executions for expired campaigns
	sqlStatement = `
		DELETE dqe
		FROM distributed_query_executions dqe
		JOIN distributed_query_campaigns dqc
		ON dqe.distributed_query_campaign_id = dqc.id
		WHERE dqc.status = ?
	`
	result, err = d.db.Exec(sqlStatement, kolide.QueryComplete)
	if err != nil {
		return expired, deleted, errors.Wrap(err, "deleting distributed campaign executions")
	}

	del, err := result.RowsAffected()
	if err != nil {
		return expired, deleted, errors.Wrap(err, "rows effected deleting distributed campaign")
	}
	deleted = uint(del)

	return expired, deleted, nil
}
