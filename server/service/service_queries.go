package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
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

func (svc service) NewDistributedQueryCampaign(ctx context.Context, userID uint, queryString string, hosts []uint, labels []uint) (*kolide.DistributedQueryCampaign, error) {
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
		UserID:  userID,
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
