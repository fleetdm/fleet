package datastore

import (
	"sort"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *inmem) NewQuery(query *kolide.Query) (*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	newQuery := *query

	for _, q := range orm.queries {
		if query.Name == q.Name {
			return nil, ErrExists
		}
	}

	newQuery.ID = uint(len(orm.queries) + 1)
	orm.queries[newQuery.ID] = &newQuery

	return &newQuery, nil
}

func (orm *inmem) SaveQuery(query *kolide.Query) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.queries[query.ID]; !ok {
		return ErrNotFound
	}

	orm.queries[query.ID] = query
	return nil
}

func (orm *inmem) DeleteQuery(query *kolide.Query) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.queries[query.ID]; !ok {
		return ErrNotFound
	}

	delete(orm.queries, query.ID)
	return nil
}

func (orm *inmem) Query(id uint) (*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	query, ok := orm.queries[id]
	if !ok {
		return nil, ErrNotFound
	}

	return query, nil
}

func (orm *inmem) ListQueries(opt kolide.ListOptions) ([]*kolide.Query, error) {
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

	// Apply limit/offset
	low, high := orm.getLimitOffsetSliceBounds(opt, len(queries))
	queries = queries[low:high]

	return queries, nil
}
