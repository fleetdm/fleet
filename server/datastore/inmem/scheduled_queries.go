package inmem

import (
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	newScheduledQuery := *sq

	newScheduledQuery.ID = d.nextID(newScheduledQuery)
	d.scheduledQueries[newScheduledQuery.ID] = &newScheduledQuery

	return &newScheduledQuery, nil
}

func (d *Datastore) SaveScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.scheduledQueries[sq.ID]; !ok {
		return nil, errors.ErrNotFound
	}

	d.scheduledQueries[sq.ID] = sq
	return sq, nil
}

func (d *Datastore) DeleteScheduledQuery(id uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.scheduledQueries[id]; !ok {
		return errors.ErrNotFound
	}

	delete(d.scheduledQueries, id)
	return nil
}

func (d *Datastore) ScheduledQuery(id uint) (*kolide.ScheduledQuery, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	sq, ok := d.scheduledQueries[id]
	if !ok {
		return nil, errors.ErrNotFound
	}

	sq.Name = d.queries[sq.QueryID].Name
	sq.Query = d.queries[sq.QueryID].Query

	return sq, nil
}

func (d *Datastore) ListScheduledQueriesInPack(id uint, opt kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, sq := range d.scheduledQueries {
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
		q := d.scheduledQueries[uint(k)]
		q.Name = d.queries[q.QueryID].Name
		q.Query = d.queries[q.QueryID].Query
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
	low, high := d.getLimitOffsetSliceBounds(opt, len(scheduledQueries))
	scheduledQueries = scheduledQueries[low:high]

	return scheduledQueries, nil
}
