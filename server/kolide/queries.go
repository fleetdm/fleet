package kolide

import (
	"context"
)

type QueryStore interface {
	// NewQuery creates a new query object in thie datastore. The returned
	// query should have the ID updated.
	NewQuery(query *Query) (*Query, error)
	// SaveQuery saves changes to an existing query object.
	SaveQuery(query *Query) error
	// DeleteQuery (soft) deletes an existing query object.
	DeleteQuery(qid uint) error
	// DeleteQueries (soft) deletes the existing query objects with the
	// provided IDs. The number of deleted queries is returned along with
	// any error.
	DeleteQueries(ids []uint) (uint, error)
	// Query returns the query associated with the provided ID. Associated
	// packs should also be loaded.
	Query(id uint) (*Query, error)
	// ListQueries returns a list of queries with the provided sorting and
	// paging options. Associated packs should also be loaded.
	ListQueries(opt ListOptions) ([]*Query, error)
	// QueryByName looks up a query by name, the second bool is true if a query
	// by the name exists.
	QueryByName(name string) (*Query, bool, error)
}

type QueryService interface {
	// ListQueries returns a list of saved queries. Note only saved queries
	// should be returned (those that are created for distributed queries
	// but not saved should not be returned).
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, id uint) error
	// DeleteQueries (soft) deletes the existing query objects with the
	// provided IDs. The number of deleted queries is returned along with
	// any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)
}

type QueryPayload struct {
	Name        *string
	Description *string
	Query       *string
}

type Query struct {
	UpdateCreateTimestamps
	DeleteFields
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
	Saved       bool   `json:"saved"`
	AuthorID    uint   `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL
	// backend (using AuthorID)
	AuthorName string `json:"author_name" db:"author_name"`
	// Packs is loaded when retrieving queries, but is stored in a join
	// table in the MySQL backend.
	Packs []Pack `json:"packs" db:"-"`
}
