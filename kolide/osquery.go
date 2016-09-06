package kolide

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"
)

type OsqueryStore interface {
	LabelQueriesForHost(host *Host, cutoff time.Time) (map[string]string, error)
	RecordLabelQueryExecutions(host *Host, results map[string]bool, t time.Time) error
	NewLabel(label *Label) error
}

type OsqueryService interface {
	EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error)
	GetClientConfig(ctx context.Context, action string, data json.RawMessage) (OsqueryConfig, error)
	GetDistributedQueries(ctx context.Context) (map[string]string, error)
	SubmitDistributedQueryResults(ctx context.Context, results OsqueryDistributedQueryResults) error
	SubmitStatusLogs(ctx context.Context, logs []OsqueryResultLog) error
	SubmitResultsLogs(ctx context.Context, logs []OsqueryStatusLog) error
}

type OsqueryDistributedQueryResults map[string][]map[string]string

type OsqueryConfig struct {
	Packs    []Pack
	Schedule []Query
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

// TODO: move this to just use LabelQueriesForHot
// LabelQueriesForHost calculates the appropriate update cutoff (given
// interval) and uses the datastore to retrieve the label queries for the
// provided host.
func LabelQueriesForHost(store OsqueryStore, host *Host, interval time.Duration) (map[string]string, error) {
	cutoff := time.Now().Add(-interval)
	return store.LabelQueriesForHost(host, cutoff)
}
