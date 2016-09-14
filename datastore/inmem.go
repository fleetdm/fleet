package datastore

import (
	"sync"

	"github.com/kolide/kolide-ose/kolide"
)

type inmem struct {
	kolide.Datastore
	Driver   string
	mtx      sync.RWMutex
	users    map[uint]*kolide.User
	sessions map[uint]*kolide.Session
}

func (orm *inmem) Name() string {
	return "mock"
}

func (orm *inmem) Migrate() error {
	return nil
}

func (orm *inmem) Drop() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	return nil
}
