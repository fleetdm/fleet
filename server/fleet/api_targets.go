package fleet

import (
	"encoding/json"
	"time"
)

type SearchTargetsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID *uint `json:"query_id" renameto:"report_id"`
	// Selected is the list of IDs that are already selected on the caller side
	// (e.g. the UI), so those are IDs that will be omitted from the returned
	// payload.
	Selected HostTargets `json:"selected"`
}

// LabelSearchResult is a label returned in target search results.
type LabelSearchResult struct {
	*Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

// TeamSearchResult is a team returned in target search results.
type TeamSearchResult struct {
	*Team
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

func (t TeamSearchResult) MarshalJSON() ([]byte, error) {
	x := struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		TeamConfig
		UserCount   int             `json:"user_count"`
		Users       []TeamUser      `json:"users,omitempty"`
		HostCount   int             `json:"host_count"`
		Hosts       []HostResponse  `json:"hosts,omitempty"`
		Secrets     []*EnrollSecret `json:"secrets,omitempty"`
		DisplayText string          `json:"display_text"`
		Count       int             `json:"count"`
	}{
		ID:          t.ID,
		CreatedAt:   t.CreatedAt,
		Name:        t.Name,
		Description: t.Description,
		TeamConfig:  t.Config,
		UserCount:   t.UserCount,
		Users:       t.Users,
		HostCount:   t.HostCount,
		Hosts:       HostResponsesForHostsCheap(t.Hosts),
		Secrets:     t.Secrets,
		DisplayText: t.DisplayText,
		Count:       t.Count,
	}

	return json.Marshal(x)
}

func (t *TeamSearchResult) UnmarshalJSON(b []byte) error {
	var x struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		TeamConfig
		UserCount   int             `json:"user_count"`
		Users       []TeamUser      `json:"users,omitempty"`
		HostCount   int             `json:"host_count"`
		Hosts       []Host          `json:"hosts,omitempty"`
		Secrets     []*EnrollSecret `json:"secrets,omitempty"`
		DisplayText string          `json:"display_text"`
		Count       int             `json:"count"`
	}

	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}

	*t = TeamSearchResult{
		Team: &Team{
			ID:          x.ID,
			CreatedAt:   x.CreatedAt,
			Name:        x.Name,
			Description: x.Description,
			Config:      x.TeamConfig,
			UserCount:   x.UserCount,
			Users:       x.Users,
			HostCount:   x.HostCount,
			Hosts:       x.Hosts,
			Secrets:     x.Secrets,
		},
		DisplayText: x.DisplayText,
		Count:       x.Count,
	}

	return nil
}

// TargetsData holds the search results for hosts, labels, and teams.
type TargetsData struct {
	Hosts  []*HostResponse     `json:"hosts"`
	Labels []LabelSearchResult `json:"labels"`
	Teams  []TeamSearchResult  `json:"teams" renameto:"fleets"`
}

type SearchTargetsResponse struct {
	Targets                *TargetsData `json:"targets,omitempty"`
	TargetsCount           uint         `json:"targets_count"`
	TargetsOnline          uint         `json:"targets_online"`
	TargetsOffline         uint         `json:"targets_offline"`
	TargetsMissingInAction uint         `json:"targets_missing_in_action"`
	Err                    error        `json:"error,omitempty"`
}

func (r SearchTargetsResponse) Error() error { return r.Err }

type CountTargetsRequest struct {
	Selected HostTargets `json:"selected"`
	QueryID  *uint       `json:"query_id" renameto:"report_id"`
}

type CountTargetsResponse struct {
	TargetsCount   uint  `json:"targets_count"`
	TargetsOnline  uint  `json:"targets_online"`
	TargetsOffline uint  `json:"targets_offline"`
	Err            error `json:"error,omitempty"`
}

func (r CountTargetsResponse) Error() error { return r.Err }
