package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
)

type runLiveQueryRequest struct {
	QueryIDs []uint `json:"query_ids"`
	HostIDs  []uint `json:"host_ids"`
}

type runOneLiveQueryRequest struct {
	QueryID uint   `url:"id"`
	HostIDs []uint `json:"host_ids"`
}

type runLiveQueryOnHostRequest struct {
	Identifier string `url:"identifier"`
	Query      string `json:"query"`
}

type runLiveQueryOnHostByIDRequest struct {
	HostID uint   `url:"id"`
	Query  string `json:"query"`
}

type summaryPayload struct {
	TargetedHostCount  int `json:"targeted_host_count"`
	RespondedHostCount int `json:"responded_host_count"`
}

type runLiveQueryResponse struct {
	Summary summaryPayload `json:"summary"`
	Err     error          `json:"error,omitempty"`

	Results []fleet.QueryCampaignResult `json:"live_query_results"`
}

func (r runLiveQueryResponse) error() error { return r.Err }

type runOneLiveQueryResponse struct {
	QueryID            uint                `json:"query_id"`
	TargetedHostCount  int                 `json:"targeted_host_count"`
	RespondedHostCount int                 `json:"responded_host_count"`
	Results            []fleet.QueryResult `json:"results"`
	Err                error               `json:"error,omitempty"`
}

func (r runOneLiveQueryResponse) error() error { return r.Err }

type runLiveQueryOnHostResponse struct {
	HostID uint                `json:"host_id"`
	Rows   []map[string]string `json:"rows"`
	Query  string              `json:"query"`
	Status fleet.HostStatus    `json:"status"`
	Error  string              `json:"error,omitempty"`
}

func (r runLiveQueryOnHostResponse) error() error { return nil }

func runOneLiveQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runOneLiveQueryRequest)

	// Only allow a host to be specified once in HostIDs
	hostIDs := server.RemoveDuplicatesFromSlice(req.HostIDs)

	campaignResults, respondedHostCount, err := runLiveQuery(ctx, svc, []uint{req.QueryID}, "", hostIDs)
	if err != nil {
		return nil, err
	}
	//goland:noinspection GoPreferNilSlice -- use an empty slice here so that API returns an empty array if there are no results
	queryResults := []fleet.QueryResult{}
	if len(campaignResults) > 0 {
		if campaignResults[0].Err != nil {
			return nil, campaignResults[0].Err
		}
		if campaignResults[0].Results != nil {
			queryResults = campaignResults[0].Results
		}
	}

	res := runOneLiveQueryResponse{
		QueryID:            req.QueryID,
		TargetedHostCount:  len(hostIDs),
		RespondedHostCount: respondedHostCount,
		Results:            queryResults,
	}
	return res, nil
}

func runLiveQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runLiveQueryRequest)

	// Only allow a query to be specified once
	queryIDs := server.RemoveDuplicatesFromSlice(req.QueryIDs)
	// Only allow a host to be specified once in HostIDs
	hostIDs := server.RemoveDuplicatesFromSlice(req.HostIDs)

	queryResults, respondedHostCount, err := runLiveQuery(ctx, svc, queryIDs, "", hostIDs)
	if err != nil {
		return nil, err
	}

	res := runLiveQueryResponse{
		Summary: summaryPayload{
			TargetedHostCount:  len(hostIDs),
			RespondedHostCount: respondedHostCount,
		},
		Results: queryResults,
	}
	return res, nil
}

func runLiveQueryOnHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runLiveQueryOnHostRequest)

	host, err := svc.HostLiteByIdentifier(ctx, req.Identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, badRequest(fmt.Sprintf("host not found: %s: %s", req.Identifier, err.Error())))
	}

	return runLiveQueryOnHost(svc, ctx, host, req.Query)
}

func runLiveQueryOnHostByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runLiveQueryOnHostByIDRequest)

	host, err := svc.HostLiteByID(ctx, req.HostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, badRequest(fmt.Sprintf("host not found: %d: %s", req.HostID, err.Error())))
	}

	return runLiveQueryOnHost(svc, ctx, host, req.Query)
}

func runLiveQueryOnHost(svc fleet.Service, ctx context.Context, host *fleet.HostLite, query string) (errorer, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ctxerr.Wrap(ctx, badRequest("query is required"))
	}

	res := runLiveQueryOnHostResponse{
		HostID: host.ID,
		Query:  query,
	}

	status := (&fleet.Host{
		DistributedInterval: host.DistributedInterval,
		ConfigTLSRefresh:    host.ConfigTLSRefresh,
		SeenTime:            host.SeenTime,
	}).Status(time.Now())
	switch status {
	case fleet.StatusOnline, fleet.StatusNew:
		res.Status = fleet.StatusOnline
	case fleet.StatusOffline, fleet.StatusMIA, fleet.StatusMissing:
		res.Status = fleet.StatusOffline
		return res, nil
	default:
		return nil, fmt.Errorf("unknown host status: %s", status)
	}

	queryResults, _, err := runLiveQuery(ctx, svc, []uint{0}, query, []uint{host.ID})
	if err != nil {
		return nil, err
	}

	if len(queryResults) > 0 {
		var err error
		if queryResults[0].Err != nil { //nolint:gocritic // ignore ifelseChain
			err = queryResults[0].Err
		} else if len(queryResults[0].Results) > 0 {
			queryResult := queryResults[0].Results[0]
			if queryResult.Error != nil {
				err = errors.New(*queryResult.Error)
			}
			res.Rows = queryResult.Rows
			res.HostID = queryResult.HostID
		} else {
			err = errors.New("timeout waiting for results")
		}
		if err != nil {
			res.Error = err.Error()
		}
	}
	return res, nil
}

func runLiveQuery(ctx context.Context, svc fleet.Service, queryIDs []uint, query string, hostIDs []uint) (
	[]fleet.QueryCampaignResult, int, error,
) {
	// The period used here should always be less than the request timeout for any load
	// balancer/proxy between Fleet and the API client.
	period := os.Getenv("FLEET_LIVE_QUERY_REST_PERIOD")
	if period == "" {
		period = "25s"
	}
	duration, err := time.ParseDuration(period)
	if err != nil {
		duration = 25 * time.Second
		logging.WithExtras(ctx, "live_query_rest_period_err", err)
	}

	queryResults, respondedHostCount, err := svc.RunLiveQueryDeadline(ctx, queryIDs, query, hostIDs, duration)
	if err != nil {
		return nil, 0, err
	}
	// Check if all query results were forbidden due to lack of authorization.
	allResultsForbidden := len(queryResults) > 0 && respondedHostCount == 0
	if allResultsForbidden {
		for _, r := range queryResults {
			if r.Error == nil || *r.Error != authz.ForbiddenErrorMessage {
				allResultsForbidden = false
				break
			}
		}
	}
	if allResultsForbidden {
		return nil, 0, authz.ForbiddenWithInternal(
			"All Live Query results were forbidden.", authz.UserFromContext(ctx), nil, nil,
		)
	}
	return queryResults, respondedHostCount, nil
}

func (svc *Service) RunLiveQueryDeadline(
	ctx context.Context, queryIDs []uint, query string, hostIDs []uint, deadline time.Duration,
) ([]fleet.QueryCampaignResult, int, error) {
	if len(queryIDs) == 0 || len(hostIDs) == 0 {
		svc.authz.SkipAuthorization(ctx)
		return nil, 0, ctxerr.Wrap(ctx, badRequest("query_ids and host_ids are required"))
	}
	wg := sync.WaitGroup{}

	resultsCh := make(chan fleet.QueryCampaignResult)

	counterMutex := sync.Mutex{}
	respondedHostIDs := make(map[uint]struct{})

	for _, queryID := range queryIDs {
		queryID := queryID
		wg.Add(1)
		go func() {
			defer wg.Done()
			queryIDPtr := &queryID
			queryString := ""
			// 0 is a special ID that indicates we should use raw SQL query instead
			if queryID == 0 {
				queryIDPtr = nil
				queryString = query
			}

			campaign, err := svc.NewDistributedQueryCampaign(ctx, queryString, queryIDPtr, fleet.HostTargets{HostIDs: hostIDs})
			if err != nil {
				level.Error(svc.logger).Log(
					"msg", "new distributed query campaign",
					"queryString", queryString,
					"queryID", queryID,
					"err", err,
				)
				resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error()), Err: err}
				return
			}
			queryID = campaign.QueryID

			// We do not want to use the outer `ctx` directly because we want to cleanup the campaign
			// even if the outer `ctx` is canceled (e.g. a client terminating the connection).
			// Also, we make sure stats and activity DB operations don't get killed after we return results.
			ctxWithoutCancel := context.WithoutCancel(ctx)
			defer func() {
				err := svc.CompleteCampaign(ctxWithoutCancel, campaign)
				if err != nil {
					level.Error(svc.logger).Log(
						"msg", "completing campaign (sync)", "query.id", campaign.QueryID, "campaign.id", campaign.ID, "err", err,
					)
					resultsCh <- fleet.QueryCampaignResult{
						QueryID: queryID,
						Error:   ptr.String(err.Error()),
						Err:     err,
					}
				}
			}()

			readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
			if err != nil {
				level.Error(svc.logger).Log(
					"msg", "get campaign reader", "query.id", campaign.QueryID, "campaign.id", campaign.ID, "err", err,
				)
				resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error()), Err: err}
				return
			}
			defer cancelFunc()

			var results []fleet.QueryResult
			timeout := time.After(deadline)

			// We process stats along with results as they are sent back to the user.
			// We do a batch update of the stats.
			// We update aggregated stats once online hosts have reported.
			const statsBatchSize = 1000
			perfStatsTracker := statsTracker{}
			perfStatsTracker.saveStats, err = svc.ds.IsSavedQuery(ctx, campaign.QueryID)
			if err != nil {
				level.Error(svc.logger).Log("msg", "error checking saved query", "query.id", campaign.QueryID, "err", err)
				perfStatsTracker.saveStats = false
			}
			totalHosts := campaign.Metrics.TotalHosts
			// We update aggregated stats and activity at the end asynchronously.
			defer func() {
				go func() {
					svc.updateStats(ctxWithoutCancel, queryID, svc.logger, &perfStatsTracker, true)
					svc.addLiveQueryActivity(ctxWithoutCancel, totalHosts, queryID, svc.logger)
				}()
			}()
		loop:
			for {
				select {
				case res := <-readChan:
					switch res := res.(type) {
					case fleet.DistributedQueryResult:
						results = append(results, fleet.QueryResult{HostID: res.Host.ID, Rows: res.Rows, Error: res.Error})
						counterMutex.Lock()
						respondedHostIDs[res.Host.ID] = struct{}{}
						counterMutex.Unlock()
						if perfStatsTracker.saveStats && res.Stats != nil {
							perfStatsTracker.stats = append(
								perfStatsTracker.stats,
								statsToSave{
									hostID: res.Host.ID, Stats: res.Stats, outputSize: calculateOutputSize(&perfStatsTracker, &res),
								},
							)
							if len(perfStatsTracker.stats) >= statsBatchSize {
								svc.updateStats(ctx, campaign.QueryID, svc.logger, &perfStatsTracker, false)
							}
						}
						if len(results) == len(hostIDs) {
							break loop
						}
					case error:
						resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(res.Error()), Err: res}
						return
					}
				case <-timeout:
					break loop
				case <-ctx.Done():
					break loop
				}
			}
			resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Results: results}
		}()
	}

	// Iterate collecting results until all the goroutines have returned
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var results []fleet.QueryCampaignResult
	for result := range resultsCh {
		results = append(results, result)
	}

	return results, len(respondedHostIDs), nil
}

func (svc *Service) GetCampaignReader(ctx context.Context, campaign *fleet.DistributedQueryCampaign) (<-chan interface{}, context.CancelFunc, error) {
	// Open the channel from which we will receive incoming query results
	// (probably from the redis pubsub implementation)
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	readChan, err := svc.resultStore.ReadChannel(cancelCtx, *campaign)
	if err != nil {
		cancelFunc()
		return nil, nil, fmt.Errorf("cannot open read channel for campaign %d ", campaign.ID)
	}

	campaign.Status = fleet.QueryRunning
	if err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign); err != nil {
		cancelFunc()
		return nil, nil, ctxerr.Wrap(ctx, err, "error saving campaign state")
	}

	return readChan, cancelFunc, nil
}

func (svc *Service) CompleteCampaign(ctx context.Context, campaign *fleet.DistributedQueryCampaign) error {
	campaign.Status = fleet.QueryComplete
	err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving distributed campaign after complete")
	}
	err = svc.liveQueryStore.StopQuery(fmt.Sprint(campaign.ID))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "stopping query after after complete")
	}
	return nil
}
