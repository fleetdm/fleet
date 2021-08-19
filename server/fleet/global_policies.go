package fleet

import (
	"context"
	"time"
)

type GlobalPoliciesService interface {
	NewGlobalPolicy(ctx context.Context, queryID uint) (*Policy, error)
	ListGlobalPolicies(ctx context.Context) ([]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)
}

type GlobalPoliciesStore interface {
	NewGlobalPolicy(queryID uint) (*Policy, error)
	Policy(id uint) (*Policy, error)
	RecordPolicyQueryExecutions(host *Host, results map[uint]*bool, updated time.Time) error

	ListGlobalPolicies() ([]*Policy, error)
	DeleteGlobalPolicies(ids []uint) ([]uint, error)

	PolicyQueriesForHost(host *Host) (map[string]string, error)
}

type Policy struct {
	ID               uint   `json:"id"`
	QueryID          uint   `json:"query_id" db:"query_id"`
	QueryName        string `json:"query_name" db:"query_name"`
	PassingHostCount uint   `json:"passing_host_count"`
	FailingHostCount uint   `json:"failing_host_count"`

	UpdateCreateTimestamps
}

func (Policy) AuthzType() string {
	return "policy"
}
