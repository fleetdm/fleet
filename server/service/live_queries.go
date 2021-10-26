package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/pkg/errors"
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

	Results []queryCampaignResult `json:"live_query_results"`
}

func (r runLiveQueryResponse) error() error { return r.Err }

type queryResult struct {
	HostID uint                `json:"host_id"`
	Rows   []map[string]string `json:"rows"`
	Error  *string             `json:"error"`
}

type queryCampaignResult struct {
	QueryID uint          `json:"query_id"`
	Error   *string       `json:"error,omitempty"`
	Results []queryResult `json:"results"`
}

func runLiveQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*runLiveQueryRequest)
	wg := sync.WaitGroup{}

	resultsCh := make(chan queryCampaignResult)

	counterMutex := sync.Mutex{}
	counter := make(map[uint]struct{})

	period := os.Getenv("FLEET_LIVE_QUERY_REST_PERIOD")
	if period == "" {
		period = "90s"
	}
	duration, err := time.ParseDuration(period)
	if err != nil {
		duration = 90 * time.Second
	}

	for _, queryID := range req.QueryIDs {
		queryID := queryID
		wg.Add(1)
		go func() {
			defer wg.Done()
			campaign, err := svc.NewDistributedQueryCampaign(ctx, "", &queryID, fleet.HostTargets{HostIDs: req.HostIDs})
			if err != nil {
				resultsCh <- queryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				return
			}

			readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
			if err != nil {
				resultsCh <- queryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				return
			}
			defer cancelFunc()

			defer func() {
				err := svc.CompleteCampaign(ctx, campaign)
				if err != nil {
					resultsCh <- queryCampaignResult{QueryID: queryID, Error: ptr.String(err.Error())}
				}
			}()

			ticker := time.NewTicker(duration)
			defer ticker.Stop()

			var results []queryResult
		loop:
			for {
				select {
				case res := <-readChan:
					// Receive a result and push it over the websocket
					switch res := res.(type) {
					case fleet.DistributedQueryResult:
						results = append(results, queryResult{HostID: res.Host.ID, Rows: res.Rows, Error: res.Error})
						counterMutex.Lock()
						counter[res.Host.ID] = struct{}{}
						counterMutex.Unlock()
					}
				case <-ticker.C:
					break loop
				}
			}
			resultsCh <- queryCampaignResult{QueryID: queryID, Results: results}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	res := runLiveQueryResponse{
		Summary: summaryPayload{
			TargetedHostCount:  len(req.HostIDs),
			RespondedHostCount: 0,
		},
	}

	for result := range resultsCh {
		res.Results = append(res.Results, result)
	}

	res.Summary.RespondedHostCount = len(counter)

	return res, nil
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

	// Setting status to running will cause the query to be returned to the
	// targets when they check in for their queries
	campaign.Status = fleet.QueryRunning
	if err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign); err != nil {
		cancelFunc()
		return nil, nil, errors.Wrap(err, "error saving campaign state")
	}

	return readChan, cancelFunc, nil
}

func (svc *Service) CompleteCampaign(ctx context.Context, campaign *fleet.DistributedQueryCampaign) error {
	campaign.Status = fleet.QueryComplete
	err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign)
	if err != nil {
		return errors.Wrap(err, "saving distributed campaign after complete")
	}
	err = svc.liveQueryStore.StopQuery(strconv.Itoa(int(campaign.ID)))
	if err != nil {
		return errors.Wrap(err, "stopping query after after complete")
	}
	return nil
}
