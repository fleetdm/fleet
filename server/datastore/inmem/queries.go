package inmem

import (
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

	if err := orm.loadPacksForQueries([]*kolide.Query{query}); err != nil {
		return nil, errors.DatabaseError(err)
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
		if orm.queries[uint(k)].Saved {
			queries = append(queries, orm.queries[uint(k)])
		}
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

	if err := orm.loadPacksForQueries(queries); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return queries, nil
}

// loadPacksForQueries loads the packs associated with the provided queries
func (orm *Datastore) loadPacksForQueries(queries []*kolide.Query) error {
	for _, q := range queries {
		q.Packs = make([]kolide.Pack, 0)
		for _, pq := range orm.packQueries {
			if pq.QueryID == q.ID {
				q.Packs = append(q.Packs, *orm.packs[pq.PackID])
			}
		}
	}

	return nil
}
