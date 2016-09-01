package datastore

import (
	"sync"

	"github.com/kolide/kolide-ose/kolide"
)

type mockDB struct {
	kolide.Datastore
	Driver          string
	sessionKeySize  int
	sessionLifespan float64
	mtx             sync.RWMutex
	users           map[uint]*kolide.User
	sessions        map[uint]*kolide.Session
}

func (orm *mockDB) Name() string {
	return "mock"
}

func (orm *mockDB) Migrate() error {
	return nil
}

func (orm *mockDB) Drop() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	return nil
}
