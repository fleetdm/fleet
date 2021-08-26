package mysql

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (d *Datastore) NewDistributedQueryCampaign(camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {

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

func (d *Datastore) DistributedQueryCampaign(id uint) (*fleet.DistributedQueryCampaign, error) {
	sql := `
		SELECT * FROM distributed_query_campaigns WHERE id = ?
	`
	campaign := &fleet.DistributedQueryCampaign{}
	if err := d.db.Get(campaign, sql, id); err != nil {
		return nil, errors.Wrap(err, "selecting distributed query campaign")
	}

	return campaign, nil
}

func (d *Datastore) SaveDistributedQueryCampaign(camp *fleet.DistributedQueryCampaign) error {
	sqlStatement := `
		UPDATE distributed_query_campaigns SET
			query_id = ?,
			status = ?,
			user_id = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(sqlStatement, camp.QueryID, camp.Status, camp.UserID, camp.ID)
	if err != nil {
		return errors.Wrap(err, "updating distributed query campaign")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating distributed query campaign")
	}
	if rowsAffected == 0 {
		return notFound("DistributedQueryCampaign").WithID(camp.ID)
	}

	return nil
}

func (d *Datastore) DistributedQueryCampaignTargetIDs(id uint) (*fleet.HostTargets, error) {
	sqlStatement := `
		SELECT * FROM distributed_query_campaign_targets WHERE distributed_query_campaign_id = ?
	`
	targets := []fleet.DistributedQueryCampaignTarget{}

	if err := d.db.Select(&targets, sqlStatement, id); err != nil {
		return nil, errors.Wrap(err, "select distributed campaign target")
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
			return nil, errors.Errorf("invalid target type: %d", target.Type)
		}
	}

	return &fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs, TeamIDs: teamIDs}, nil
}

func (d *Datastore) NewDistributedQueryCampaignTarget(target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
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

func (d *Datastore) CleanupDistributedQueryCampaigns(now time.Time) (expired uint, err error) {
	// Expire old waiting/running campaigns
	sqlStatement := `
		UPDATE distributed_query_campaigns
		SET status = ?
		WHERE (status = ? AND created_at < ?)
		OR (status = ? AND created_at < ?)
	`
	result, err := d.db.Exec(sqlStatement, fleet.QueryComplete,
		fleet.QueryWaiting, now.Add(-1*time.Minute),
		fleet.QueryRunning, now.Add(-24*time.Hour))
	if err != nil {
		return 0, errors.Wrap(err, "updating distributed query campaign")
	}

	exp, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "rows effected updating distributed query campaign")
	}

	return uint(exp), nil
}
