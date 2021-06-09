package fleet

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

type QueryStore interface {
	// ApplyQueries applies a list of queries (likely from a yaml file) to
	// the datastore. Existing queries are updated, and new queries are
	// created.
	ApplyQueries(authorID uint, queries []*Query) error

	// NewQuery creates a new query object in thie datastore. The returned
	// query should have the ID updated.
	NewQuery(query *Query, opts ...OptionalArg) (*Query, error)
	// SaveQuery saves changes to an existing query object.
	SaveQuery(query *Query) error
	// DeleteQuery deletes an existing query object.
	DeleteQuery(name string) error
	// DeleteQueries deletes the existing query objects with the provided IDs.
	// The number of deleted queries is returned along with any error.
	DeleteQueries(ids []uint) (uint, error)
	// Query returns the query associated with the provided ID. Associated
	// packs should also be loaded.
	Query(id uint) (*Query, error)
	// ListQueries returns a list of queries with the provided sorting and
	// paging options. Associated packs should also be loaded.
	ListQueries(opt ListOptions) ([]*Query, error)
	// QueryByName looks up a query by name.
	QueryByName(name string, opts ...OptionalArg) (*Query, error)
}

type QueryService interface {
	// ApplyQuerySpecs applies a list of queries (creating or updating
	// them as necessary)
	ApplyQuerySpecs(ctx context.Context, specs []*QuerySpec) error
	// GetQuerySpecs gets the YAML file representing all the stored queries.
	GetQuerySpecs(ctx context.Context) ([]*QuerySpec, error)
	// GetQuerySpec gets the spec for the query with the given name.
	GetQuerySpec(ctx context.Context, name string) (*QuerySpec, error)

	// ListQueries returns a list of saved queries. Note only saved queries
	// should be returned (those that are created for distributed queries
	// but not saved should not be returned).
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, name string) error
	// For backwards compatibility with UI
	DeleteQueryByID(ctx context.Context, id uint) error
	// DeleteQueries deletes the existing query objects with the provided IDs.
	// The number of deleted queries is returned along with any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)
}

type QueryPayload struct {
	Name           *string
	Description    *string
	Query          *string
	ObserverCanRun *bool `json:"observer_can_run"`
}

type Query struct {
	UpdateCreateTimestamps
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
	Saved       bool   `json:"saved"`
	// ObserverCanRun indicates whether users with Observer role can run this as
	// a live query.
	ObserverCanRun bool  `json:"observer_can_run" db:"observer_can_run"`
	AuthorID       *uint `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL
	// backend (using AuthorID)
	AuthorName string `json:"author_name" db:"author_name"`
	// Packs is loaded when retrieving queries, but is stored in a join
	// table in the MySQL backend.
	Packs []Pack `json:"packs" db:"-"`
}

func (q Query) AuthzType() string {
	return "query"
}

var (
	validateSQLRegexp = regexp.MustCompile(`(?i)attach[^\w]+.*[^\w]+as[^\w]+`)
)

// ValidateSQL performs security validations on the input query. It does not
// actually determine whether the query is well formed.
func (q Query) ValidateSQL() error {
	if validateSQLRegexp.MatchString(q.Query) {
		return fmt.Errorf("ATTACH not allowed in queries")
	}
	return nil
}

const (
	QueryKind = "query"
)

type QueryObject struct {
	ObjectMetadata
	Spec QuerySpec `json:"spec"`
}

type QuerySpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Query       string `json:"query"`
}

func LoadQueriesFromYaml(yml string) ([]*Query, error) {
	queries := []*Query{}
	for _, s := range strings.Split(yml, "---") {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}

		var q QueryObject
		err := yaml.Unmarshal([]byte(s), &q)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal yaml")
		}
		queries = append(queries,
			&Query{Name: q.Spec.Name, Description: q.Spec.Description, Query: q.Spec.Query},
		)
	}

	return queries, nil
}

func WriteQueriesToYaml(queries []*Query) (string, error) {
	ymlStrings := []string{}
	for _, q := range queries {
		qYaml := QueryObject{
			ObjectMetadata: ObjectMetadata{
				ApiVersion: ApiVersion,
				Kind:       QueryKind,
			},
			Spec: QuerySpec{
				Name:        q.Name,
				Description: q.Description,
				Query:       q.Query,
			},
		}
		yml, err := yaml.Marshal(qYaml)
		if err != nil {
			return "", errors.Wrap(err, "marshal YAML")
		}
		ymlStrings = append(ymlStrings, string(yml))
	}

	return strings.Join(ymlStrings, "---\n"), nil
}
