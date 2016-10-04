package datastore

import (
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

func (orm *inmem) Queries() ([]*kolide.Query, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	queries := []*kolide.Query{}
	for _, query := range orm.queries {
		queries = append(queries, query)
	}

	return queries, nil
}
