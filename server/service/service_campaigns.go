package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
	lastStatsEntry    *fleet.LiveQueryStats
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

	// Find the campaign and ensure it is active.
	// Since we are reading from the replica DB, the campaign may not be found until it is replicated from the master.
	done := make(chan error, 1)
	stop := make(chan struct{}, 1)
	var campaign *fleet.DistributedQueryCampaign
	go func() {
		var err error
		for {
			select {
			case <-stop:
				return
			default:
				campaign, err = svc.ds.DistributedQueryCampaign(ctx, campaignID)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						time.Sleep(30 * time.Millisecond) // We see the replication time less than 30 ms in production.
						continue
					}
					done <- err
					return
				}
				done <- nil
				return
			}
		}
	}()
	select {
	case err := <-done:
		if err != nil {
			_ = conn.WriteJSONError(fmt.Sprintf("cannot find campaign for ID %d", campaignID)) //nolint:errcheck
			return
		}
	case <-time.After(5 * time.Second):
		stop <- struct{}{}
		_ = conn.WriteJSONError(fmt.Sprintf("timeout: cannot find campaign for ID %d", campaignID)) //nolint:errcheck
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

// overwriteLastExecuted is used for testing purposes to overwrite the last executed time of the live query stats.
var overwriteLastExecuted = false
var overwriteLastExecutedTime time.Time

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
		// We round to the nearest second because MySQL default precision of TIMESTAMP is 1 second.
		// We could alter the table to increase precision. However, this precision granularity is sufficient for the live query stats use case.
		lastExecuted := time.Now().Round(time.Second)
		if overwriteLastExecuted {
			lastExecuted = overwriteLastExecutedTime
		}
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
					LastExecuted:  lastExecuted,
				}
				currentStats = append(currentStats, &newStats)
			} else {
				// Combine old and new stats.
				stats.AverageMemory = (stats.AverageMemory*stats.Executions + gatheredStats.Memory) / (stats.Executions + 1)
				stats.Executions++
				stats.SystemTime += gatheredStats.SystemTime
				stats.UserTime += gatheredStats.UserTime
				stats.WallTime += gatheredStats.WallTimeMs
				stats.OutputSize += gatheredStats.outputSize
				stats.LastExecuted = lastExecuted
			}
		}

		// Insert/overwrite updated stats
		err = svc.ds.UpdateLiveQueryStats(ctx, queryID, currentStats)
		if err != nil {
			level.Error(logger).Log("msg", "error updating live query stats", "err", err)
			tracker.saveStats = false
			return
		}

		if len(currentStats) > 0 {
			tracker.lastStatsEntry = currentStats[0]
		}
		tracker.aggregationNeeded = true
		tracker.stats = nil
	}

	// Do aggregation
	if aggregateStats && tracker.aggregationNeeded {
		// Since we just wrote new stats, we need the write data to sync to the replica before calculating aggregated stats.
		// The calculations are done on the replica to reduce the load on the master.
		// Although this check is not necessary if replica is not used, we leave it in for consistency and to ensure the code is exercised in dev/test environments.
		// To sync with the replica, we read the last stats entry from the replica and compare the timestamp to what was written on the master.
		if tracker.lastStatsEntry != nil { // This check is just to be safe. It should never be nil.
			done := make(chan error, 1)
			stop := make(chan struct{}, 1)
			go func() {
				var stats []*fleet.LiveQueryStats
				var err error
				for {
					select {
					case <-stop:
						return
					default:
						stats, err = svc.ds.GetLiveQueryStats(ctx, queryID, []uint{tracker.lastStatsEntry.HostID})
						if err != nil {
							done <- err
							return
						}
						if !(len(stats) == 0 || stats[0].LastExecuted.Before(tracker.lastStatsEntry.LastExecuted)) {
							// Replica is in sync with the last query stats update
							done <- nil
							return
						}
						time.Sleep(30 * time.Millisecond) // We see the replication time less than 30 ms in production.
					}
				}
			}()
			select {
			case err := <-done:
				if err != nil {
					level.Error(logger).Log("msg", "error syncing replica to master", "err", err)
					tracker.saveStats = false
					return
				}
			case <-time.After(5 * time.Second):
				stop <- struct{}{}
				level.Error(logger).Log("msg", "replica sync timeout: replica did not catch up to the master in 5 seconds")
				// We proceed with the aggregation even if the replica is not in sync.
			}
		}

		err := svc.ds.CalculateAggregatedPerfStatsPercentiles(ctx, fleet.AggregatedStatsTypeScheduledQuery, queryID)
		if err != nil {
			level.Error(logger).Log("msg", "error aggregating performance stats", "err", err)
			tracker.saveStats = false
			return
		}
		tracker.aggregationNeeded = false
	}
}
