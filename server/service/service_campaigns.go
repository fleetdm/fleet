package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/go-kit/kit/log/level"
	"github.com/igm/sockjs-go/v3/sockjs"
)

type targetTotals struct {
	Total           uint `json:"count"`
	Online          uint `json:"online"`
	Offline         uint `json:"offline"`
	MissingInAction uint `json:"missing_in_action"`
}

const (
	campaignStatusPending  = "pending"
	campaignStatusFinished = "finished"
)

type campaignStatus struct {
	ExpectedResults uint   `json:"expected_results"`
	ActualResults   uint   `json:"actual_results"`
	Status          string `json:"status"`
}

func (svc Service) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	logging.WithExtras(ctx, "campaign_id", campaignID)

	// Explicitly set ObserverCanRun: true in this check because we check that the user trying to
	// read results is the same user that initiated the query. This means the observer check already
	// happened with the actual value for this query.
	if err := svc.authz.Authorize(ctx, &fleet.TargetedQuery{Query: &fleet.Query{ObserverCanRun: true}}, fleet.ActionRun); err != nil {
		level.Info(svc.logger).Log("err", "stream results authorization failed")
		conn.WriteJSONError(authz.ForbiddenErrorMessage)
		return
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		level.Info(svc.logger).Log("err", "stream results viewer missing")
		conn.WriteJSONError(authz.ForbiddenErrorMessage)
		return
	}

	// Find the campaign and ensure it is active
	campaign, err := svc.ds.DistributedQueryCampaign(ctx, campaignID)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot find campaign for ID %d", campaignID))
		return
	}

	// Ensure the same user is opening to read results as initiated the query
	if campaign.UserID != vc.User.ID {
		level.Info(svc.logger).Log(
			"err", "stream results ID does not match",
			"expected", campaign.UserID,
			"got", vc.User.ID,
		)
		conn.WriteJSONError(authz.ForbiddenErrorMessage)
		return
	}

	// Open the channel from which we will receive incoming query results
	// (probably from the redis pubsub implementation)
	readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
	if err != nil {
		conn.WriteJSONError("error getting campaign reader: " + err.Error())
		return
	}
	defer cancelFunc()

	// Setting the status to completed stops the query from being sent to
	// targets. If this fails, there is a background job that will clean up
	// this campaign.
	defer svc.CompleteCampaign(ctx, campaign)

	status := campaignStatus{
		Status: campaignStatusPending,
	}
	lastStatus := status
	lastTotals := targetTotals{}

	// to improve performance of the frontend rendering the results table, we
	// add the "host_hostname" field to every row and clean null rows.
	mapHostnameRows := func(res *fleet.DistributedQueryResult) {
		filteredRows := []map[string]string{}
		for _, row := range res.Rows {
			if row == nil {
				continue
			}
			row["host_hostname"] = res.Host.Hostname
			filteredRows = append(filteredRows, row)
		}

		res.Rows = filteredRows
	}

	targets, err := svc.ds.DistributedQueryCampaignTargetIDs(ctx, campaign.ID)
	if err != nil {
		conn.WriteJSONError("error retrieving campaign targets: " + err.Error())
		return
	}

	updateStatus := func() error {
		metrics, err := svc.CountHostsInTargets(ctx, &campaign.QueryID, *targets)
		if err != nil {
			if err = conn.WriteJSONError("error retrieving target counts"); err != nil {
				return ctxerr.Wrap(ctx, err, "retrieve target counts, write failed")
			}
			return ctxerr.Wrap(ctx, err, "retrieve target counts")
		}

		totals := targetTotals{
			Total:           metrics.TotalHosts,
			Online:          metrics.OnlineHosts,
			Offline:         metrics.OfflineHosts,
			MissingInAction: metrics.MissingInActionHosts,
		}
		if lastTotals != totals {
			lastTotals = totals
			if err = conn.WriteJSONMessage("totals", totals); err != nil {
				return ctxerr.Wrap(ctx, err, "write totals")
			}
		}

		status.ExpectedResults = totals.Online
		if status.ActualResults >= status.ExpectedResults {
			status.Status = campaignStatusFinished
		}
		// only write status message if status has changed
		if lastStatus != status {
			lastStatus = status
			if err = conn.WriteJSONMessage("status", status); err != nil {
				return ctxerr.Wrap(ctx, err, "write status")
			}
		}

		return nil
	}

	if err := updateStatus(); err != nil {
		_ = svc.logger.Log("msg", "error updating status", "err", err)
		return
	}

	// Push status updates every 5 seconds at most
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	// Loop, pushing updates to results and expected totals
	for {
		// Update the expected hosts total (Should happen before
		// any results are written, to avoid the frontend showing "x of
		// 0 Hosts Returning y Records")
		select {
		case res := <-readChan:
			// Receive a result and push it over the websocket
			switch res := res.(type) {
			case fleet.DistributedQueryResult:
				mapHostnameRows(&res)
				err = conn.WriteJSONMessage("result", res)
				if ctxerr.Cause(err) == sockjs.ErrSessionNotOpen {
					// return and stop sending the query if the session was closed
					// by the client
					return
				}
				if err != nil {
					_ = svc.logger.Log("msg", "error writing to channel", "err", err)
				}
				status.ActualResults++
			}

		case <-ticker.C:
			if conn.GetSessionState() == sockjs.SessionClosed {
				// return and stop sending the query if the session was closed
				// by the client
				return
			}
			// Update status
			if err := updateStatus(); err != nil {
				svc.logger.Log("msg", "error updating status", "err", err)
				return
			}
		}
	}
}
