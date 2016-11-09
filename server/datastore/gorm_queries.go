package datastore

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) NewQuery(query *kolide.Query) (*kolide.Query, error) {
	if query == nil {
		return nil, errors.New(
			"error creating query",
			"nil pointer passed to NewQuery",
		)
	}
	err := orm.DB.Create(query).Error
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (orm gormDB) SaveQuery(query *kolide.Query) error {
	if query == nil {
		return errors.New(
			"error saving query",
			"nil pointer passed to SaveQuery",
		)
	}
	return orm.DB.Save(query).Error
}

func (orm gormDB) DeleteQuery(query *kolide.Query) error {
	if query == nil {
		return errors.New(
			"error deleting query",
			"nil pointer passed to DeleteQuery",
		)
	}
	return orm.DB.Delete(query).Error
}

func (orm gormDB) Query(id uint) (*kolide.Query, error) {
	query := &kolide.Query{
		ID: id,
	}
	err := orm.DB.Where(query).First(query).Error
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (orm gormDB) ListQueries(opt kolide.ListOptions) ([]*kolide.Query, error) {
	var queries []*kolide.Query
	err := orm.applyListOptions(opt).Find(&queries).Error
	return queries, err
}

func (orm gormDB) NewDistributedQueryExecution(exec kolide.DistributedQueryExecution) (kolide.DistributedQueryExecution, error) {
	err := orm.DB.Create(&exec).Error
	return exec, err
}

func (orm gormDB) NewDistributedQueryCampaign(camp kolide.DistributedQueryCampaign) (kolide.DistributedQueryCampaign, error) {
	err := orm.DB.Create(&camp).Error
	return camp, err
}

func (orm gormDB) SaveDistributedQueryCampaign(camp kolide.DistributedQueryCampaign) error {
	return orm.DB.Save(&camp).Error
}

func (orm gormDB) NewDistributedQueryCampaignTarget(target kolide.DistributedQueryCampaignTarget) (kolide.DistributedQueryCampaignTarget, error) {
	err := orm.DB.Create(&target).Error
	return target, err
}
