package chart

import "time"

// MostIgnoredPolicy represents one row in the "most-ignored policies" snapshot.
// Ranked descending by FailingHostCount, this answers "which policies are getting
// the most failures right now" org-wide.
//
// TeamName is the display name of the scoping fleet, or empty when the policy
// is global (TeamID nil). The frontend can render "Global" in that case.
type MostIgnoredPolicy struct {
	PolicyID         uint   `json:"policy_id"`
	Name             string `json:"name"`
	TeamID           *uint  `json:"team_id"`
	TeamName         string `json:"team_name"`
	FailingHostCount int    `json:"failing_host_count"`
}

// TeamCompliance represents one row in the team compliance leaderboard. The
// headline metric is FullyCompliantPct: the fraction of hosts in the team that
// are NOT failing any policy. HostsFailingAny is the raw numerator for drill-in.
// FailingHostCount is the total count of (host, policy) failure pairs within
// the team — useful as a secondary signal.
type TeamCompliance struct {
	TeamID            *uint   `json:"team_id"`
	Name              string  `json:"name"`
	HostCount         int     `json:"host_count"`
	HostsFailingAny   int     `json:"hosts_failing_any"`
	FullyCompliantPct float64 `json:"fully_compliant_pct"`
	PoliciesTracked   int     `json:"policies_tracked"`
	PoliciesFailing   int     `json:"policies_failing"`
}

// PolicyFailingSnapshot pairs a policy_id with its current failing-host bitmap.
// Returned by the datastore for the in-memory compositions used by the
// leaderboard and most-ignored views.
type PolicyFailingSnapshot struct {
	PolicyID   uint
	HostBitmap []byte
}

// HostTeam is a (host_id, team_id) pair used to group hosts by team. TeamID is
// nil for hosts not assigned to any team ("no team" bucket in the leaderboard).
type HostTeam struct {
	HostID uint
	TeamID *uint
}

// PolicyMeta is lightweight policy metadata used to render the most-ignored
// policies view without a second round trip.
type PolicyMeta struct {
	ID     uint
	Name   string
	TeamID *uint
}

// TeamMeta is minimal team data used to render the leaderboard.
type TeamMeta struct {
	ID   uint
	Name string
}

// NoTeamBucketKey is the sentinel map key used in TeamTrendPoint.Counts for
// hosts whose team_id is NULL. Chosen so it cannot collide with a numeric
// team_id converted with strconv.
const NoTeamBucketKey = "no_team"

// TeamTrendPoint is a single day in the per-team stacked-bar trend response.
// Counts maps team_id (as string, with NoTeamBucketKey for the nil-team bucket)
// to the number of hosts in that team failing at least one policy on that day.
type TeamTrendPoint struct {
	Timestamp time.Time      `json:"timestamp"`
	Counts    map[string]int `json:"counts"`
}

// HostFailingSummary represents one row in the "most non-compliant hosts"
// snapshot: a host plus the number of policies it is currently failing.
// Ranked descending by FailingPolicyCount.
type HostFailingSummary struct {
	HostID             uint   `json:"host_id"`
	Hostname           string `json:"hostname"`
	ComputerName       string `json:"computer_name"`
	TeamID             *uint  `json:"team_id"`
	TeamName           string `json:"team_name"`
	FailingPolicyCount int    `json:"failing_policy_count"`
}
