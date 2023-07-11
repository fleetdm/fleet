package fleet

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/ptr"
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
	ID uint `json:"id"`
	// TeamID to which team this query belongs. If not set, then the query belongs to the 'Global'
	// team. The table schema for queries includes another column related to this one:
	// `team_id_char`, this is because the unique constraint for queries is based on both the
	// team_id and their name, but since team_id can be null (and (NULL == NULL) != true), we need
	// to use something else to guarantee uniqueness, hence the use of team_id_char. team_id_char
	// will be computed as string(team_id), if team_id IS NULL then team_char_id will be ''.
	TeamID *uint `json:"team_id" db:"team_id"`
	// Interval frequency of execution (in seconds), if 0 then, this query will never run.
	ScheduleInterval uint `json:"interval" db:"schedule_interval"`
	// Platform if set, specifies the platform(s) this query will target.
	Platform string `json:"platform" db:"platform"`
	// MinOsqueryVersion if set, specifies the min required version of osquery that must be
	// installed on the host.
	MinOsqueryVersion string `json:"min_osquery_version" db:"min_osquery_version"`
	// AutomationsEnabled whether to send data to the configured log destination
	AutomationsEnabled bool `json:"automations_enabled" db:"automations_enabled"`
	// LoggingType the type of log output for this query
	LoggingType string `json:"logging" db:"logging_type"`
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

// TeamIDStr returns either the string representation of q.TeamID or ” if nil
func (q *Query) TeamIDStr() string {
	if q == nil || q.TeamID == nil {
		return ""
	}
	return fmt.Sprint(*q.TeamID)
}

func (q *Query) GetSnapshot() *bool {
	var loggingType string
	if q != nil {
		loggingType = q.LoggingType
	}

	switch loggingType {
	case "snapshot":
		return ptr.Bool(true)
	case "differential", "differential_ignore_removals":
		return ptr.Bool(false)
	default:
		// Default value of `snapshot` according the docs is false
		return ptr.Bool(false)
	}
}

func (q *Query) GetRemoved() *bool {
	var loggingType string
	if q != nil {
		loggingType = q.LoggingType
	}

	switch loggingType {
	case "snapshot", "differential_ignore_removals":
		return ptr.Bool(false)
	default:
		// Default value of `removed` according the docs is true
		return ptr.Bool(true)
	}
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

type TargetedQuery struct {
	*Query
	HostTargets HostTargets `json:"host_targets"`
}

func (tq *TargetedQuery) AuthzType() string {
	return "targeted_query"
}

var (
	errQueryEmptyName  = errors.New("query name cannot be empty")
	errQueryEmptyQuery = errors.New("query's SQL query cannot be empty")
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
