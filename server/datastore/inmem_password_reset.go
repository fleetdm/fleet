package datastore

import "github.com/kolide/kolide-ose/server/kolide"

func (orm *inmem) NewPasswordResetRequest(req *kolide.PasswordResetRequest) (*kolide.PasswordResetRequest, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	req.ID = orm.nextID(req)
	orm.passwordResets[req.ID] = req
	return req, nil
}

func (orm *inmem) SavePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.passwordResets[req.ID]; !ok {
		return ErrNotFound
	}

	orm.passwordResets[req.ID] = req
	return nil
}

func (orm *inmem) DeletePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.passwordResets[req.ID]; !ok {
		return ErrNotFound
	}

	delete(orm.passwordResets, req.ID)
	return nil
}

func (orm *inmem) DeletePasswordResetRequestsForUser(userID uint) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, pr := range orm.passwordResets {
		if pr.UserID == userID {
			delete(orm.passwordResets, pr.ID)
		}
	}
	return nil
}

func (orm *inmem) FindPassswordResetByID(id uint) (*kolide.PasswordResetRequest, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if req, ok := orm.passwordResets[id]; ok {
		return req, nil
	}

	return nil, ErrNotFound
}

func (orm *inmem) FindPassswordResetsByUserID(userID uint) ([]*kolide.PasswordResetRequest, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	resets := make([]*kolide.PasswordResetRequest, 0)

	for _, pr := range orm.passwordResets {
		if pr.UserID == userID {
			resets = append(resets, pr)
		}
	}

	if len(resets) == 0 {
		return nil, ErrNotFound
	}

	return resets, nil
}

func (orm *inmem) FindPassswordResetByToken(token string) (*kolide.PasswordResetRequest, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, pr := range orm.passwordResets {
		if pr.Token == token {
			return pr, nil
		}
	}

	return nil, ErrNotFound
}

func (orm *inmem) FindPassswordResetByTokenAndUserID(token string, userID uint) (*kolide.PasswordResetRequest, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, pr := range orm.passwordResets {
		if pr.Token == token && pr.UserID == userID {
			return pr, nil
		}
	}

	return nil, ErrNotFound
}
