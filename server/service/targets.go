package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Search Targets
////////////////////////////////////////////////////////////////////////////////

type searchTargetsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID *uint `json:"query_id"`
	// Selected is the list of IDs that are already selected on the caller side
	// (e.g. the UI), so those are IDs that will be omitted from the returned
	// payload.
	Selected fleet.HostTargets `json:"selected"`
}

type labelSearchResult struct {
	*fleet.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

type teamSearchResult struct {
	*fleet.Team
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

func (t teamSearchResult) MarshalJSON() ([]byte, error) {
	x := struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		fleet.TeamConfig
		UserCount   int                   `json:"user_count"`
		Users       []fleet.TeamUser      `json:"users,omitempty"`
		HostCount   int                   `json:"host_count"`
		Hosts       []fleet.HostResponse  `json:"hosts,omitempty"`
		Secrets     []*fleet.EnrollSecret `json:"secrets,omitempty"`
		DisplayText string                `json:"display_text"`
		Count       int                   `json:"count"`
	}{
		ID:          t.ID,
		CreatedAt:   t.CreatedAt,
		Name:        t.Name,
		Description: t.Description,
		TeamConfig:  t.Config,
		UserCount:   t.UserCount,
		Users:       t.Users,
		HostCount:   t.HostCount,
		Hosts:       fleet.HostResponsesForHostsCheap(t.Hosts),
		Secrets:     t.Secrets,
		DisplayText: t.DisplayText,
		Count:       t.Count,
	}

	return json.Marshal(x)
}

func (t *teamSearchResult) UnmarshalJSON(b []byte) error {
	var x struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		fleet.TeamConfig
		UserCount   int                   `json:"user_count"`
		Users       []fleet.TeamUser      `json:"users,omitempty"`
		HostCount   int                   `json:"host_count"`
		Hosts       []fleet.Host          `json:"hosts,omitempty"`
		Secrets     []*fleet.EnrollSecret `json:"secrets,omitempty"`
		DisplayText string                `json:"display_text"`
		Count       int                   `json:"count"`
	}

	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}

	*t = teamSearchResult{
		Team: &fleet.Team{
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

type targetsData struct {
	Hosts  []*fleet.HostResponse `json:"hosts"`
	Labels []labelSearchResult   `json:"labels"`
	Teams  []teamSearchResult    `json:"teams"`
}

type searchTargetsResponse struct {
	Targets                *targetsData `json:"targets,omitempty"`
	TargetsCount           uint         `json:"targets_count"`
	TargetsOnline          uint         `json:"targets_online"`
	TargetsOffline         uint         `json:"targets_offline"`
	TargetsMissingInAction uint         `json:"targets_missing_in_action"`
	Err                    error        `json:"error,omitempty"`
}

func (r searchTargetsResponse) Error() error { return r.Err }

func searchTargetsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*searchTargetsRequest)

	results, err := svc.SearchTargets(ctx, req.MatchQuery, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	targets := &targetsData{
		Hosts:  []*fleet.HostResponse{},
		Labels: []labelSearchResult{},
		Teams:  []teamSearchResult{},
	}

	for _, host := range results.Hosts {
		targets.Hosts = append(targets.Hosts, fleet.HostResponseForHostCheap(host))
	}

	for _, label := range results.Labels {
		targets.Labels = append(targets.Labels,
			labelSearchResult{
				Label:       label,
				DisplayText: label.Name,
				Count:       label.HostCount,
			},
		)
	}

	for _, team := range results.Teams {
		targets.Teams = append(targets.Teams,
			teamSearchResult{
				Team:        team,
				DisplayText: team.Name,
				Count:       team.HostCount,
			},
		)
	}

	metrics, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	return searchTargetsResponse{
		Targets:                targets,
		TargetsCount:           metrics.TotalHosts,
		TargetsOnline:          metrics.OnlineHosts,
		TargetsOffline:         metrics.OfflineHosts,
		TargetsMissingInAction: metrics.MissingInActionHosts,
	}, nil
}

func (svc *Service) SearchTargets(ctx context.Context, matchQuery string, queryID *uint, targets fleet.HostTargets) (*fleet.TargetSearchResults, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Target{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := &fleet.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(ctx, filter, matchQuery, targets.HostIDs...)
	if err != nil {
		return nil, err
	}

	results.Hosts = append(results.Hosts, hosts...)

	labels, err := svc.ds.SearchLabels(ctx, filter, matchQuery, targets.LabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	teams, err := svc.ds.SearchTeams(ctx, filter, matchQuery, targets.TeamIDs...)
	if err != nil {
		return nil, err
	}
	results.Teams = teams

	return results, nil
}

func (svc *Service) CountHostsInTargets(ctx context.Context, queryID *uint, targets fleet.HostTargets) (*fleet.TargetMetrics, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Target{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	metrics, err := svc.ds.CountHostsInTargets(ctx, filter, targets, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

type countTargetsRequest struct {
	Selected fleet.HostTargets `json:"selected"`
	QueryID  *uint             `json:"query_id"`
}

type countTargetsResponse struct {
	TargetsCount   uint  `json:"targets_count"`
	TargetsOnline  uint  `json:"targets_online"`
	TargetsOffline uint  `json:"targets_offline"`
	Err            error `json:"error,omitempty"`
}

func (r countTargetsResponse) Error() error { return r.Err }

func countTargetsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*countTargetsRequest)

	counts, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	return countTargetsResponse{
		TargetsCount:   counts.TotalHosts,
		TargetsOnline:  counts.OnlineHosts,
		TargetsOffline: counts.OfflineHosts,
	}, nil
}
