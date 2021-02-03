package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/websocket"
	"github.com/igm/sockjs-go/v3/sockjs"
	"github.com/pkg/errors"
)

func (svc service) NewDistributedQueryCampaignByNames(ctx context.Context, queryString string, hosts []string, labels []string) (*kolide.DistributedQueryCampaign, error) {
	hostIDs, err := svc.ds.HostIDsByName(hosts)
	if err != nil {
		return nil, errors.Wrap(err, "finding host IDs")
	}

	labelIDs, err := svc.ds.LabelIDsByName(labels)
	if err != nil {
		return nil, errors.Wrap(err, "finding label IDs")
	}

	return svc.NewDistributedQueryCampaign(ctx, queryString, hostIDs, labelIDs)
}

func uintPtr(n uint) *uint {
	return &n
}

func (svc service) NewDistributedQueryCampaign(ctx context.Context, queryString string, hosts []uint, labels []uint) (*kolide.DistributedQueryCampaign, error) {
	if err := svc.StatusLiveQuery(ctx); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}

	query := &kolide.Query{
		Name:     fmt.Sprintf("distributed_%s_%d", vc.Username(), time.Now().Unix()),
		Query:    queryString,
		Saved:    false,
		AuthorID: uintPtr(vc.UserID()),
	}
	if err := query.ValidateSQL(); err != nil {
		return nil, err
	}
	query, err := svc.ds.NewQuery(query)
	if err != nil {
		return nil, errors.Wrap(err, "new query")
	}

	campaign, err := svc.ds.NewDistributedQueryCampaign(&kolide.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  kolide.QueryWaiting,
		UserID:  vc.UserID(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "new campaign")
	}

	// Add host targets
	for _, hid := range hosts {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			return nil, errors.Wrap(err, "adding host target")
		}
	}

	// Add label targets
	for _, lid := range labels {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			return nil, errors.Wrap(err, "adding label target")
		}
	}

	hostIDs, err := svc.ds.HostIDsInTargets(hosts, labels)
	if err != nil {
		return nil, errors.Wrap(err, "get target IDs")
	}

	err = svc.liveQueryStore.RunQuery(strconv.Itoa(int(campaign.ID)), queryString, hostIDs)
	if err != nil {
		return nil, errors.Wrap(err, "run query")
	}

	campaign.Metrics, err = svc.ds.CountHostsInTargets(hosts, labels, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "counting hosts")
	}
	return campaign, nil
}

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

func (svc service) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	// Find the campaign and ensure it is active
	campaign, err := svc.ds.DistributedQueryCampaign(campaignID)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot find campaign for ID %d", campaignID))
		return
	}

	// Open the channel from which we will receive incoming query results
	// (probably from the redis pubsub implementation)
	readChan, err := svc.resultStore.ReadChannel(context.Background(), *campaign)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot open read channel for campaign %d ", campaignID))
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
		_ = svc.ds.SaveDistributedQueryCampaign(campaign)
		_ = svc.liveQueryStore.StopQuery(strconv.Itoa(int(campaign.ID)))
	}()

	status := campaignStatus{
		Status: campaignStatusPending,
	}
	lastStatus := status
	lastTotals := targetTotals{}

	// to improve performance of the frontend rendering the results table, we
	// add the "host_hostname" field to every row and clean null rows.
	mapHostnameRows := func(res *kolide.DistributedQueryResult) {
		filteredRows := []map[string]string{}
		for _, row := range res.Rows {
			if row == nil {
				continue
			}
			row["host_hostname"] = res.Host.HostName
			filteredRows = append(filteredRows, row)
		}

		res.Rows = filteredRows
	}

	hostIDs, labelIDs, err := svc.ds.DistributedQueryCampaignTargetIDs(campaign.ID)
	if err != nil {
		conn.WriteJSONError("error retrieving campaign targets: " + err.Error())
		return
	}

	updateStatus := func() error {
		metrics, err := svc.CountHostsInTargets(context.Background(), hostIDs, labelIDs)
		if err != nil {
			if err = conn.WriteJSONError("error retrieving target counts"); err != nil {
				return errors.New("retrieve target counts")
			}
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
				return errors.New("write totals")
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
				return errors.New("write status")
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
			case kolide.DistributedQueryResult:
				mapHostnameRows(&res)
				err = conn.WriteJSONMessage("result", res)
				if errors.Cause(err) == sockjs.ErrSessionNotOpen {
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
