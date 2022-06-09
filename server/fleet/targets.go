package fleet

import (
	"encoding/json"
	"fmt"
)

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

func (t TargetType) String() string {
	switch t {
	case TargetLabel:
		return "label"
	case TargetHost:
		return "host"
	case TargetTeam:
		return "team"
	default:
		return fmt.Sprintf("unknown: %d", t)
	}
}

func ParseTargetType(s string) (TargetType, error) {
	switch s {
	case "label":
		return TargetLabel, nil
	case "host":
		return TargetHost, nil
	case "team":
		return TargetTeam, nil
	default:
		return 0, fmt.Errorf("invalid TargetType: %s", s)
	}
}

func (t TargetType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TargetType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := ParseTargetType(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

type Target struct {
	Type        TargetType `db:"type" json:"type"`
	TargetID    uint       `db:"target_id" json:"id"`
	DisplayText string     `db:"display_text" json:"display_text"`
}

func (t Target) AuthzType() string {
	return "target"
}
