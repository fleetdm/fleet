package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type QueryStore interface {
	// Query methods
	NewQuery(query *Query) (*Query, error)
	SaveQuery(query *Query) error
	DeleteQuery(query *Query) error
	Query(id uint) (*Query, error)
	ListQueries(opt ListOptions) ([]*Query, error)
}

type QueryService interface {
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, id uint) error
}

type QueryPayload struct {
	Name         *string
	Query        *string
	Interval     *uint
	Snapshot     *bool
	Differential *bool
	Platform     *string
	Version      *string
}

type PackPayload struct {
	Name     *string
	Platform *string
}

type Query struct {
	ID           uint      `json:"id" gorm:"primary_key"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
	Name         string    `json:"name" gorm:"not null;unique_index:idx_query_unique_name"`
	Query        string    `json:"query" gorm:"not null"`
	Interval     uint      `json:"interval"`
	Snapshot     bool      `json:"snapshot"`
	Differential bool      `json:"differential"`
	Platform     string    `json:"platform"`
	Version      string    `json:"version"`
}

type DistributedQueryStatus int

const (
	QueryRunning  DistributedQueryStatus = iota
	QueryComplete DistributedQueryStatus = iota
	QueryError    DistributedQueryStatus = iota
)

type DistributedQueryCampaign struct {
	ID          uint `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	QueryID     uint
	MaxDuration time.Duration
	Status      DistributedQueryStatus
	UserID      uint
}

type DistributedQueryCampaignTarget struct {
	ID                         uint `gorm:"primary_key"`
	Type                       TargetType
	DistributedQueryCampaignID uint
	TargetID                   uint
}

type DistributedQueryExecutionStatus int

const (
	ExecutionWaiting DistributedQueryExecutionStatus = iota
	ExecutionRequested
	ExecutionSucceeded
	ExecutionFailed
)

type DistributedQueryExecution struct {
	HostID             uint
	DistributedQueryID uint
	Status             DistributedQueryExecutionStatus
	Error              string `gorm:"size:1024"`
	ExecutionDuration  time.Duration
}

type Option struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Key       string `gorm:"not null;unique_index:idx_option_unique_key"`
	Value     string `gorm:"not null"`
	Platform  string
}

type DecoratorType int

const (
	DecoratorLoad DecoratorType = iota
	DecoratorAlways
	DecoratorInterval
)

type Decorator struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      DecoratorType `gorm:"not null"`
	Interval  int
	Query     string
}
