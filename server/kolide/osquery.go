package kolide

import (
	"golang.org/x/net/context"
)

type OsqueryService interface {
	EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error)
	AuthenticateHost(ctx context.Context, nodeKey string) (*Host, error)
	GetClientConfig(ctx context.Context) (*OsqueryConfig, error)
	GetDistributedQueries(ctx context.Context) (map[string]string, error)
	SubmitDistributedQueryResults(ctx context.Context, results OsqueryDistributedQueryResults) error
	SubmitStatusLogs(ctx context.Context, logs []OsqueryStatusLog) error
	SubmitResultLogs(ctx context.Context, logs []OsqueryResultLog) error
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

type OsqueryOptions struct {
	PackDelimiter      string `json:"pack_delimiter,omitempty"`
	DisableDistributed bool   `json:"disable_distributed"`
}

type Decorators struct {
	Load     []string            `json:"load,omitempty"`
	Always   []string            `json:"always,omitempty"`
	Interval map[string][]string `json:"interval,omitempty"`
}

type OsqueryConfig struct {
	Options    OsqueryOptions `json:"options,omitempty"`
	Decorators Decorators     `json:"decorators,omitempty"`
	Packs      Packs          `json:"packs,omitempty"`
}

type OsqueryResultLog struct {
	Name           string            `json:"name"`
	HostIdentifier string            `json:"hostIdentifier"`
	UnixTime       string            `json:"unixTime"`
	CalendarTime   string            `json:"calendarTime"`
	Columns        map[string]string `json:"columns"`
	Action         string            `json:"action"`
	Decorations    map[string]string `json:"decorations"`
}

type OsqueryStatusLog struct {
	Severity    string            `json:"severity"`
	Filename    string            `json:"filename"`
	Line        string            `json:"line"`
	Message     string            `json:"message"`
	Version     string            `json:"version"`
	Decorations map[string]string `json:"decorations"`
}
