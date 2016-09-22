package datastore

import (
	"sync"

	"github.com/kolide/kolide-ose/kolide"
)

type inmem struct {
	kolide.Datastore
	Driver         string
	mtx            sync.RWMutex
	users          map[uint]*kolide.User
	sessions       map[uint]*kolide.Session
	passwordResets map[uint]*kolide.PasswordResetRequest
	orginfo        *kolide.OrgInfo
}

func (orm *inmem) Name() string {
	return "inmem"
}

func (orm *inmem) Migrate() error {
	return nil
}

func (orm *inmem) Drop() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	orm.passwordResets = make(map[uint]*kolide.PasswordResetRequest)
	return nil
}
