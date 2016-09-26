package datastore

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) FindSessionByID(id uint) (*kolide.Session, error) {
	session := &kolide.Session{
		ID: id,
	}

	err := orm.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, kolide.ErrNoActiveSession
		default:
			return nil, err
		}
	}

	return session, nil

}

func (orm gormDB) FindSessionByKey(key string) (*kolide.Session, error) {
	session := &kolide.Session{
		Key: key,
	}

	err := orm.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, kolide.ErrNoActiveSession
		default:
			return nil, err
		}
	}

	return session, nil
}

func (orm gormDB) FindAllSessionsForUser(id uint) ([]*kolide.Session, error) {
	var sessions []*kolide.Session
	err := orm.DB.Where("user_id = ?", id).Find(&sessions).Error
	return sessions, err
}

func (orm gormDB) NewSession(session *kolide.Session) (*kolide.Session, error) {
	err := orm.DB.Create(session).Error
	if err != nil {
		return nil, err
	}

	err = orm.MarkSessionAccessed(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (orm gormDB) DestroySession(session *kolide.Session) error {
	return orm.DB.Delete(session).Error
}

func (orm gormDB) DestroyAllSessionsForUser(id uint) error {
	return orm.DB.Delete(&kolide.Session{}, "user_id = ?", id).Error
}

func (orm gormDB) MarkSessionAccessed(session *kolide.Session) error {
	session.AccessedAt = time.Now().UTC()
	return orm.DB.Save(session).Error
}
