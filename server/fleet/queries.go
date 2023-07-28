package fleet

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
)

// QueryPayload is the payload used to create and modify queries.
//
// Fields are pointers to allow omitting fields when modifying existing queries.
type QueryPayload struct {
	// Name is the name to set to the query.
	Name *string `json:"name"`
	// Description is the description of the query.
	Description *string `json:"description"`
	// Query is the actual SQL query to run on devices.
	Query *string `json:"query"`
	// ObserverCanRun is set to false if not set when creating a query.
	ObserverCanRun *bool `json:"observer_can_run"`
	// TeamID is only used when creating a query. When modifying a query
	// TeamID is ignored.
	TeamID *uint `json:"team_id"`
	// Interval is the interval to set on the query. If not set when creating
	// a query, then the default value 0 is set on the query.
	Interval *uint `json:"interval"`
	// Platform is set to empty if not set when creating a query.
	Platform *string `json:"platform"`
	// MinOsqueryVersion is set to empty if not set when creating a query.
	MinOsqueryVersion *string `json:"min_osquery_version"`
	// AutomationsEnabled is set to false if not set when creating a query.
	AutomationsEnabled *bool `json:"automations_enabled"`
	// Logging is set to "snapshot" if not set when creating a query.
	Logging *string `json:"logging"`
}

// Query represents a osquery query to run on devices.
//
// - If Interval is 0 or AutomationsEnabled is false, then the query is disabled from running as
// a scheduled query (the only way to run them on devices is manually via the live queries API).
// - If Interval is not 0 and AutomationsEnabled is true, then the query is configured to run on
// devices at the provided interval; the query considered a "scheduled query". Fields `Platform`,
// `MinOsqueryVersion`, `AutomationsEnabled` and `Logging` are used when this is the case.
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
	Interval uint `json:"interval" db:"schedule_interval"`
	// Platform if set, specifies the platform(s) this query will target.
	//
	// It's a comma-separated list of platforms where this query will run be when configured
	// on a schedule.
	Platform string `json:"platform" db:"platform"`
	// MinOsqueryVersion if set, specifies the min required version of osquery that must be
	// installed on the host.
	MinOsqueryVersion string `json:"min_osquery_version" db:"min_osquery_version"`
	// AutomationsEnabled whether to send data to the configured log destination
	AutomationsEnabled bool `json:"automations_enabled" db:"automations_enabled"`
	// Logging the type of log output for this query
	Logging     string `json:"logging" db:"logging_type"`
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
	// AggregatedStats are the stats aggregated from all the individual stats reported
	// by hosts.
	//
	// This field has null values if the query did not run as a schedule on any host.
	AggregatedStats `json:"stats"`
}

var (
	LoggingSnapshot                   = "snapshot"
	LoggingDifferential               = "differential"
	LoggingDifferentialIgnoreRemovals = "differential_ignore_removals"
)

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
	var logging string
	if q != nil {
		logging = q.Logging
	}

	switch logging {
	case "snapshot":
		return ptr.Bool(true)
	default:
		return nil
	}
}

func (q *Query) GetRemoved() *bool {
	var logging string
	if q != nil {
		logging = q.Logging
	}

	switch logging {
	case "differential":
		return ptr.Bool(true)
	case "differential_ignore_removals":
		return ptr.Bool(false)
	default:
		return nil
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
	if q.Logging != nil {
		if err := verifyLogging(*q.Logging); err != nil {
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
	if err := verifyLogging(q.Logging); err != nil {
		return err
	}
	return nil
}

func (q *Query) ToQueryContent() QueryContent {
	return QueryContent{
		Query:    q.Query,
		Interval: q.Interval,
		Platform: &q.Platform,
		Version:  &q.MinOsqueryVersion,
		Removed:  q.GetRemoved(),
		Snapshot: q.GetSnapshot(),
	}
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
	errInvalidLogging  = fmt.Errorf("invalid logging value, must be one of '%s', '%s', '%s'", LoggingSnapshot, LoggingDifferential, LoggingDifferentialIgnoreRemovals)
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

func verifyLogging(logging string) error {
	// Empty string means snapshot.
	if logging != "" && logging != LoggingSnapshot && logging != LoggingDifferential && logging != LoggingDifferentialIgnoreRemovals {
		return errInvalidLogging
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

// QuerySpec allows creating/editing queries using "specs".
type QuerySpec struct {
	// Name is the name of the query (which is unique in its team or globally).
	// This field must be non-empty.
	Name string `json:"name"`
	// Description is the description of the query.
	Description string `json:"description"`
	// Query is the actual osquery SQL query. This field must be non-empty.
	Query string `json:"query"`

	// TeamName is the team's name, the default "" means the query will be
	// created globally. This field is only used when creating a query,
	// when editing a query this field is ignored.
	TeamName string `json:"team"`
	// Interval is set to 0 if not set.
	Interval uint `json:"interval"`
	// ObserverCanRun is set to false if not set.
	ObserverCanRun bool `json:"observer_can_run"`
	// Platform is set to empty if not set when creating a query.
	Platform string `json:"platform"`
	// MinOsqueryVersion is set to empty if not set.
	MinOsqueryVersion string `json:"min_osquery_version"`
	// AutomationsEnabled is set to false if not set.
	AutomationsEnabled bool `json:"automations_enabled"`
	// Logging is set to "snapshot" if not set.
	Logging string `json:"logging"`
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
			&Query{
				Name:        q.Spec.Name,
				Description: q.Spec.Description,
				Query:       q.Spec.Query,
			},
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

type QueryStats struct {
	ID          uint   `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	TeamID      *uint  `json:"team_id" db:"team_id"`

	// From osquery directly
	AverageMemory int  `json:"average_memory" db:"average_memory"`
	Denylisted    bool `json:"denylisted" db:"denylisted"`
	Executions    int  `json:"executions" db:"executions"`
	// Note schedule_interval is used for DB since "interval" is a reserved word in MySQL
	Interval     int       `json:"interval" db:"schedule_interval"`
	LastExecuted time.Time `json:"last_executed" db:"last_executed"`
	OutputSize   int       `json:"output_size" db:"output_size"`
	SystemTime   int       `json:"system_time" db:"system_time"`
	UserTime     int       `json:"user_time" db:"user_time"`
	WallTime     int       `json:"wall_time" db:"wall_time"`
}
