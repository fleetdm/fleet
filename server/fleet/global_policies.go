package fleet

type Policy struct {
	ID               uint   `json:"id"`
	QueryID          uint   `json:"query_id" db:"query_id"`
	QueryName        string `json:"query_name" db:"query_name"`
	PassingHostCount uint   `json:"passing_host_count" db:"passing_host_count"`
	FailingHostCount uint   `json:"failing_host_count" db:"failing_host_count"`
	TeamID           *uint  `json:"team_id" db:"team_id"`

	UpdateCreateTimestamps
}

func (p Policy) AuthzType() string {
	return "policy"
}

type HostPolicy struct {
	ID        uint   `json:"id" db:"id"`
	QueryID   uint   `json:"query_id" db:"query_id"`
	QueryName string `json:"query_name" db:"query_name"`
	Response  string `json:"response" db:"response"`
}
