package inmem

import (
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (d *Datastore) NewDistributedQueryCampaign(camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	camp.ID = d.nextID(camp)
	d.distributedQueryCampaigns[camp.ID] = *camp

	return camp, nil
}

func (d *Datastore) DistributedQueryCampaign(id uint) (*fleet.DistributedQueryCampaign, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	campaign, ok := d.distributedQueryCampaigns[id]
	if !ok {
		return nil, notFound("DistributedQueryCampaign").WithID(id)
	}

	return &campaign, nil
}

func (d *Datastore) SaveDistributedQueryCampaign(camp *fleet.DistributedQueryCampaign) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.distributedQueryCampaigns[camp.ID]; !ok {
		return notFound("DistributedQueryCampaign").WithID(camp.ID)
	}

	d.distributedQueryCampaigns[camp.ID] = *camp
	return nil
}

func (d *Datastore) DistributedQueryCampaignTargetIDs(id uint) (*fleet.HostTargets, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	hostIDs := []uint{}
	labelIDs := []uint{}
	for _, target := range d.distributedQueryCampaignTargets {
		if target.DistributedQueryCampaignID == id {
			if target.Type == fleet.TargetHost {
				hostIDs = append(hostIDs, target.TargetID)
			} else if target.Type == fleet.TargetLabel {
				labelIDs = append(labelIDs, target.TargetID)
			} else {
				return nil, fmt.Errorf("invalid target type: %d", target.Type)
			}
		}
	}

	return &fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs}, nil
}

func (d *Datastore) NewDistributedQueryCampaignTarget(target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	target.ID = d.nextID(target)
	d.distributedQueryCampaignTargets[target.ID] = *target

	return target, nil
}

func (d *Datastore) CleanupDistributedQueryCampaigns(now time.Time) (expired uint, err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// First expire old waiting and running campaigns
	for id, c := range d.distributedQueryCampaigns {
		if (c.Status == fleet.QueryWaiting && c.CreatedAt.Before(now.Add(-1*time.Minute))) ||
			(c.Status == fleet.QueryRunning && c.CreatedAt.Before(now.Add(-24*time.Hour))) {
			c.Status = fleet.QueryComplete
			d.distributedQueryCampaigns[id] = c
			expired++
		}
	}

	return expired, nil
}
