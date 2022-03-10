package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

type cleanupsJobStats struct {
	StartedAt        time.Time `json:"started_at" db:"started_at"`
	CompletedAt      time.Time `json:"completed_at" db:"completed_at"`
	TotalRunTime     string    `json:"total_run_time" db:"total_run_time"`
	ExpiredCampaigns uint      `json:"expired_campaigns" db:"expired_campaigns"`
	ExpiredCarves    int       `json:"expired_carves" db:"expired_carves"`
}

func DoCleanups(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, license *fleet.LicenseInfo) (interface{}, error) {
	stats := make(map[string]string)
	startedAt := time.Now()

	expiredCampaigns, err := ds.CleanupDistributedQueryCampaigns(ctx, time.Now())
	if err != nil {
		level.Error(logger).Log("err", "cleaning distributed query campaigns", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.CleanupIncomingHosts(ctx, time.Now())
	if err != nil {
		level.Error(logger).Log("err", "cleaning incoming hosts", "details", err)
		sentry.CaptureException(err)
	}
	expiredCarves, err := ds.CleanupCarves(ctx, time.Now())
	if err != nil {
		level.Error(logger).Log("err", "cleaning carves", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.UpdateQueryAggregatedStats(ctx)
	if err != nil {
		level.Error(logger).Log("err", "aggregating query stats", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.UpdateScheduledQueryAggregatedStats(ctx)
	if err != nil {
		level.Error(logger).Log("err", "aggregating scheduled query stats", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.CleanupExpiredHosts(ctx)
	if err != nil {
		level.Error(logger).Log("err", "cleaning expired hosts", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.GenerateAggregatedMunkiAndMDM(ctx)
	if err != nil {
		level.Error(logger).Log("err", "aggregating munki and mdm data", "details", err)
		sentry.CaptureException(err)
	}
	err = ds.CleanupPolicyMembership(ctx, time.Now())
	if err != nil {
		level.Error(logger).Log("err", "cleanup policy membership", "details", err)
		sentry.CaptureException(err)
	}

	err = trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", license)
	if err != nil {
		level.Error(logger).Log("err", "sending statistics", "details", err)
		sentry.CaptureException(err)
	}

	level.Debug(logger).Log("loop", "done")

	jobStats := &cleanupsJobStats{
		StartedAt:        startedAt,
		CompletedAt:      time.Now(),
		TotalRunTime:     fmt.Sprint(time.Now().Sub(startedAt)),
		ExpiredCampaigns: expiredCampaigns,
		ExpiredCarves:    expiredCarves,
	}
	statsData, err := json.Marshal(jobStats)
	if err != nil {
		level.Error(logger).Log("msg", "marshalling cleanups job stats", "err", err)
		sentry.CaptureException(err)
	}
	stats["do_cleanups"] = string(statsData)

	return stats, nil // TODO
}

func trySendStatistics(ctx context.Context, ds fleet.Datastore, frequency time.Duration, url string, license *fleet.LicenseInfo) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.ServerSettings.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(ctx, frequency, license)
	if err != nil {
		return err
	}
	if !shouldSend {
		return nil
	}

	err = server.PostJSONWithTimeout(ctx, url, stats)
	if err != nil {
		return err
	}
	return ds.RecordStatisticsSent(ctx)
}
