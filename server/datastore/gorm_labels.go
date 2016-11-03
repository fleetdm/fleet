package datastore

import (
	"bytes"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) NewLabel(label *kolide.Label) (*kolide.Label, error) {
	if label == nil {
		return nil, errors.New(
			"error creating label",
			"nil pointer passed to NewLabel",
		)
	}
	err := orm.DB.Create(label).Error
	if err != nil {
		return nil, err
	}
	return label, nil
}

func (orm gormDB) DeleteLabel(lid uint) error {
	err := orm.DB.Where("id = ?", lid).Delete(&kolide.Label{}).Error
	if err != nil {
		return err
	}

	return orm.DB.Where("target_id = ? and type = ?", lid, kolide.TargetLabel).Delete(&kolide.PackTarget{}).Error
}

func (orm gormDB) Label(lid uint) (*kolide.Label, error) {
	label := &kolide.Label{
		ID: lid,
	}
	err := orm.DB.Where("id = ?", label.ID).First(&label).Error
	if err != nil {
		return nil, err
	}
	return label, nil
}

func (orm gormDB) ListLabels(opt kolide.ListOptions) ([]*kolide.Label, error) {
	var labels []*kolide.Label
	err := orm.applyListOptions(opt).Find(&labels).Error
	return labels, err
}

func (orm gormDB) LabelQueriesForHost(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
	if host == nil {
		return nil, errors.New(
			"error finding host queries",
			"nil pointer passed to LabelQueriesForHost",
		)
	}
	rows, err := orm.DB.Raw(`
SELECT l.id, l.query
from labels l
WHERE l.platform = ?
AND l.id NOT IN /* subtract the set of executions that are recent enough */
(
  SELECT l.id
  FROM labels l
  JOIN label_query_executions lqe
  ON lqe.label_id = l.id
  WHERE lqe.host_id = ? AND lqe.updated_at > ?
)`, host.Platform, host.ID, cutoff).Rows()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var id, query string
		err = rows.Scan(&id, &query)
		if err != nil {
			return nil, errors.DatabaseError(err)
		}
		results[id] = query
	}

	return results, nil
}

func (orm gormDB) RecordLabelQueryExecutions(host *kolide.Host, results map[string]bool, t time.Time) error {
	if host == nil {
		return errors.New(
			"error recording host label query execution",
			"nil pointer passed to RecordLabelQueryExecutions",
		)
	}

	insert := new(bytes.Buffer)

	insert.WriteString(
		"INSERT INTO label_query_executions (updated_at, matches, label_id, host_id) VALUES",
	)

	// Build up all the values and the query string
	vals := []interface{}{}
	for labelId, res := range results {
		insert.WriteString("(?,?,?,?),")
		vals = append(vals, t, res, labelId, host.ID)
	}

	queryString := insert.String()
	queryString = strings.TrimSuffix(queryString, ",")

	queryString += `
ON DUPLICATE KEY UPDATE
updated_at = VALUES(updated_at),
matches = VALUES(matches)
`

	if err := orm.DB.Exec(queryString, vals...).Error; err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (orm gormDB) ListLabelsForHost(hid uint) ([]kolide.Label, error) {
	results := []kolide.Label{}
	err := orm.DB.Raw(`
SELECT labels.* from labels, label_query_executions lqe
WHERE lqe.host_id = ?
AND lqe.label_id = labels.id
AND lqe.matches
`, hid).Scan(&results).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}

func (orm gormDB) SearchLabels(query string, omit []uint) ([]kolide.Label, error) {
	sql := `
SELECT *
FROM labels
WHERE MATCH(name)
AGAINST(? IN BOOLEAN MODE)
`
	results := []kolide.Label{}

	var db *gorm.DB
	if len(omit) > 0 {
		sql += "AND id NOT IN (?) LIMIT 10;"
		db = orm.DB.Raw(sql, query+"*", omit)
	} else {
		sql += "LIMIT 10;"
		db = orm.DB.Raw(sql, query+"*")
	}

	err := db.Scan(&results).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}
	return results, nil
}

func (orm gormDB) ListHostsInLabel(lid uint) ([]kolide.Host, error) {
	results := []kolide.Host{}
	err := orm.DB.Raw(`
SELECT h.*
FROM label_query_executions lqe
JOIN hosts h
ON lqe.host_id = h.id
WHERE lqe.label_id = ?
AND lqe.matches = 1;
`, lid).Scan(&results).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}

func (orm gormDB) ListUniqueHostsInLabels(labels []uint) ([]kolide.Host, error) {
	if labels == nil || len(labels) == 0 {
		return nil, nil
	}

	results := []kolide.Host{}
	err := orm.DB.Raw(`
SELECT h.*
FROM label_query_executions lqe
JOIN hosts h
ON lqe.host_id = h.id
WHERE lqe.label_id in (?)
AND lqe.matches = 1
GROUP BY h.id;
`, labels).Scan(&results).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}
