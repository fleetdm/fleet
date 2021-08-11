package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/go-kit/kit/log/level"
	"github.com/igm/sockjs-go/v3/sockjs"
	"github.com/pkg/errors"
)

func (svc Service) NewDistributedQueryCampaignByNames(ctx context.Context, queryString string, queryID *uint, hosts []string, labels []string) (*fleet.DistributedQueryCampaign, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	hostIDs, err := svc.ds.HostIDsByName(filter, hosts)
	if err != nil {
		return nil, errors.Wrap(err, "finding host IDs")
	}

	labelIDs, err := svc.ds.LabelIDsByName(labels)
	if err != nil {
		return nil, errors.Wrap(err, "finding label IDs")
	}

	targets := fleet.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs}
	return svc.NewDistributedQueryCampaign(ctx, queryString, queryID, targets)
}

func (svc Service) NewDistributedQueryCampaign(ctx context.Context, queryString string, queryID *uint, targets fleet.HostTargets) (*fleet.DistributedQueryCampaign, error) {
	if err := svc.StatusLiveQuery(ctx); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	if queryID == nil && queryString == "" {
		return nil, fleet.NewInvalidArgumentError("query", "one of query or query_id must be specified")
	}

	var query *fleet.Query
	var err error
	if queryID != nil {
		query, err = svc.ds.Query(*queryID)
		if err != nil {
			return nil, err
		}
		queryString = query.Query
	} else {
		if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionWrite); err != nil {
			return nil, err
		}
		query = &fleet.Query{
			Name:     fmt.Sprintf("distributed_%s_%d", vc.Email(), time.Now().Unix()),
			Query:    queryString,
			Saved:    false,
			AuthorID: ptr.Uint(vc.UserID()),
		}
		err := query.ValidateSQL()
		if err != nil {
			return nil, err
		}
		query, err = svc.ds.NewQuery(query)
		if err != nil {
			return nil, errors.Wrap(err, "new query")
		}
	}

	if err := svc.authz.Authorize(ctx, query, fleet.ActionRun); err != nil {
		return nil, err
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: query.ObserverCanRun}

	campaign, err := svc.ds.NewDistributedQueryCampaign(&fleet.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  fleet.QueryWaiting,
		UserID:  vc.UserID(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "new campaign")
	}

	defer func() {
		var numHosts uint = 0
		if campaign != nil {
			numHosts = campaign.Metrics.TotalHosts
		}
		logging.WithExtras(ctx, "sql", queryString, "query_id", queryID, "numHosts", numHosts)
	}()

	// Add host targets
	for _, hid := range targets.HostIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			return nil, errors.Wrap(err, "adding host target")
		}
	}

	// Add label targets
	for _, lid := range targets.LabelIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			return nil, errors.Wrap(err, "adding label target")
		}
	}

	// Add team targets
	for _, tid := range targets.TeamIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetTeam,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   tid,
		})
		if err != nil {
			return nil, errors.Wrap(err, "adding team target")
		}
	}

	hostIDs, err := svc.ds.HostIDsInTargets(filter, targets)
	if err != nil {
		return nil, errors.Wrap(err, "get target IDs")
	}

	err = svc.liveQueryStore.RunQuery(strconv.Itoa(int(campaign.ID)), queryString, hostIDs)
	if err != nil {
		return nil, errors.Wrap(err, "run query")
	}

	campaign.Metrics, err = svc.ds.CountHostsInTargets(filter, targets, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "counting hosts")
	}

	if err := svc.ds.NewActivity(
		authz.UserFromContext(ctx),
		fleet.ActivityTypeLiveQuery,
		&map[string]interface{}{"targets_count": campaign.Metrics.TotalHosts},
	); err != nil {
		return nil, err
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

func (svc Service) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	logging.WithExtras(ctx, "campaign_id", campaignID)

	// Explicitly set ObserverCanRun: true in this check because we check that the user trying to
	// read results is the same user that initiated the query. This means the observer check already
	// happened with the actual value for this query.
	if err := svc.authz.Authorize(ctx, &fleet.Query{ObserverCanRun: true}, fleet.ActionRun); err != nil {
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
	campaign, err := svc.ds.DistributedQueryCampaign(campaignID)
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
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	readChan, err := svc.resultStore.ReadChannel(cancelCtx, *campaign)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot open read channel for campaign %d ", campaignID))
		return
	}

	// Setting status to running will cause the query to be returned to the
	// targets when they check in for their queries
	campaign.Status = fleet.QueryRunning
	if err := svc.ds.SaveDistributedQueryCampaign(campaign); err != nil {
		conn.WriteJSONError("error saving campaign state")
		return
	}

	// Setting the status to completed stops the query from being sent to
	// targets. If this fails, there is a background job that will clean up
	// this campaign.
	defer func() {
		campaign.Status = fleet.QueryComplete
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

	targets, err := svc.ds.DistributedQueryCampaignTargetIDs(campaign.ID)
	if err != nil {
		conn.WriteJSONError("error retrieving campaign targets: " + err.Error())
		return
	}

	updateStatus := func() error {
		metrics, err := svc.CountHostsInTargets(ctx, &campaign.QueryID, *targets)
		if err != nil {
			if err = conn.WriteJSONError("error retrieving target counts"); err != nil {
				return errors.Wrap(err, "retrieve target counts, write failed")
			}
			return errors.Wrap(err, "retrieve target counts")
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
				return errors.Wrap(err, "write totals")
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
				return errors.Wrap(err, "write status")
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
