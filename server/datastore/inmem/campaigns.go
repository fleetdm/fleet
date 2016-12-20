package inmem

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) (*kolide.DistributedQueryCampaign, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	camp.ID = d.nextID(camp)
	d.distributedQueryCampaigns[camp.ID] = *camp

	return camp, nil
}

func (d *Datastore) DistributedQueryCampaign(id uint) (*kolide.DistributedQueryCampaign, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	campaign, ok := d.distributedQueryCampaigns[id]
	if !ok {
		return nil, notFound("DistributedQueryCampaign").WithID(id)
	}

	return &campaign, nil
}

func (d *Datastore) SaveDistributedQueryCampaign(camp *kolide.DistributedQueryCampaign) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.distributedQueryCampaigns[camp.ID]; !ok {
		return notFound("DistributedQueryCampaign").WithID(camp.ID)
	}

	d.distributedQueryCampaigns[camp.ID] = *camp
	return nil
}

func (d *Datastore) DistributedQueryCampaignTargetIDs(id uint) (hostIDs []uint, labelIDs []uint, err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	hostIDs = []uint{}
	labelIDs = []uint{}
	for _, target := range d.distributedQueryCampaignTargets {
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

func (d *Datastore) NewDistributedQueryCampaignTarget(target *kolide.DistributedQueryCampaignTarget) (*kolide.DistributedQueryCampaignTarget, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	target.ID = d.nextID(target)
	d.distributedQueryCampaignTargets[target.ID] = *target

	return target, nil
}

func (d *Datastore) NewDistributedQueryExecution(exec *kolide.DistributedQueryExecution) (*kolide.DistributedQueryExecution, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, e := range d.distributedQueryExecutions {
		if exec.HostID == e.HostID && exec.DistributedQueryCampaignID == e.DistributedQueryCampaignID {
			fmt.Printf("%+v -- %+v\n", exec, d.distributedQueryExecutions)
			return exec, alreadyExists("DistributedQueryExecution", exec.HostID)
		}
	}

	exec.ID = d.nextID(exec)
	d.distributedQueryExecutions[exec.ID] = *exec

	return exec, nil
}

func (d *Datastore) CleanupDistributedQueryCampaigns(now time.Time) (expired uint, deleted uint, err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// First expire old waiting and running campaigns
	for id, c := range d.distributedQueryCampaigns {
		if (c.Status == kolide.QueryWaiting && c.CreatedAt.Before(now.Add(-1*time.Minute))) ||
			(c.Status == kolide.QueryRunning && c.CreatedAt.Before(now.Add(-24*time.Hour))) {
			c.Status = kolide.QueryComplete
			d.distributedQueryCampaigns[id] = c
			expired++
		}
	}

	// Now delete executions for expired campaigns
	for id, e := range d.distributedQueryExecutions {
		c, ok := d.distributedQueryCampaigns[e.DistributedQueryCampaignID]
		if !ok || c.Status == kolide.QueryComplete {
			delete(d.distributedQueryExecutions, id)
			deleted++
		}
	}

	return expired, deleted, nil
}
