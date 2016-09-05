package datastore

import "github.com/kolide/kolide-ose/kolide"

// NewUser creates a new user in the gorm backend
func (orm gormDB) NewUser(user *kolide.User) (*kolide.User, error) {
	err := orm.DB.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// User returns a specific user in the gorm backend
func (orm gormDB) User(username string) (*kolide.User, error) {
	user := &kolide.User{
		Username: username,
	}
	err := orm.DB.Where("username = ?", username).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UserByID returns a datastore user given a user ID
func (orm gormDB) UserByID(id uint) (*kolide.User, error) {
	user := &kolide.User{ID: id}
	err := orm.DB.Where(user).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (orm gormDB) SaveUser(user *kolide.User) error {
	return orm.DB.Save(user).Error
}
