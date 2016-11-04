package datastore

import (
	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	err := orm.DB.First(info).Error
	switch err {
	case gorm.ErrRecordNotFound:
		err = orm.DB.Create(info).Error
		if err != nil {
			return nil, err
		}
		return info, nil
	case nil:
		return info, orm.SaveAppConfig(info)
	default:
		return nil, err
	}
}

func (orm gormDB) AppConfig() (*kolide.AppConfig, error) {
	info := &kolide.AppConfig{}
	err := orm.DB.First(info).Error
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (orm gormDB) SaveAppConfig(info *kolide.AppConfig) error {
	return orm.DB.Save(info).Error
}
