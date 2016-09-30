package kolide

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"
)

type OsqueryStore interface {
	// LabelQueriesForHost returns the label queries that should be executed
	// for the given host. The cutoff is the minimum timestamp a query
	// execution should have to be considered "fresh". Executions that are
	// not fresh will be repeated. Results are returned in a map of label
	// id -> query
	LabelQueriesForHost(host *Host, cutoff time.Time) (map[string]string, error)

	// RecordLabelQueryExecutions saves the results of label queries. The
	// results map is a map of label id -> whether or not the label
	// matches. The time parameter is the timestamp to save with the query
	// execution.
	RecordLabelQueryExecutions(host *Host, results map[string]bool, t time.Time) error

	// NewLabel saves a new label.
	NewLabel(label *Label) (*Label, error)

	// LabelsForHost returns the labels that the given host is in.
	LabelsForHost(host *Host) ([]Label, error)
}

type OsqueryService interface {
	EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error)
	AuthenticateHost(ctx context.Context, nodeKey string) (*Host, error)
	GetClientConfig(ctx context.Context) (*OsqueryConfig, error)
	GetDistributedQueries(ctx context.Context) (map[string]string, error)
	SubmitDistributedQueryResults(ctx context.Context, results OsqueryDistributedQueryResults) error
	SubmitStatusLogs(ctx context.Context, logs []OsqueryStatusLog) error
	SubmitResultLogs(ctx context.Context, logs []OsqueryResultLog) error
	SubmitLogs(ctx context.Context, logType string, data *json.RawMessage) error
}

type OsqueryDistributedQueryResults map[string][]map[string]string

type QueryContent struct {
	Query       string `json:"query"`
	Description string `json:"description,omitempty"`
	Interval    uint   `json:"interval"`
	Platform    string `json:"platform,omitempty"`
	Version     string `json:"version,omitempty"`
	Snapshot    bool   `json:"snapshot,omitempty"`
	Removed     bool   `json:"removed,omitempty"`
	Shard       uint   `json:"shard,omitempty"`
}

type Queries map[string]QueryContent

type PackContent struct {
	Platform  string   `json:"platform,omitempty"`
	Version   string   `json:"version,omitempty"`
	Shard     uint     `json:"shard,omitempty"`
	Discovery []string `json:"discovery,omitempty"`
	Queries   Queries  `json:"queries"`
}

type Packs map[string]PackContent

type Options struct {
	PackDelimiter      string `json:"pack_delimiter,omitempty"`
	DisableDistributed bool   `json:"disable_distributed"`
}

type Decorators struct {
	Load     []string            `json:"load,omitempty"`
	Always   []string            `json:"always,omitempty"`
	Interval map[string][]string `json:"interval,omitempty"`
}

type OsqueryConfig struct {
	Options    Options    `json:"options,omitempty"`
	Decorators Decorators `json:"decorators,omitempty"`
	Packs      Packs      `json:"packs,omitempty"`
}

type OsqueryResultLog struct {
	Name           string            `json:"name"`
	HostIdentifier string            `json:"hostIdentifier"`
	UnixTime       string            `json:"unixTime"`
	CalendarTime   string            `json:"calendarTime"`
	Columns        map[string]string `json:"columns"`
	Action         string            `json:"action"`
}

type OsqueryStatusLog struct {
	Severity    string            `json:"severity"`
	Filename    string            `json:"filename"`
	Line        string            `json:"line"`
	Message     string            `json:"message"`
	Version     string            `json:"version"`
	Decorations map[string]string `json:"decorations"`
}
