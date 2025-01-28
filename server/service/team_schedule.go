package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"gopkg.in/guregu/null.v3"
)

/////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries of a team.
/////////////////////////////////////////////////////////////////////////////////

type getTeamScheduleRequest struct {
	TeamID      uint              `url:"team_id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type getTeamScheduleResponse struct {
	Scheduled []scheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r getTeamScheduleResponse) error() error { return r.Err }

func getTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getTeamScheduleRequest)
	resp := getTeamScheduleResponse{Scheduled: []scheduledQueryResponse{}}
	queries, err := svc.GetTeamScheduledQueries(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return getTeamScheduleResponse{Err: err}, nil
	}
	for _, q := range queries {
		resp.Scheduled = append(resp.Scheduled, scheduledQueryResponse{
			ScheduledQuery: *q,
		})
	}
	return resp, nil
}

func (svc Service) GetTeamScheduledQueries(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	var teamID_ *uint
	if teamID != 0 {
		teamID_ = &teamID
	}
	queries, _, _, err := svc.ListQueries(ctx, opts, teamID_, ptr.Bool(true), false, nil)
	if err != nil {
		return nil, err
	}
	scheduledQueries := make([]*fleet.ScheduledQuery, 0, len(queries))
	for _, query := range queries {
		scheduledQueries = append(scheduledQueries, fleet.ScheduledQueryFromQuery(query))
	}
	return scheduledQueries, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Add schedule query to a team.
/////////////////////////////////////////////////////////////////////////////////

type teamScheduleQueryRequest struct {
	TeamID uint `url:"team_id"`
	fleet.ScheduledQueryPayload
}

type teamScheduleQueryResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r teamScheduleQueryResponse) error() error { return r.Err }

func uintValueOrZero(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func nullIntToPtrUint(v *null.Int) *uint {
	if v == nil {
		return nil
	}
	return ptr.Uint(uint(v.ValueOrZero())) //nolint:gosec // dismiss G115
}

func teamScheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*teamScheduleQueryRequest)
	resp, err := svc.TeamScheduleQuery(ctx, req.TeamID, &fleet.ScheduledQuery{
		QueryID:  uintValueOrZero(req.QueryID),
		Interval: uintValueOrZero(req.Interval),
		Snapshot: req.Snapshot,
		Removed:  req.Removed,
		Platform: req.Platform,
		Version:  req.Version,
		Shard:    nullIntToPtrUint(req.Shard),
	})
	if err != nil {
		return teamScheduleQueryResponse{Err: err}, nil
	}
	return teamScheduleQueryResponse{
		Scheduled: resp,
	}, nil
}

func nameForCopiedQuery(originalName string) string {
	return "Copy of " + originalName + " (" + fmt.Sprintf("%d", time.Now().Unix()) + ")"
}

func (svc Service) TeamScheduleQuery(ctx context.Context, teamID uint, scheduledQuery *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	originalQuery, err := svc.ds.Query(ctx, scheduledQuery.QueryID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get query from id")
	}
	if originalQuery.TeamID != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.New(ctx, "cannot create a team schedule from a team query")
	}
	originalQuery.Name = nameForCopiedQuery(originalQuery.Name)
	originalQuery.TeamID = &teamID
	newQuery, err := svc.NewQuery(ctx, fleet.ScheduledQueryToQueryPayloadForNewQuery(originalQuery, scheduledQuery))
	if err != nil {
		return nil, err
	}
	return fleet.ScheduledQueryFromQuery(newQuery), nil
}

/////////////////////////////////////////////////////////////////////////////////
// Modify team scheduled query.
/////////////////////////////////////////////////////////////////////////////////

type modifyTeamScheduleRequest struct {
	TeamID           uint `url:"team_id"`
	ScheduledQueryID uint `url:"scheduled_query_id"`
	fleet.ScheduledQueryPayload
}

type modifyTeamScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r modifyTeamScheduleResponse) error() error { return r.Err }

func modifyTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyTeamScheduleRequest)
	if _, err := svc.ModifyTeamScheduledQueries(ctx, req.TeamID, req.ScheduledQueryID, req.ScheduledQueryPayload); err != nil {
		return modifyTeamScheduleResponse{Err: err}, nil
	}
	return modifyTeamScheduleResponse{}, nil
}

// TODO(lucas): Document new behavior.
// teamID is not used because of mismatch between old internal representation and API.
func (svc Service) ModifyTeamScheduledQueries(
	ctx context.Context,
	teamID uint,
	scheduledQueryID uint,
	scheduledQueryPayload fleet.ScheduledQueryPayload,
) (*fleet.ScheduledQuery, error) {
	query, err := svc.ModifyQuery(ctx, scheduledQueryID, fleet.ScheduledQueryPayloadToQueryPayloadForModifyQuery(scheduledQueryPayload))
	if err != nil {
		return nil, err
	}
	return fleet.ScheduledQueryFromQuery(query), nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete a scheduled query from a team.
/////////////////////////////////////////////////////////////////////////////////

type deleteTeamScheduleRequest struct {
	TeamID           uint `url:"team_id"`
	ScheduledQueryID uint `url:"scheduled_query_id"`
}

type deleteTeamScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r deleteTeamScheduleResponse) error() error { return r.Err }

func deleteTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteTeamScheduleRequest)
	err := svc.DeleteTeamScheduledQueries(ctx, req.TeamID, req.ScheduledQueryID)
	if err != nil {
		return deleteTeamScheduleResponse{Err: err}, nil
	}
	return deleteTeamScheduleResponse{}, nil
}

// TODO(lucas): Document new behavior.
// teamID is not used because of mismatch between old internal representation and API.
func (svc Service) DeleteTeamScheduledQueries(ctx context.Context, teamID uint, scheduledQueryID uint) error {
	return svc.DeleteQueryByID(ctx, scheduledQueryID)
}
