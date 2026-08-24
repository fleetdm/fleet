package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"gopkg.in/guregu/null.v3"
)

/////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries of a team.
/////////////////////////////////////////////////////////////////////////////////

type getTeamScheduleRequest struct {
	TeamID      uint              `url:"fleet_id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type getTeamScheduleResponse struct {
	Scheduled []fleet.ScheduledQueryResponse `json:"scheduled"`
	Err       error                          `json:"error,omitempty"`
}

func (r getTeamScheduleResponse) Error() error { return r.Err }

func getTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getTeamScheduleRequest)
	resp := getTeamScheduleResponse{Scheduled: []fleet.ScheduledQueryResponse{}}
	queries, err := svc.GetTeamScheduledQueries(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return getTeamScheduleResponse{Err: err}, nil
	}
	for _, q := range queries {
		resp.Scheduled = append(resp.Scheduled, fleet.ScheduledQueryResponse{
			ScheduledQuery: *q,
		})
	}
	return resp, nil
}

// The team schedule routes spell the global schedule as fleet_id 0.
func teamIDOrNilForGlobal(teamID uint) *uint {
	if teamID == 0 {
		return nil
	}
	return &teamID
}

func (svc Service) GetTeamScheduledQueries(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	queries, _, _, _, err := svc.ListQueries(ctx, opts, teamIDOrNilForGlobal(teamID), new(true), false, nil)
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
	TeamID uint `url:"fleet_id"`
	fleet.ScheduledQueryPayload
}

type teamScheduleQueryResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r teamScheduleQueryResponse) Error() error { return r.Err }

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

func teamScheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

// scheduledQueryInScope loads scheduledQueryID and rejects it unless it
// belongs to teamID (nil for the global schedule). The schedule endpoints
// carry their scope in the URL path but address the report by a globally
// unique ID, so without this the path scope is decorative: a report can be
// reached through any schedule's URL. A mismatch returns the same not-found
// error as a report that doesn't exist, so probing can't distinguish the two.
func (svc Service) scheduledQueryInScope(ctx context.Context, scheduledQueryID uint, teamID *uint) (*fleet.Query, error) {
	query, err := svc.ds.Query(ctx, scheduledQueryID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get scheduled query")
	}
	if !ptr.Equal(query.TeamID, teamID) {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Report").WithID(scheduledQueryID), "get scheduled query")
	}
	return query, nil
}

func (svc Service) TeamScheduleQuery(ctx context.Context, teamID uint, scheduledQuery *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	// Authorize before loading the source report, so a caller who can't create
	// anything here learns nothing about which source IDs exist.
	if err := svc.authz.Authorize(ctx, fleet.Query{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	// nil scope: only a global report may be used as the source.
	originalQuery, err := svc.scheduledQueryInScope(ctx, scheduledQuery.QueryID, nil)
	if err != nil {
		return nil, err
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
	TeamID           uint `url:"fleet_id"`
	ScheduledQueryID uint `url:"report_id"`
	fleet.ScheduledQueryPayload
}

type modifyTeamScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r modifyTeamScheduleResponse) Error() error { return r.Err }

func modifyTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*modifyTeamScheduleRequest)
	if _, err := svc.ModifyTeamScheduledQueries(ctx, req.TeamID, req.ScheduledQueryID, req.ScheduledQueryPayload); err != nil {
		return modifyTeamScheduleResponse{Err: err}, nil
	}
	return modifyTeamScheduleResponse{}, nil
}

func (svc Service) ModifyTeamScheduledQueries(
	ctx context.Context,
	teamID uint,
	scheduledQueryID uint,
	scheduledQueryPayload fleet.ScheduledQueryPayload,
) (*fleet.ScheduledQuery, error) {
	scoped, err := svc.scheduledQueryInScope(ctx, scheduledQueryID, teamIDOrNilForGlobal(teamID))
	if err != nil {
		return nil, err
	}
	query, err := svc.modifyLoadedQuery(ctx, scoped, fleet.ScheduledQueryPayloadToQueryPayloadForModifyQuery(scheduledQueryPayload))
	if err != nil {
		return nil, err
	}
	return fleet.ScheduledQueryFromQuery(query), nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete a scheduled query from a team.
/////////////////////////////////////////////////////////////////////////////////

type deleteTeamScheduleRequest struct {
	TeamID           uint `url:"fleet_id"`
	ScheduledQueryID uint `url:"report_id"`
}

type deleteTeamScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r deleteTeamScheduleResponse) Error() error { return r.Err }

func deleteTeamScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteTeamScheduleRequest)
	err := svc.DeleteTeamScheduledQueries(ctx, req.TeamID, req.ScheduledQueryID)
	if err != nil {
		return deleteTeamScheduleResponse{Err: err}, nil
	}
	return deleteTeamScheduleResponse{}, nil
}

func (svc Service) DeleteTeamScheduledQueries(ctx context.Context, teamID uint, scheduledQueryID uint) error {
	scoped, err := svc.scheduledQueryInScope(ctx, scheduledQueryID, teamIDOrNilForGlobal(teamID))
	if err != nil {
		return err
	}
	return svc.deleteLoadedQuery(ctx, scoped)
}
