package inmem

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) (*kolide.DistributedQueryCampaign, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	camp.ID = orm.nextID(camp)
	orm.distributedQueryCampaigns[camp.ID] = *camp

	return camp, nil
}

func (orm *Datastore) DistributedQueryCampaign(id uint) (*kolide.DistributedQueryCampaign, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	campaign, ok := orm.distributedQueryCampaigns[id]
	if !ok {
		return nil, errors.ErrNotFound
	}

	return &campaign, nil
}

func (orm *Datastore) SaveDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.distributedQueryCampaigns[camp.ID]; !ok {
		return errors.ErrNotFound
	}

	orm.distributedQueryCampaigns[camp.ID] = *camp
	return nil
}

func (orm *Datastore) DistributedQueryCampaignTargetIDs(id uint) (hostIDs []uint, labelIDs []uint, err error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	hostIDs = []uint{}
	labelIDs = []uint{}
	for _, target := range orm.distributedQueryCampaignTargets {
		if target.DistributedQueryCampaignID == id {
			if target.Type == kolide.TargetHost {
				hostIDs = append(hostIDs, target.TargetID)
			} else if target.Type == kolide.TargetLabel {
				labelIDs = append(labelIDs, target.TargetID)
			} else {
				return []uint{}, []uint{}, fmt.Errorf("invalid target type: %d", target.Type)
			}
		}
	}

	return hostIDs, labelIDs, nil
}

func (orm *Datastore) NewDistributedQueryCampaignTarget(target *kolide.DistributedQueryCampaignTarget) (*kolide.DistributedQueryCampaignTarget, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	target.ID = orm.nextID(target)
	orm.distributedQueryCampaignTargets[target.ID] = *target

	return target, nil
}

func (orm *Datastore) NewDistributedQueryExecution(exec *kolide.DistributedQueryExecution) (*kolide.DistributedQueryExecution, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, e := range orm.distributedQueryExecutions {
		if exec.HostID == e.HostID && exec.DistributedQueryCampaignID == e.DistributedQueryCampaignID {
			fmt.Printf("%+v -- %+v\n", exec, orm.distributedQueryExecutions)
			return exec, errors.ErrExists
		}
	}

	exec.ID = orm.nextID(exec)
	orm.distributedQueryExecutions[exec.ID] = *exec

	return exec, nil
}

func (orm *Datastore) CleanupDistributedQueryCampaigns(now time.Time) (expired uint, deleted uint, err error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// First expire old waiting and running campaigns
	for id, c := range orm.distributedQueryCampaigns {
		if (c.Status == kolide.QueryWaiting && c.CreatedAt.Before(now.Add(-1*time.Minute))) ||
			(c.Status == kolide.QueryRunning && c.CreatedAt.Before(now.Add(-24*time.Hour))) {
			c.Status = kolide.QueryComplete
			orm.distributedQueryCampaigns[id] = c
			expired++
		}
	}

	// Now delete executions for expired campaigns
	for id, e := range orm.distributedQueryExecutions {
		c, ok := orm.distributedQueryCampaigns[e.DistributedQueryCampaignID]
		if !ok || c.Status == kolide.QueryComplete {
			delete(orm.distributedQueryExecutions, id)
			deleted++
		}
	}

	return expired, deleted, nil
}
