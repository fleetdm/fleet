package service

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/websocket"
	"golang.org/x/net/context"
)

func (svc service) ListQueries(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Query, error) {
	return svc.ds.ListQueries(opt)
}

func (svc service) GetQuery(ctx context.Context, id uint) (*kolide.Query, error) {
	return svc.ds.Query(id)
}

func (svc service) NewQuery(ctx context.Context, p kolide.QueryPayload) (*kolide.Query, error) {
	query := &kolide.Query{}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Description != nil {
		query.Description = *p.Description
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	if p.Interval != nil {
		query.Interval = *p.Interval
	}

	if p.Snapshot != nil {
		query.Snapshot = *p.Snapshot
	}

	if p.Differential != nil {
		query.Differential = *p.Differential
	}

	if p.Platform != nil {
		query.Platform = *p.Platform
	}

	if p.Version != nil {
		query.Version = *p.Version
	}

	query, err := svc.ds.NewQuery(query)
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (svc service) ModifyQuery(ctx context.Context, id uint, p kolide.QueryPayload) (*kolide.Query, error) {
	query, err := svc.ds.Query(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Description != nil {
		query.Description = *p.Description
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	if p.Interval != nil {
		query.Interval = *p.Interval
	}

	if p.Snapshot != nil {
		query.Snapshot = *p.Snapshot
	}

	if p.Differential != nil {
		query.Differential = *p.Differential
	}

	if p.Platform != nil {
		query.Platform = *p.Platform
	}

	if p.Version != nil {
		query.Version = *p.Version
	}

	err = svc.ds.SaveQuery(query)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (svc service) DeleteQuery(ctx context.Context, id uint) error {
	query, err := svc.ds.Query(id)
	if err != nil {
		return err
	}

	err = svc.ds.DeleteQuery(query)
	if err != nil {
		return err
	}

	return nil
}

func (svc service) NewDistributedQueryCampaign(ctx context.Context, queryString string, hosts []uint, labels []uint) (*kolide.DistributedQueryCampaign, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}

	query, err := svc.NewQuery(ctx, kolide.QueryPayload{
		Name:  &queryString,
		Query: &queryString,
	})
	if err != nil {
		return nil, err
	}

	campaign, err := svc.ds.NewDistributedQueryCampaign(&kolide.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  kolide.QueryRunning,
		UserID:  vc.UserID(),
	})
	if err != nil {
		return nil, err
	}

	// Add host targets
	for _, hid := range hosts {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&kolide.DistributedQueryCampaignTarget{
			Type: kolide.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			return nil, err
		}
	}

	// Add label targets
	for _, lid := range labels {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&kolide.DistributedQueryCampaignTarget{
			Type: kolide.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			return nil, err
		}
	}

	return campaign, nil
}

type targetTotals struct {
	Total  uint `json:"count"`
	Online uint `json:"online"`
}

func (svc service) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	// Find the campaign and ensure it is active
	campaign, err := svc.ds.DistributedQueryCampaign(campaignID)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot find campaign for ID %d", campaignID))
		return
	}

	if campaign.Status != kolide.QueryRunning {
		conn.WriteJSONError(fmt.Sprintf("campaign %d not running", campaignID))
		return
	}

	// Open the channel from which we will receive incoming query results
	// (probably from the redis pubsub implementation)
	readChan, err := svc.resultStore.ReadChannel(context.Background(), *campaign)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot open read channel for campaign %d ", campaignID))
		return
	}

	// Loop, pushing updates to results and expected totals
	for {
		select {
		case res := <-readChan:
			// Receive a result and push it over the websocket
			switch res := res.(type) {
			case kolide.DistributedQueryResult:
				err = conn.WriteJSONMessage("result", res)
				if err != nil {
					fmt.Println("error writing to channel")
				}
			}

		case <-time.After(1 * time.Second):
			// Update the expected hosts total
			hostIDs, labelIDs, err := svc.ds.DistributedQueryCampaignTargetIDs(campaign.ID)
			if err != nil {
				if err = conn.WriteJSONError("error retrieving campaign targets"); err != nil {
					return
				}
			}

			var totals targetTotals
			totals.Total, totals.Online, err = svc.CountHostsInTargets(
				context.Background(), hostIDs, labelIDs,
			)
			if err != nil {
				if err = conn.WriteJSONError("error retrieving target counts"); err != nil {
					return
				}
			}

			if err = conn.WriteJSONMessage("totals", totals); err != nil {
				return
			}
		}
	}

}
