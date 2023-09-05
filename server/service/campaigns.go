package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
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
	fmt.Println("Running createDistributedQueryCampaignEndpoint")
	req := request.(*createDistributedQueryCampaignRequest)
	campaign, err := svc.NewDistributedQueryCampaign(ctx, req.QuerySQL, req.QueryID, req.Selected)
	if err != nil {
		fmt.Println("Error in createDistributedQueryCampaignEndpoint: ", err)
		return createDistributedQueryCampaignResponse{Err: err}, nil
	}
	return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
}

func (svc *Service) NewDistributedQueryCampaign(ctx context.Context, queryString string, queryID *uint, targets fleet.HostTargets) (*fleet.DistributedQueryCampaign, error) {
	fmt.Println("Running NewDistributedQueryCampaign Func")
	if err := svc.StatusLiveQuery(ctx); err != nil {
		fmt.Println("Error in StatusLiveQuery: ", err)
		return nil, err
	}

	fmt.Println("Getting viewer from context")
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		fmt.Println("Error in viewer.FromContext: ", fleet.ErrNoContext)
		return nil, fleet.ErrNoContext
	}

	fmt.Println("Checking queryID == nil && strings.TrimSpace(queryString) == ")
	if queryID == nil && strings.TrimSpace(queryString) == "" {
		fmt.Println("Error in queryID == nil && strings.TrimSpace(queryString) == ")
		return nil, fleet.NewInvalidArgumentError("query", "one of query or query_id must be specified")
	}

	fmt.Println("Checking queryID != nil")
	var query *fleet.Query
	var err error
	if queryID != nil {
		fmt.Println("queryID != nil")
		query, err = svc.ds.Query(ctx, *queryID)
		if err != nil {
			fmt.Println("Error in svc.ds.Query: ", err)
			return nil, err
		}
		queryString = query.Query
	} else {
		fmt.Println("queryID == nil")
		if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRunNew); err != nil {
			fmt.Println("Error in svc.authz.Authorize: ", err)
			return nil, err
		}
		query = &fleet.Query{
			Name:     fmt.Sprintf("distributed_%s_%d", vc.Email(), time.Now().UnixNano()),
			Query:    queryString,
			Saved:    false,
			AuthorID: ptr.Uint(vc.UserID()),
		}
		fmt.Println("Checking query.Verify")
		if err := query.Verify(); err != nil {
			fmt.Println("Error in query.Verify: ", err)
			return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("query payload verification: %s", err),
			})
		}
		fmt.Println("Checking svc.ds.NewQuery")
		query, err = svc.ds.NewQuery(ctx, query)
		if err != nil {
			fmt.Println("Error in svc.ds.NewQuery: ", err)
			return nil, ctxerr.Wrap(ctx, err, "new query")
		}
	}

	fmt.Println("Checking query.ObserverCanRun")
	tq := &fleet.TargetedQuery{Query: query, HostTargets: targets}
	if err := svc.authz.Authorize(ctx, tq, fleet.ActionRun); err != nil {
		fmt.Println("Error in svc.authz.Authorize: ", err)
		return nil, err
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: query.ObserverCanRun}

	fmt.Println("Checking svc.ds.NewDistributedQueryCampaign")
	campaign, err := svc.ds.NewDistributedQueryCampaign(ctx, &fleet.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  fleet.QueryWaiting,
		UserID:  vc.UserID(),
	})
	if err != nil {
		fmt.Println("Error in svc.ds.NewDistributedQueryCampaign: ", err)
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
	fmt.Println("Adding host targets")

	for _, hid := range targets.HostIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			fmt.Println("Error in svc.ds.NewDistributedQueryCampaignTarget: ", err)
			return nil, ctxerr.Wrap(ctx, err, "adding host target")
		}
	}

	// Add label targets
	fmt.Println("Adding label targets")
	for _, lid := range targets.LabelIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			fmt.Println("Error in svc.ds.NewDistributedQueryCampaignTarget: ", err)
			return nil, ctxerr.Wrap(ctx, err, "adding label target")
		}
	}

	// Add team targets
	fmt.Println("Adding team targets")
	for _, tid := range targets.TeamIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetTeam,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   tid,
		})
		if err != nil {
			fmt.Println("Error in svc.ds.NewDistributedQueryCampaignTarget: ", err)
			return nil, ctxerr.Wrap(ctx, err, "adding team target")
		}
	}

	fmt.Println("Checking svc.ds.HostIDsInTargets")
	hostIDs, err := svc.ds.HostIDsInTargets(ctx, filter, targets)
	if err != nil {
		fmt.Println("Error in svc.ds.HostIDsInTargets: ", err)
		return nil, ctxerr.Wrap(ctx, err, "get target IDs")
	}

	if len(hostIDs) == 0 {
		fmt.Println("Error in len(hostIDs) == 0")
		return nil, &fleet.BadRequestError{
			Message: "no hosts targeted",
		}
	}

	fmt.Println("Checking svc.liveQueryStore.RunQuery")
	err = svc.liveQueryStore.RunQuery(strconv.Itoa(int(campaign.ID)), queryString, hostIDs)
	if err != nil {
		fmt.Println("Error in svc.liveQueryStore.RunQuery: ", err)
		return nil, ctxerr.Wrap(ctx, err, "run query")
	}

	fmt.Println("Checking svc.ds.CountHostsInTargets")
	campaign.Metrics, err = svc.ds.CountHostsInTargets(ctx, filter, targets, time.Now())
	if err != nil {
		fmt.Println("Error in svc.ds.CountHostsInTargets: ", err)
		return nil, ctxerr.Wrap(ctx, err, "counting hosts")
	}

	activityData := fleet.ActivityTypeLiveQuery{
		TargetsCount: campaign.Metrics.TotalHosts,
		QuerySQL:     query.Query,
	}
	if queryID != nil {
		activityData.QueryName = &query.Name
	}
	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		activityData,
	); err != nil {
		fmt.Println("Error in svc.ds.NewActivity: ", err)
		return nil, ctxerr.Wrap(ctx, err, "create activity for campaign creation")
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

	labelIDs, err := svc.ds.LabelIDsByName(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding label IDs")
	}

	targets := fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs}
	return svc.NewDistributedQueryCampaign(ctx, queryString, queryID, targets)
}
