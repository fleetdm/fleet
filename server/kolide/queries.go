package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type QueryStore interface {
	// NewQuery creates a new query object in thie datastore. The returned
	// query should have the ID updated.
	NewQuery(query *Query) (*Query, error)
	// SaveQuery saves changes to an existing query object.
	SaveQuery(query *Query) error
	// DeleteQuery (soft) deletes an existing query object.
	DeleteQuery(query *Query) error
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
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Query        *string `json:"query"`
	Interval     *uint   `json:"interval"`
	Snapshot     *bool   `json:"snapshot"`
	Differential *bool   `json:"differential"`
	Platform     *string `json:"platform"`
	Version      *string `json:"version"`
}

type Query struct {
	UpdateCreateTimestamps
	DeleteFields
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Saved        bool   `json:"saved"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	AuthorID     uint   `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL
	// backend (using AuthorID)
	AuthorName string `json:"author_name" db:"author_name"`
	// Packs is loaded when retrieving queries, but is stored in a join
	// table in the MySQL backend.
	Packs []Pack `json:"packs" db:"-"`
}

type Option struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Key       string
	Value     string
	Platform  string
}

type DecoratorType int

const (
	DecoratorLoad DecoratorType = iota
	DecoratorAlways
	DecoratorInterval
)

type Decorator struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      DecoratorType
	Interval  int
	Query     string
}
