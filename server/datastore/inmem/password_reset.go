package inmem

import (
	"fmt"

	"github.com/kolide/kolide/server/kolide"
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
		return notFound("PasswordResetRequest").WithID(req.ID)
	}

	d.passwordResets[req.ID] = req
	return nil
}

func (d *Datastore) DeletePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.passwordResets[req.ID]; !ok {
		return notFound("PasswordResetRequest").WithID(req.ID)
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

	return nil, notFound("PasswordResetRequest").WithID(id)
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
		return nil, notFound("PasswordResetRequest").
			WithMessage(fmt.Sprintf("for user id %d", userID))
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

	return nil, notFound("PasswordResetRequest")
}

func (d *Datastore) FindPassswordResetByTokenAndUserID(token string, userID uint) (*kolide.PasswordResetRequest, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pr := range d.passwordResets {
		if pr.Token == token && pr.UserID == userID {
			return pr, nil
		}
	}

	return nil, notFound("PasswordResetRequest")
}
