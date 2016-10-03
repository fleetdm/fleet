package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type LabelStore interface {
	// Label methods
	NewLabel(Label *Label) (*Label, error)
	SaveLabel(Label *Label) error
	DeleteLabel(lid uint) error
	Label(lid uint) (*Label, error)
	Labels() ([]*Label, error)

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

	// LabelsForHost returns the labels that the given host is in.
	LabelsForHost(hid uint) ([]Label, error)
}

type LabelService interface {
	GetAllLabels(ctx context.Context) ([]*Label, error)
	GetLabel(ctx context.Context, id uint) (*Label, error)
	NewLabel(ctx context.Context, p LabelPayload) (*Label, error)
	ModifyLabel(ctx context.Context, id uint, p LabelPayload) (*Label, error)
	DeleteLabel(ctx context.Context, id uint) error
}

type LabelPayload struct {
	Name    *string
	QueryID *uint `json:"query_id"`
}

type Label struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"not null;unique_index:idx_label_unique_name"`
	QueryID   uint
}

type LabelQueryExecution struct {
	ID        uint `gorm:"primary_key"`
	UpdatedAt time.Time
	Matches   bool
	LabelID   uint // Note we manually specify a unique index on these
	HostID    uint // fields in gormDB.Migrate
}
