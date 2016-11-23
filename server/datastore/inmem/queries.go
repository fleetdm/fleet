package inmem

import (
	"fmt"
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewQuery(query *kolide.Query) (*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	newQuery := *query

	for _, q := range orm.queries {
		if query.Name == q.Name {
			return nil, errors.ErrExists
		}
	}

	newQuery.ID = orm.nextID(newQuery)
	orm.queries[newQuery.ID] = &newQuery

	return &newQuery, nil
}

func (orm *Datastore) SaveQuery(query *kolide.Query) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.queries[query.ID]; !ok {
		return errors.ErrNotFound
	}

	orm.queries[query.ID] = query
	return nil
}

func (orm *Datastore) DeleteQuery(query *kolide.Query) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.queries[query.ID]; !ok {
		return errors.ErrNotFound
	}

	delete(orm.queries, query.ID)
	return nil
}

func (orm *Datastore) Query(id uint) (*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	query, ok := orm.queries[id]
	if !ok {
		return nil, errors.ErrNotFound
	}

	return query, nil
}

func (orm *Datastore) ListQueries(opt kolide.ListOptions) ([]*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range orm.queries {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	queries := []*kolide.Query{}
	for _, k := range keys {
		queries = append(queries, orm.queries[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":           "ID",
			"created_at":   "CreatedAt",
			"updated_at":   "UpdatedAt",
			"name":         "Name",
			"query":        "Query",
			"interval":     "Interval",
			"snapshot":     "Snapshot",
			"differential": "Differential",
			"platform":     "Platform",
			"version":      "Version",
		}
		if err := sortResults(queries, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := orm.getLimitOffsetSliceBounds(opt, len(queries))
	queries = queries[low:high]

	return queries, nil
}

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
		if exec.HostID == e.ID && exec.DistributedQueryCampaignID == e.DistributedQueryCampaignID {
			return exec, errors.ErrExists
		}
	}

	exec.ID = orm.nextID(exec)
	orm.distributedQueryExecutions[exec.ID] = *exec

	return exec, nil
}
