package datastore

import (
	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) NewPack(pack *kolide.Pack) error {
	if pack == nil {
		return errors.New(
			"error creating pack",
			"nil pointer passed to NewPack",
		)
	}
	return orm.DB.Create(pack).Error
}

func (orm gormDB) SavePack(pack *kolide.Pack) error {
	if pack == nil {
		return errors.New(
			"error saving pack",
			"nil pointer passed to SavePack",
		)
	}
	return orm.DB.Save(pack).Error
}

func (orm gormDB) DeletePack(pid uint) error {
	err := orm.DB.Where("id = ?", pid).Delete(&kolide.Pack{}).Error
	if err != nil {
		return err
	}

	err = orm.DB.Where("pack_id = ?", pid).Delete(&kolide.PackQuery{}).Error
	if err != nil {
		return err
	}
	return orm.DB.Where("pack_id = ?", pid).Delete(&kolide.PackTarget{}).Error
}

func (orm gormDB) Pack(pid uint) (*kolide.Pack, error) {
	pack := &kolide.Pack{
		ID: pid,
	}
	err := orm.DB.Where(pack).First(pack).Error
	if err != nil {
		return nil, err
	}
	return pack, nil
}

func (orm gormDB) ListPacks(opt kolide.ListOptions) ([]*kolide.Pack, error) {
	var packs []*kolide.Pack
	err := orm.applyListOptions(opt).Find(&packs).Error
	return packs, err
}

func (orm gormDB) AddQueryToPack(qid uint, pid uint) error {
	pq := &kolide.PackQuery{
		QueryID: qid,
		PackID:  pid,
	}
	return orm.DB.Create(pq).Error
}

func (orm gormDB) ListQueriesInPack(pack *kolide.Pack) ([]*kolide.Query, error) {
	if pack == nil {
		return nil, errors.New(
			"error getting queries in pack",
			"nil pointer passed to GetQueriesInPack",
		)
	}

	results := []*kolide.Query{}
	err := orm.DB.Raw(`
SELECT
  q.*
FROM
  queries q
JOIN
  pack_queries pq
ON
  pq.query_id = q.id
AND
  pq.pack_id = ?;
`, pack.ID).Scan(&results).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}

func (orm gormDB) RemoveQueryFromPack(query *kolide.Query, pack *kolide.Pack) error {
	if query == nil || pack == nil {
		return errors.New(
			"error removing query from pack",
			"nil pointer passed to RemoveQueryFromPack",
		)
	}
	pq := &kolide.PackQuery{
		QueryID: query.ID,
		PackID:  pack.ID,
	}
	return orm.DB.Where(pq).Delete(pq).Error
}

func (orm gormDB) AddLabelToPack(lid uint, pid uint) error {
	pt := &kolide.PackTarget{
		PackID: pid,
		Target: kolide.Target{
			Type:     kolide.TargetLabel,
			TargetID: lid,
		},
	}

	return orm.DB.Create(pt).Error
}

func (orm gormDB) ListLabelsForPack(pack *kolide.Pack) ([]*kolide.Label, error) {
	if pack == nil {
		return nil, errors.New(
			"error getting labels for pack",
			"nil pointer passed to GetLabelsForPack",
		)
	}

	results := []*kolide.Label{}
	err := orm.DB.Raw(`
SELECT
	l.*
FROM
	labels l
JOIN
	pack_targets pt
ON
	pt.target_id = l.id
WHERE
	pt.type = ?
		AND
	pt.pack_id = ?;

`,
		kolide.TargetLabel, pack.ID).Scan(&results).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}

func (orm gormDB) RemoveLabelFromPack(label *kolide.Label, pack *kolide.Pack) error {
	if label == nil || pack == nil {
		return errors.New(
			"error removing label from pack",
			"nil pointer passed to RemoveLabelFromPack",
		)
	}

	return orm.DB.Where("pack_id = ? AND type = ? AND target_id = ?", pack.ID, kolide.TargetLabel, label.ID).Delete(&kolide.PackTarget{}).Error
}
