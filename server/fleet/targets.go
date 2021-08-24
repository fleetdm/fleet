package fleet

type TargetSearchResults struct {
	Hosts  []*Host
	Labels []*Label
	Teams  []*Team
}

// TargetMetrics contains information about the online status of a set of
// hosts.
type TargetMetrics struct {
	// TotalHosts is the total hosts in any status. It should equal
	// OnlineHosts + OfflineHosts + MissingInActionHosts.
	TotalHosts uint `db:"total"`
	// OnlineHosts is the count of hosts that have checked in within their
	// expected checkin interval (based on the configuration interval
	// values, see Host.Status()).
	OnlineHosts uint `db:"online"`
	// OfflineHosts is the count of hosts that have not checked in within
	// their expected interval.
	OfflineHosts uint `db:"offline"`
	// MissingInActionHosts is the count of hosts that have not checked in
	// within the last 30 days.
	MissingInActionHosts uint `db:"mia"`
	// NewHosts is the count of hosts that have enrolled in the last 24
	// hours.
	NewHosts uint `db:"new"`
}

// HostTargets is the set of targets for a campaign (live query). These
// targets are additive (include all hosts and all hosts in labels and all hosts
// in teams).
type HostTargets struct {
	// HostIDs is the IDs of hosts to be targeted
	HostIDs []uint `json:"hosts"`
	// LabelIDs is the IDs of labels to be targeted
	LabelIDs []uint `json:"labels"`
	// TeamIDs is the IDs of teams to be targeted
	TeamIDs []uint `json:"teams"`
}

type TargetType int

const (
	TargetLabel TargetType = iota
	TargetHost
	TargetTeam
)

type Target struct {
	Type     TargetType `db:"type"`
	TargetID uint       `db:"target_id"`
}

func (t Target) AuthzType() string {
	return "target"
}
