package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

type runLiveQueryRequest struct {
	QueryIDs []uint `json:"query_id"`
	HostIDs  []uint `json:"host_ids"`
}

type runLiveQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r runLiveQueryResponse) error() error { return r.Err }

type queryCampaignResult struct {
	QueryID uint          `json:"query_id"`
	Errors  []error       `json:"errors"`
	Results []interface{} `json:"results"`
}

func runLiveQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*runLiveQueryRequest)
	wg := sync.WaitGroup{}

	resultsCh := make(chan queryCampaignResult)

	for _, queryID := range req.QueryIDs {
		queryID := queryID
		wg.Add(1)
		go func() {
			defer wg.Done()
			campaign, err := svc.NewDistributedQueryCampaign(ctx, "", &queryID, fleet.HostTargets{HostIDs: req.HostIDs})
			if err != nil {
				resultsCh <- queryCampaignResult{QueryID: queryID, Errors: []error{err}}
				return
			}

			readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
			defer cancelFunc()
			if err != nil {
				resultsCh <- queryCampaignResult{QueryID: queryID, Errors: []error{err}}
				return
			}

			defer svc.CompleteCampaign(ctx, campaign)

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			var results []interface{}
		loop:
			for {
				select {
				case res := <-readChan:
					// Receive a result and push it over the websocket
					switch res := res.(type) {
					case fleet.DistributedQueryResult:
						results = append(results, res)
					}
				case <-ticker.C:
					break loop
				}
			}
			resultsCh <- queryCampaignResult{QueryID: queryID, Results: results}
		}()
	}

	for result := range resultsCh {
		fmt.Println(result)
	}

	wg.Wait()

	return runLiveQueryResponse{}, nil
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
