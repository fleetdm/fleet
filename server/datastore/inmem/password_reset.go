package inmem

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewPasswordResetRequest(req *kolide.PasswordResetRequest) (*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	req.ID = d.nextID(req)
	d.passwordResets[req.ID] = req
	return req, nil
}

func (d *Datastore) SavePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.passwordResets[req.ID]; !ok {
		return errors.ErrNotFound
	}

	d.passwordResets[req.ID] = req
	return nil
}

func (d *Datastore) DeletePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.passwordResets[req.ID]; !ok {
		return errors.ErrNotFound
	}

	delete(d.passwordResets, req.ID)
	return nil
}

func (d *Datastore) DeletePasswordResetRequestsForUser(userID uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pr := range d.passwordResets {
		if pr.UserID == userID {
			delete(d.passwordResets, pr.ID)
		}
	}
	return nil
}

func (d *Datastore) FindPassswordResetByID(id uint) (*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if req, ok := d.passwordResets[id]; ok {
		return req, nil
	}

	return nil, errors.ErrNotFound
}

func (d *Datastore) FindPassswordResetsByUserID(userID uint) ([]*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	resets := make([]*kolide.PasswordResetRequest, 0)

	for _, pr := range d.passwordResets {
		if pr.UserID == userID {
			resets = append(resets, pr)
		}
	}

	if len(resets) == 0 {
		return nil, errors.ErrNotFound
	}

	return resets, nil
}

func (d *Datastore) FindPassswordResetByToken(token string) (*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pr := range d.passwordResets {
		if pr.Token == token {
			return pr, nil
		}
	}

	return nil, errors.ErrNotFound
}

func (d *Datastore) FindPassswordResetByTokenAndUserID(token string, userID uint) (*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pr := range d.passwordResets {
		if pr.Token == token && pr.UserID == userID {
			return pr, nil
		}
	}

	return nil, errors.ErrNotFound
}
