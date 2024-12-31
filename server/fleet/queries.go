package fleet

import (
	"database/sql"
	"encoding/json"
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
	// DiscardData indicates if the scheduled query results should be discarded (true)
	// or kept (false) in a query report.
	//
	// If not set during creation of a query, then the default value is false.
	DiscardData *bool `json:"discard_data"`
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
	// DiscardData indicates if the scheduled query results should be discarded (true)
	// or kept (false) in a query report.
	DiscardData bool `json:"discard_data" db:"discard_data"`

	/////////////////////////////////////////////////////////////////
	// WARNING: If you add to this struct make sure it's taken into
	// account in the Query Clone implementation!
	/////////////////////////////////////////////////////////////////
}

// Clone implements cloner for Query.
func (q *Query) Clone() (Cloner, error) {
	return q.Copy(), nil
}

// Copy returns a deep copy of the Query.
func (q *Query) Copy() *Query {
	if q == nil {
		return nil
	}

	clone := *q

	if q.TeamID != nil {
		clone.TeamID = ptr.Uint(*q.TeamID)
	}
	if q.AuthorID != nil {
		clone.AuthorID = ptr.Uint(*q.AuthorID)
	}

	if q.Packs != nil {
		clone.Packs = make([]Pack, len(q.Packs))
		for i, p := range q.Packs {
			newP := p.Copy()
			clone.Packs[i] = *newP
		}
	}

	if q.AggregatedStats.SystemTimeP50 != nil {
		clone.AggregatedStats.SystemTimeP50 = ptr.Float64(*q.AggregatedStats.SystemTimeP50)
	}
	if q.AggregatedStats.SystemTimeP95 != nil {
		clone.AggregatedStats.SystemTimeP95 = ptr.Float64(*q.AggregatedStats.SystemTimeP95)
	}
	if q.AggregatedStats.UserTimeP50 != nil {
		clone.AggregatedStats.UserTimeP50 = ptr.Float64(*q.AggregatedStats.UserTimeP50)
	}
	if q.AggregatedStats.UserTimeP95 != nil {
		clone.AggregatedStats.UserTimeP95 = ptr.Float64(*q.AggregatedStats.UserTimeP95)
	}
	if q.AggregatedStats.TotalExecutions != nil {
		clone.AggregatedStats.TotalExecutions = ptr.Float64(*q.AggregatedStats.TotalExecutions)
	}
	return &clone
}

type LiveQueryStats struct {
	// host_id, average_memory, execution, system_time, user_time
	HostID        uint      `db:"host_id"`
	Executions    uint64    `db:"executions"`
	AverageMemory uint64    `db:"average_memory"`
	SystemTime    uint64    `db:"system_time"`
	UserTime      uint64    `db:"user_time"`
	WallTime      uint64    `db:"wall_time"`
	OutputSize    uint64    `db:"output_size"`
	LastExecuted  time.Time `db:"last_executed"`
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
// Called when creating or modifying a query
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
	if q.Platform != nil {
		if err := verifyQueryPlatforms(*q.Platform); err != nil {
			return err
		}
	}
	return nil
}

// Verify verifies the query fields are valid.
// Called when creating queries by spec
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
	if err := verifyQueryPlatforms(q.Platform); err != nil {
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
	errQueryEmptyName       = errors.New("query name cannot be empty")
	errQueryEmptyQuery      = errors.New("query's SQL query cannot be empty")
	ErrQueryInvalidPlatform = errors.New("query's platform must be a comma-separated list of 'darwin', 'linux', 'windows', and/or 'chrome' in a single string")
	errInvalidLogging       = fmt.Errorf("invalid logging value, must be one of '%s', '%s', '%s'", LoggingSnapshot, LoggingDifferential, LoggingDifferentialIgnoreRemovals)
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
	if logging != LoggingSnapshot && logging != LoggingDifferential && logging != LoggingDifferentialIgnoreRemovals {
		return errInvalidLogging
	}
	return nil
}

func verifyQueryPlatforms(platforms string) error {
	if emptyString(platforms) {
		return nil
	}
	platformsList := strings.Split(platforms, ",")
	for _, platform := range platformsList {
		// TODO(jacob) – should we accept these strings with spaces? If not, remove `TrimSpace`
		switch strings.TrimSpace(platform) {
		case "windows", "linux", "darwin", "chrome":
			// OK
		default:
			return ErrQueryInvalidPlatform
		}
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
	// DiscardData indicates if the scheduled query results should be discarded (true)
	// or kept (false) in a query report.
	//
	// If not set, then the default value is false.
	DiscardData bool `json:"discard_data"`
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
	AverageMemory uint64 `json:"average_memory" db:"average_memory"`
	Denylisted    bool   `json:"denylisted" db:"denylisted"`
	Executions    uint64 `json:"executions" db:"executions"`
	// Note schedule_interval is used for DB since "interval" is a reserved word in MySQL
	Interval           int        `json:"interval" db:"schedule_interval"`
	DiscardData        bool       `json:"discard_data" db:"discard_data"`
	LastFetched        *time.Time `json:"last_fetched" db:"last_fetched"`
	AutomationsEnabled bool       `json:"automations_enabled" db:"automations_enabled"`
	LastExecuted       time.Time  `json:"last_executed" db:"last_executed"`
	OutputSize         uint64     `json:"output_size" db:"output_size"`
	SystemTime         uint64     `json:"system_time" db:"system_time"`
	UserTime           uint64     `json:"user_time" db:"user_time"`
	WallTime           uint64     `json:"wall_time" db:"wall_time"`
}

// MapQueryReportsResultsToRows converts the scheduled query results as stored in Fleet's database
// to HostQueryResultRows to be exposed to the API.
func MapQueryReportResultsToRows(rows []*ScheduledQueryResultRow) ([]HostQueryResultRow, error) {
	var results []HostQueryResultRow
	for _, row := range rows {
		var columns map[string]string
		if row.Data == nil {
			continue
		}
		if err := json.Unmarshal(*row.Data, &columns); err != nil {
			return nil, err
		}
		results = append(results, HostQueryResultRow{
			HostID:      row.HostID,
			Hostname:    row.HostDisplayName(),
			LastFetched: row.LastFetched,
			Columns:     columns,
		})
	}
	return results, nil
}

// HostQueryResultRow contains a single scheduled query result row from a host.
// This type is used to expose the results on the API.
type HostQueryResultRow struct {
	// HostID is the unique ID of the host.
	HostID uint `json:"host_id"`
	// Hostname is the host's hostname.
	Hostname string `json:"host_name"`
	// LastFetched is the time this result row was received.
	LastFetched time.Time `json:"last_fetched"`
	// Columns contains the key-value pairs of a result row.
	// The map key is the name of the column, and the map value is the value.
	Columns map[string]string `json:"columns"`
}

type HostQueryReportResult struct {
	// Columns contains the key-value pairs of a result row.
	// The map key is the name of the column, and the map value is the value.
	Columns map[string]string `json:"columns"`
}

// ScheduledQueryResult holds results of a scheduled query received from a osquery agent.
type ScheduledQueryResult struct {
	// QueryName is the name of the query.
	QueryName string `json:"name,omitempty"`
	// OsqueryHostID is the identifier of the host.
	OsqueryHostID string `json:"hostIdentifier"`
	// Snapshot holds the result rows. It's an array of maps, where the map keys
	// are column names and map values are the values.
	Snapshot []*json.RawMessage `json:"snapshot"`
	// LastFetched is the time this result was received.
	UnixTime uint `json:"unixTime"`
}

// ScheduledQueryResultRow is a scheduled query result row.
type ScheduledQueryResultRow struct {
	// QueryID is the unique identifier of the query.
	QueryID uint `db:"query_id"`
	// HostID is the unique identifier of the host.
	HostID uint `db:"host_id"`
	// Hostname is the host's hostname. NullString is used in case host does not exist.
	Hostname sql.NullString `db:"hostname"`
	// ComputerName is the host's computer_name.
	ComputerName sql.NullString `db:"computer_name"`
	// HardwareModel is the host's hardware_model.
	HardwareModel sql.NullString `db:"hardware_model"`
	// HardwareSerial is the host's hardware_serial.
	HardwareSerial sql.NullString `db:"hardware_serial"`
	// Data holds a single result row. It holds a map where the map keys
	// are column names and map values are the values.
	Data *json.RawMessage `db:"data"`
	// LastFetched is the time this result was received.
	LastFetched time.Time `db:"last_fetched"`
}

func (s *ScheduledQueryResultRow) HostDisplayName() string {
	// If host does not exist, all values below default to empty string
	return HostDisplayName(
		s.ComputerName.String, s.Hostname.String,
		s.HardwareModel.String, s.HardwareSerial.String,
	)
}
