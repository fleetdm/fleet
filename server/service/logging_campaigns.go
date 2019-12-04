package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/websocket"
)

func (mw loggingMiddleware) NewDistributedQueryCampaign(ctx context.Context, queryString string, hosts []uint, labels []uint) (*kolide.DistributedQueryCampaign, error) {
	var (
		loggedInUser = "unauthenticated"
		campaign     *kolide.DistributedQueryCampaign
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "NewDistributedQueryCampaign",
			"err", err,
			"user", loggedInUser,
			"sql", queryString,
			"numHosts", campaign.Metrics.TotalHosts,
			"took", time.Since(begin),
		)
	}(time.Now())
	campaign, err = mw.Service.NewDistributedQueryCampaign(ctx, queryString, hosts, labels)
	return campaign, err
}

func (mw loggingMiddleware) NewDistributedQueryCampaignByNames(ctx context.Context, queryString string, hosts []string, labels []string) (*kolide.DistributedQueryCampaign, error) {
	var (
		loggedInUser = "unauthenticated"
		campaign     *kolide.DistributedQueryCampaign
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "NewDistributedQueryCampaignByNames",
			"err", err,
			"user", loggedInUser,
			"numHosts", campaign.Metrics.TotalHosts,
			"took", time.Since(begin),
		)
	}(time.Now())
	campaign, err = mw.Service.NewDistributedQueryCampaignByNames(ctx, queryString, hosts, labels)
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
		_ = mw.logger.Log(
			"method", "StreamCampaignResults",
			"campaignID", campaignID,
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	mw.Service.StreamCampaignResults(ctx, conn, campaignID)
}
