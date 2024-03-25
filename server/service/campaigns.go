package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignRequest struct {
	QuerySQL string            `json:"query"`
	QueryID  *uint             `json:"query_id"`
	Selected fleet.HostTargets `json:"selected"`
}

type createDistributedQueryCampaignResponse struct {
	Campaign *fleet.DistributedQueryCampaign `json:"campaign,omitempty"`
	Err      error                           `json:"error,omitempty"`
}

func (r createDistributedQueryCampaignResponse) error() error { return r.Err }

func createDistributedQueryCampaignEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createDistributedQueryCampaignRequest)
	campaign, err := svc.NewDistributedQueryCampaign(ctx, req.QuerySQL, req.QueryID, req.Selected)
	if err != nil {
		return createDistributedQueryCampaignResponse{Err: err}, nil
	}
	return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
}

func (svc *Service) NewDistributedQueryCampaign(ctx context.Context, queryString string, queryID *uint, targets fleet.HostTargets) (*fleet.DistributedQueryCampaign, error) {
	if err := svc.StatusLiveQuery(ctx); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	if queryID == nil && strings.TrimSpace(queryString) == "" {
		return nil, fleet.NewInvalidArgumentError("query", "one of query or query_id must be specified")
	}

	var query *fleet.Query
	var err error
	if queryID != nil {
		query, err = svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		queryString = query.Query
	} else {
		if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRunNew); err != nil {
			return nil, err
		}
		query = &fleet.Query{
			Name:     fmt.Sprintf("distributed_%s_%d", vc.Email(), time.Now().UnixNano()),
			Query:    queryString,
			Saved:    false,
			AuthorID: ptr.Uint(vc.UserID()),
			// We must set a valid value for this field, even if unused by live queries.
			Logging: fleet.LoggingSnapshot,
		}
		if err := query.Verify(); err != nil {
			return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("query payload verification: %s", err),
			})
		}
		query, err = svc.ds.NewQuery(ctx, query)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "new query")
		}
	}

	tq := &fleet.TargetedQuery{Query: query, HostTargets: targets}
	if err := svc.authz.Authorize(ctx, tq, fleet.ActionRun); err != nil {
		return nil, err
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: query.ObserverCanRun}

	campaign, err := svc.ds.NewDistributedQueryCampaign(ctx, &fleet.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  fleet.QueryWaiting,
		UserID:  vc.UserID(),
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new campaign")
	}

	defer func() {
		var numHosts uint
		if campaign != nil {
			numHosts = campaign.Metrics.TotalHosts
		}
		logging.WithExtras(ctx, "sql", queryString, "query_id", queryID, "numHosts", numHosts)
	}()

	// Add host targets
	for _, hid := range targets.HostIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding host target")
		}
	}

	// Add label targets
	for _, lid := range targets.LabelIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding label target")
		}
	}

	// Add team targets
	for _, tid := range targets.TeamIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetTeam,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   tid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding team target")
		}
	}

	hostIDs, err := svc.ds.HostIDsInTargets(ctx, filter, targets)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get target IDs")
	}

	if len(hostIDs) == 0 {
		return nil, &fleet.BadRequestError{
			Message: "no hosts targeted",
		}
	}

	// Metrics are used for total hosts targeted for the activity feed.
	campaign.Metrics, err = svc.ds.CountHostsInTargets(ctx, filter, targets, time.Now())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "counting hosts")
	}

	err = svc.liveQueryStore.RunQuery(strconv.Itoa(int(campaign.ID)), queryString, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "run query")
	}

	return campaign, nil
}

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign By Names
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignByNamesRequest struct {
	QuerySQL string                                 `json:"query"`
	QueryID  *uint                                  `json:"query_id"`
	Selected distributedQueryCampaignTargetsByNames `json:"selected"`
}

type distributedQueryCampaignTargetsByNames struct {
	Labels []string `json:"labels"`
	Hosts  []string `json:"hosts"`
}

func createDistributedQueryCampaignByNamesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createDistributedQueryCampaignByNamesRequest)
	campaign, err := svc.NewDistributedQueryCampaignByNames(ctx, req.QuerySQL, req.QueryID, req.Selected.Hosts, req.Selected.Labels)
	if err != nil {
		return createDistributedQueryCampaignResponse{Err: err}, nil
	}
	return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
}

func (svc *Service) NewDistributedQueryCampaignByNames(ctx context.Context, queryString string, queryID *uint, hosts []string, labels []string) (*fleet.DistributedQueryCampaign, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	hostIDs, err := svc.ds.HostIDsByName(ctx, filter, hosts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding host IDs")
	}

	labelMap, err := svc.ds.LabelIDsByName(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding label IDs")
	}

	var labelIDs []uint
	for _, labelID := range labelMap {
		labelIDs = append(labelIDs, labelID)
	}

	targets := fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs}
	return svc.NewDistributedQueryCampaign(ctx, queryString, queryID, targets)
}
