package service

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/websocket"
	"golang.org/x/net/context"
)

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
		Status:  kolide.QueryWaiting,
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

	if campaign.Status != kolide.QueryWaiting {
		conn.WriteJSONError(fmt.Sprintf("campaign %d not running", campaignID))
		return
	}

	// Setting status to running will cause the query to be returned to the
	// targets when they check in for their queries
	campaign.Status = kolide.QueryRunning
	if err := svc.ds.SaveDistributedQueryCampaign(campaign); err != nil {
		conn.WriteJSONError("error saving campaign state")
		return
	}

	// Setting the status to completed stops the query from being sent to
	// targets. If this fails, there is a background job that will clean up
	// this campaign.
	defer func() {
		campaign.Status = kolide.QueryComplete
		svc.ds.SaveDistributedQueryCampaign(campaign)
	}()

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
