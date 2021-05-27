package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/websocket"
)

func (mw loggingMiddleware) NewDistributedQueryCampaign(ctx context.Context, querySQL string, queryID *uint, hosts []uint, labels []uint) (*kolide.DistributedQueryCampaign, error) {
	var (
		loggedInUser = "unauthenticated"
		campaign     *kolide.DistributedQueryCampaign
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		var numHosts uint = 0
		if campaign != nil {
			numHosts = campaign.Metrics.TotalHosts
		}
		_ = mw.loggerInfo(err).Log(
			"method", "NewDistributedQueryCampaign",
			"err", err,
			"user", loggedInUser,
			"sql", querySQL,
			"query_id", queryID,
			"numHosts", numHosts,
			"took", time.Since(begin),
		)
	}(time.Now())
	campaign, err = mw.Service.NewDistributedQueryCampaign(ctx, querySQL, queryID, hosts, labels)
	return campaign, err
}

func (mw loggingMiddleware) NewDistributedQueryCampaignByNames(ctx context.Context, querySQL string, queryID *uint, hosts []string, labels []string) (*kolide.DistributedQueryCampaign, error) {
	var (
		loggedInUser = "unauthenticated"
		campaign     *kolide.DistributedQueryCampaign
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		var numHosts uint = 0
		if campaign != nil {
			numHosts = campaign.Metrics.TotalHosts
		}
		_ = mw.loggerInfo(err).Log(
			"method", "NewDistributedQueryCampaignByNames",
			"err", err,
			"user", loggedInUser,
			"sql", querySQL,
			"query_id", queryID,
			"numHosts", numHosts,
			"took", time.Since(begin),
		)
	}(time.Now())
	campaign, err = mw.Service.NewDistributedQueryCampaignByNames(ctx, querySQL, queryID, hosts, labels)
	return campaign, err
}

func (mw loggingMiddleware) StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint) {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "StreamCampaignResults",
			"campaignID", campaignID,
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	mw.Service.StreamCampaignResults(ctx, conn, campaignID)
}
