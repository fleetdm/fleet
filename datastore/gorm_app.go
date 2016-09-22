package datastore

import (
	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/kolide"
)

func (orm gormDB) NewOrgInfo(info *kolide.OrgInfo) (*kolide.OrgInfo, error) {
	err := orm.DB.First(info).Error
	switch err {
	case gorm.ErrRecordNotFound:
		err = orm.DB.Create(info).Error
		if err != nil {
			return nil, err
		}
		return info, nil
	case nil:
		return info, orm.SaveOrgInfo(info)
	default:
		return nil, err
	}
}

func (orm gormDB) OrgInfo() (*kolide.OrgInfo, error) {
	info := &kolide.OrgInfo{}
	err := orm.DB.First(info).Error
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (orm gormDB) SaveOrgInfo(info *kolide.OrgInfo) error {
	return orm.DB.Save(info).Error
}
