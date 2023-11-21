package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

type runLiveQueryRequest struct {
	QueryIDs []uint `json:"query_ids"`
	HostIDs  []uint `json:"host_ids"`
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

func runLiveQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runLiveQueryRequest)

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

	// Only allow a host to be specified once in HostIDs
	req.HostIDs = server.RemoveDuplicatesFromSlice(req.HostIDs)
	res := runLiveQueryResponse{
		Summary: summaryPayload{
			TargetedHostCount:  len(req.HostIDs),
			RespondedHostCount: 0,
		},
	}

	queryResults, respondedHostCount, err := svc.RunLiveQueryDeadline(ctx, req.QueryIDs, req.HostIDs, duration)
	if err != nil {
		return nil, err
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
		return nil, authz.ForbiddenWithInternal("All Live Query results were forbidden.", authz.UserFromContext(ctx), nil, nil)
	}
	res.Results = queryResults
	res.Summary.RespondedHostCount = respondedHostCount

	return res, nil
}

func (svc *Service) RunLiveQueryDeadline(
	ctx context.Context, queryIDs []uint, hostIDs []uint, deadline time.Duration,
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
			campaign, err := svc.NewDistributedQueryCampaign(ctx, "", &queryID, fleet.HostTargets{HostIDs: hostIDs})
			if err != nil {
				resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				return
			}

			readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
			if err != nil {
				resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				return
			}
			defer cancelFunc()

			defer func() {
				err := svc.CompleteCampaign(ctx, campaign)
				if err != nil {
					resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				}
			}()

			var results []fleet.QueryResult
			timeout := time.After(deadline)
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
					case error:
						resultsCh <- fleet.QueryCampaignResult{QueryID: queryID, Error: ptr.String(res.Error())}
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
	err = svc.liveQueryStore.StopQuery(strconv.Itoa(int(campaign.ID)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "stopping query after after complete")
	}
	return nil
}
