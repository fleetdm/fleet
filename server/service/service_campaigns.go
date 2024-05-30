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
	"github.com/go-kit/kit/log"
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
	statsBatchSize         = 1000
)

type campaignStatus struct {
	ExpectedResults uint   `json:"expected_results"`
	ActualResults   uint   `json:"actual_results"`
	Status          string `json:"status"`
}

type statsToSave struct {
	hostID uint
	*fleet.Stats
	outputSize uint64
}

type statsTracker struct {
	saveStats         bool
	aggregationNeeded bool
	stats             []statsToSave
}

func (svc Service) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	logging.WithExtras(ctx, "campaign_id", campaignID)
	logger := log.With(svc.logger, "campaignID", campaignID)

	// Explicitly set ObserverCanRun: true in this check because we check that the user trying to
	// read results is the same user that initiated the query. This means the observer check already
	// happened with the actual value for this query.
	if err := svc.authz.Authorize(ctx, &fleet.TargetedQuery{Query: &fleet.Query{ObserverCanRun: true}}, fleet.ActionRun); err != nil {
		level.Info(logger).Log("err", "stream results authorization failed")
		conn.WriteJSONError(authz.ForbiddenErrorMessage) //nolint:errcheck
		return
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		level.Info(logger).Log("err", "stream results viewer missing")
		conn.WriteJSONError(authz.ForbiddenErrorMessage) //nolint:errcheck
		return
	}

	// Find the campaign and ensure it is active
	campaign, err := svc.ds.DistributedQueryCampaign(ctx, campaignID)
	if err != nil {
		conn.WriteJSONError(fmt.Sprintf("cannot find campaign for ID %d", campaignID)) //nolint:errcheck
		return
	}

	// Ensure the same user is opening to read results as initiated the query
	if campaign.UserID != vc.User.ID {
		level.Info(logger).Log(
			"err", "campaign user ID does not match",
			"expected", campaign.UserID,
			"got", vc.User.ID,
		)
		conn.WriteJSONError(authz.ForbiddenErrorMessage) //nolint:errcheck
		return
	}

	// Open the channel from which we will receive incoming query results
	// (probably from the redis pubsub implementation)
	readChan, cancelFunc, err := svc.GetCampaignReader(ctx, campaign)
	if err != nil {
		conn.WriteJSONError("error getting campaign reader: " + err.Error()) //nolint:errcheck
		return
	}
	defer cancelFunc()

	// Setting the status to completed stops the query from being sent to
	// targets. If this fails, there is a background job that will clean up
	// this campaign.
	defer func() {
		// We do not want to use the outer `ctx` because we want to make sure
		// to cleanup the campaign.
		ctx := context.WithoutCancel(ctx)
		if err := svc.CompleteCampaign(ctx, campaign); err != nil {
			level.Error(logger).Log("msg", "complete campaign (async)", "err", err)
		}
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
			row["host_display_name"] = res.Host.DisplayName
			filteredRows = append(filteredRows, row)
		}

		res.Rows = filteredRows
	}

	targets, err := svc.ds.DistributedQueryCampaignTargetIDs(ctx, campaign.ID)
	if err != nil {
		conn.WriteJSONError("error retrieving campaign targets: " + err.Error()) //nolint:errcheck
		return
	}

	updateStatus := func() error {
		metrics, err := svc.CountHostsInTargets(ctx, &campaign.QueryID, *targets)
		if err != nil {
			if err := conn.WriteJSONError("error retrieving target counts"); err != nil {
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
			if err := conn.WriteJSONMessage("totals", totals); err != nil {
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
			if err := conn.WriteJSONMessage("status", status); err != nil {
				return ctxerr.Wrap(ctx, err, "write status")
			}
		}

		return nil
	}

	if err := updateStatus(); err != nil {
		_ = logger.Log("msg", "error updating status", "err", err)
		return
	}

	// Push status updates every 5 seconds at most
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// We process stats along with results as they are sent back to the user.
	// We do a batch update of the stats.
	// We update aggregated stats once online hosts have reported, and again (if needed) on client disconnect.
	perfStatsTracker := statsTracker{}
	perfStatsTracker.saveStats, err = svc.ds.IsSavedQuery(ctx, campaign.QueryID)
	if err != nil {
		level.Error(logger).Log("msg", "error checking saved query", "query.id", campaign.QueryID, "err", err)
		perfStatsTracker.saveStats = false
	}
	// We aggregate stats and add activity at the end. Using context without cancel for precaution.
	queryID := campaign.QueryID
	ctxWithoutCancel := context.WithoutCancel(ctx)
	defer func() {
		svc.updateStats(ctxWithoutCancel, queryID, logger, &perfStatsTracker, true)
		svc.addLiveQueryActivity(ctxWithoutCancel, lastTotals.Total, queryID, logger)
	}()

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
				// Calculate result size for performance stats
				outputSize := calculateOutputSize(&perfStatsTracker, &res)
				mapHostnameRows(&res)
				err = conn.WriteJSONMessage("result", res)
				if perfStatsTracker.saveStats && res.Stats != nil {
					perfStatsTracker.stats = append(
						perfStatsTracker.stats, statsToSave{hostID: res.Host.ID, Stats: res.Stats, outputSize: outputSize},
					)
					if len(perfStatsTracker.stats) >= statsBatchSize {
						svc.updateStats(ctx, campaign.QueryID, logger, &perfStatsTracker, false)
					}
				}
				if ctxerr.Cause(err) == sockjs.ErrSessionNotOpen {
					// return and stop sending the query if the session was closed
					// by the client
					return
				}
				if err != nil {
					_ = level.Error(logger).Log("msg", "error writing to channel", "err", err)
				}
				status.ActualResults++
			case error:
				level.Error(logger).Log("msg", "received error from pubsub channel", "err", res)
				if err := conn.WriteJSONError("pubsub error: " + res.Error()); err != nil {
					logger.Log("msg", "failed to write pubsub error", "err", err)
				}
			}

		case <-ticker.C:
			if conn.GetSessionState() == sockjs.SessionClosed {
				// return and stop sending the query if the session was closed
				// by the client
				return
			}
			// Update status
			if err := updateStatus(); err != nil {
				level.Error(logger).Log("msg", "error updating status", "err", err)
				return
			}
			if status.ActualResults == status.ExpectedResults {
				// We update stats when all expected results come in.
				// The WebSockets connection can remain open indefinitely, so we make sure we update the stats at this critical point.
				svc.updateStats(ctx, campaign.QueryID, logger, &perfStatsTracker, true)
			}
		}
	}
}

// addLiveQueryActivity adds live query activity to the activity feed, including the updated aggregated stats
func (svc Service) addLiveQueryActivity(
	ctx context.Context, targetsCount uint, queryID uint, logger log.Logger,
) {
	activityData := fleet.ActivityTypeLiveQuery{
		TargetsCount: targetsCount,
	}
	// Query returns SQL, name, and aggregated stats
	q, err := svc.ds.Query(ctx, queryID)
	if err != nil {
		level.Error(logger).Log("msg", "error getting query", "id", queryID, "err", err)
	} else {
		activityData.QuerySQL = q.Query
		if q.Saved {
			activityData.QueryName = &q.Name
			activityData.Stats = &q.AggregatedStats
		}
	}
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		activityData,
	); err != nil {
		level.Error(logger).Log("msg", "error creating activity for live query", "err", err)
	}
}

func calculateOutputSize(perfStatsTracker *statsTracker, res *fleet.DistributedQueryResult) uint64 {
	outputSize := uint64(0)
	// We only need the output size if other stats are present.
	if perfStatsTracker.saveStats && res.Stats != nil {
		for _, row := range res.Rows {
			if row == nil {
				continue
			}
			for key, value := range row {
				outputSize = outputSize + uint64(len(key)) + uint64(len(value))
			}
		}
	}
	return outputSize
}

func (svc Service) updateStats(
	ctx context.Context, queryID uint, logger log.Logger, tracker *statsTracker, aggregateStats bool,
) {
	// If we are not saving stats
	if tracker == nil || !tracker.saveStats ||
		// Or there are no stats to save, and we don't need to calculate aggregated stats
		(len(tracker.stats) == 0 && (!aggregateStats || !tracker.aggregationNeeded)) {
		return
	}

	if len(tracker.stats) > 0 {
		// Get the existing stats from DB
		hostIDs := []uint{}
		for i := range tracker.stats {
			hostIDs = append(hostIDs, tracker.stats[i].hostID)
		}
		currentStats, err := svc.ds.GetLiveQueryStats(ctx, queryID, hostIDs)
		if err != nil {
			level.Error(logger).Log("msg", "error getting current live query stats", "err", err)
			tracker.saveStats = false
			return
		}
		// Convert current Stats to a map
		statsMap := make(map[uint]*fleet.LiveQueryStats)
		for i := range currentStats {
			statsMap[currentStats[i].HostID] = currentStats[i]
		}

		// Update stats
		for _, gatheredStats := range tracker.stats {
			stats, ok := statsMap[gatheredStats.hostID]
			if !ok {
				newStats := fleet.LiveQueryStats{
					HostID:        gatheredStats.hostID,
					Executions:    1,
					AverageMemory: gatheredStats.Memory,
					SystemTime:    gatheredStats.SystemTime,
					UserTime:      gatheredStats.UserTime,
					WallTime:      gatheredStats.WallTimeMs,
					OutputSize:    gatheredStats.outputSize,
				}
				currentStats = append(currentStats, &newStats)
			} else {
				// Combine old and new stats.
				stats.AverageMemory = (stats.AverageMemory*stats.Executions + gatheredStats.Memory) / (stats.Executions + 1)
				stats.Executions = stats.Executions + 1
				stats.SystemTime = stats.SystemTime + gatheredStats.SystemTime
				stats.UserTime = stats.UserTime + gatheredStats.UserTime
				stats.WallTime = stats.WallTime + gatheredStats.WallTimeMs
				stats.OutputSize = stats.OutputSize + gatheredStats.outputSize
			}
		}

		// Insert/overwrite updated stats
		err = svc.ds.UpdateLiveQueryStats(ctx, queryID, currentStats)
		if err != nil {
			level.Error(logger).Log("msg", "error updating live query stats", "err", err)
			tracker.saveStats = false
			return
		}

		tracker.aggregationNeeded = true
		tracker.stats = nil
	}

	// Do aggregation
	if aggregateStats && tracker.aggregationNeeded {
		err := svc.ds.CalculateAggregatedPerfStatsPercentiles(ctx, fleet.AggregatedStatsTypeScheduledQuery, queryID)
		if err != nil {
			level.Error(logger).Log("msg", "error aggregating performance stats", "err", err)
			tracker.saveStats = false
			return
		}
		tracker.aggregationNeeded = false
	}
	return
}
