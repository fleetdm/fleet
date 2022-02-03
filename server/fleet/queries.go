package fleet

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
)

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
	// AuthorEmail is the email address of the author, which is also used to
	// generate the avatar.
	AuthorEmail string `json:"author_email" db:"author_email"`
	// Packs is loaded when retrieving queries, but is stored in a join
	// table in the MySQL backend.
	Packs []Pack `json:"packs" db:"-"`

	AggregatedStats `json:"stats,omitempty"`
}

func (q Query) AuthzType() string {
	return "query"
}

// Verify verifies the query payload is valid.
func (q *QueryPayload) Verify() error {
	if q.Name != nil {
		if err := verifyQueryName(*q.Name); err != nil {
			return err
		}
	}
	if q.Query != nil {
		if err := verifyQuerySQL(*q.Query); err != nil {
			return err
		}
	}
	return nil
}

// Verify verifies the query fields are valid.
func (q *Query) Verify() error {
	if err := verifyQueryName(q.Name); err != nil {
		return err
	}
	if err := verifyQuerySQL(q.Query); err != nil {
		return err
	}
	return nil
}

var (
	validateSQLRegexp  = regexp.MustCompile(`(?i)attach[^\w]+.*[^\w]+as[^\w]+`)
	errQueryEmptyName  = errors.New("query name cannot be empty")
	errQueryEmptyQuery = errors.New("query's SQL query cannot be empty")
	errQueryInvalidSQL = errors.New("invalid query's SQL")
)

func verifyQueryName(name string) error {
	if emptyString(name) {
		return errQueryEmptyName
	}
	return nil
}

func verifyQuerySQL(query string) error {
	if emptyString(query) {
		return errQueryEmptyQuery
	}
	if err := verifySQL(query); err != nil {
		return err
	}
	return nil
}

func verifySQL(query string) error {
	if validateSQLRegexp.MatchString(query) {
		return errQueryInvalidSQL
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
			return nil, fmt.Errorf("unmarshal yaml: %w", err)
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
			return "", fmt.Errorf("marshal YAML: %w", err)
		}
		ymlStrings = append(ymlStrings, string(yml))
	}

	return strings.Join(ymlStrings, "---\n"), nil
}
