package datastore

import "github.com/kolide/kolide-ose/server/kolide"

func (orm gormDB) NewInvite(invite *kolide.Invite) (*kolide.Invite, error) {
	err := orm.DB.Create(invite).Error
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (orm gormDB) InviteByEmail(email string) (*kolide.Invite, error) {
	invite := &kolide.Invite{
		Email: email,
	}
	err := orm.DB.Where("email = ?", email).First(invite).Error
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (orm gormDB) Invites(opt kolide.ListOptions) ([]*kolide.Invite, error) {
	var invites []*kolide.Invite
	err := orm.applyListOptions(opt).Find(&invites).Error
	if err != nil {
		return nil, err
	}
	return invites, nil
}

func (orm gormDB) Invite(id uint) (*kolide.Invite, error) {
	invite := &kolide.Invite{ID: id}
	err := orm.DB.Where(invite).First(invite).Error
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (orm gormDB) SaveInvite(invite *kolide.Invite) error {
	return orm.DB.Save(invite).Error
}

func (orm gormDB) DeleteInvite(invite *kolide.Invite) error {
	return orm.DB.Delete(invite).Error
}
