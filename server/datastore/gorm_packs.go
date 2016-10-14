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
	var queries []*kolide.Query
	if pack == nil {
		return nil, errors.New(
			"error getting queries in pack",
			"nil pointer passed to GetQueriesInPack",
		)
	}

	rows, err := orm.DB.Raw(`
SELECT
  q.id,
  q.created_at,
  q.updated_at,
  q.name,
  q.query,
  q.interval,
  q.snapshot,
  q.differential,
  q.platform,
  q.version
FROM
  queries q
JOIN
  pack_queries pq
ON
  pq.query_id = q.id
AND
  pq.pack_id = ?;
`, pack.ID).Rows()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}
	defer rows.Close()

	for rows.Next() {
		query := new(kolide.Query)
		err = rows.Scan(
			&query.ID,
			&query.CreatedAt,
			&query.UpdatedAt,
			&query.Name,
			&query.Query,
			&query.Interval,
			&query.Snapshot,
			&query.Differential,
			&query.Platform,
			&query.Version,
		)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}

	return queries, nil
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
		Type:     kolide.TargetLabel,
		PackID:   pid,
		TargetID: lid,
	}

	return orm.DB.Create(pt).Error
}

func (orm gormDB) ListPacksForHost(hid uint) ([]*kolide.Pack, error) {
	packs := []*kolide.Pack{}

	// we will need to give some subset of packs to this host based on the
	// labels which this host is known to belong to
	allPacks, err := orm.ListPacks(kolide.ListOptions{})
	if err != nil {
		return nil, err
	}

	// pull the labels that this host belongs to
	labels, err := orm.ListLabelsForHost(hid)
	if err != nil {
		return nil, err
	}

	// in order to use o(1) array indexing in an o(n) loop vs a o(n^2) double
	// for loop iteration, we must create the array which may be indexed below
	labelIDs := map[uint]bool{}
	for _, label := range labels {
		labelIDs[label.ID] = true
	}

	for _, pack := range allPacks {
		// for each pack, we must know what labels have been assigned to that
		// pack
		labelsForPack, err := orm.ListLabelsForPack(pack)
		if err != nil {
			return nil, err
		}

		// o(n) iteration to determine whether or not a pack is enabled
		// in this case, n is len(labelsForPack)
		for _, label := range labelsForPack {
			if labelIDs[label.ID] {
				packs = append(packs, pack)
				break
			}
		}
	}

	return packs, nil
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
	l.id,
	l.created_at,
	l.updated_at,
	l.name,
	l.query_id
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

	pt := &kolide.PackTarget{
		Type:     kolide.TargetLabel,
		PackID:   pack.ID,
		TargetID: label.ID,
	}

	return orm.DB.Delete(pt).Error
}
