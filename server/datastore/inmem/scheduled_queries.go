package inmem

import (
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	newScheduledQuery := *sq

	newScheduledQuery.ID = orm.nextID(newScheduledQuery)
	orm.scheduledQueries[newScheduledQuery.ID] = &newScheduledQuery

	return &newScheduledQuery, nil
}

func (orm *Datastore) SaveScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.scheduledQueries[sq.ID]; !ok {
		return nil, errors.ErrNotFound
	}

	orm.scheduledQueries[sq.ID] = sq
	return sq, nil
}

func (orm *Datastore) DeleteScheduledQuery(id uint) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.scheduledQueries[id]; !ok {
		return errors.ErrNotFound
	}

	delete(orm.scheduledQueries, id)
	return nil
}

func (orm *Datastore) ScheduledQuery(id uint) (*kolide.ScheduledQuery, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	sq, ok := orm.scheduledQueries[id]
	if !ok {
		return nil, errors.ErrNotFound
	}

	sq.Name = orm.queries[sq.QueryID].Name
	sq.Query = orm.queries[sq.QueryID].Query

	return sq, nil
}

func (orm *Datastore) ListScheduledQueriesInPack(id uint, opt kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, sq := range orm.scheduledQueries {
		if sq.PackID == id {
			keys = append(keys, int(k))
		}
	}

	if len(keys) == 0 {
		return []*kolide.ScheduledQuery{}, nil
	}

	sort.Ints(keys)

	scheduledQueries := []*kolide.ScheduledQuery{}
	for _, k := range keys {
		q := orm.scheduledQueries[uint(k)]
		q.Name = orm.queries[q.QueryID].Name
		q.Query = orm.queries[q.QueryID].Query
		scheduledQueries = append(scheduledQueries, q)
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":         "ID",
			"created_at": "CreatedAt",
			"updated_at": "UpdatedAt",
			"name":       "Name",
			"query":      "Query",
			"interval":   "Interval",
			"snapshot":   "Snapshot",
			"removed":    "Removed",
			"platform":   "Platform",
			"version":    "Version",
		}
		if err := sortResults(scheduledQueries, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := orm.getLimitOffsetSliceBounds(opt, len(scheduledQueries))
	scheduledQueries = scheduledQueries[low:high]

	return scheduledQueries, nil
}
