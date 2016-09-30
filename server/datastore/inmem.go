package datastore

import (
	"sync"

	"github.com/kolide/kolide-ose/server/kolide"
)

type inmem struct {
	kolide.Datastore
	Driver               string
	mtx                  sync.RWMutex
	users                map[uint]*kolide.User
	sessions             map[uint]*kolide.Session
	passwordResets       map[uint]*kolide.PasswordResetRequest
	invites              map[uint]*kolide.Invite
	labels               map[uint]*kolide.Label
	labelQueryExecutions map[uint]*kolide.LabelQueryExecution
	queries              map[uint]*kolide.Query
	hosts                map[uint]*kolide.Host
	orginfo              *kolide.OrgInfo
}

func (orm *inmem) Name() string {
	return "inmem"
}

func (orm *inmem) Migrate() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	orm.passwordResets = make(map[uint]*kolide.PasswordResetRequest)
	orm.invites = make(map[uint]*kolide.Invite)
	orm.labels = make(map[uint]*kolide.Label)
	orm.labelQueryExecutions = make(map[uint]*kolide.LabelQueryExecution)
	orm.queries = make(map[uint]*kolide.Query)
	orm.hosts = make(map[uint]*kolide.Host)
	return nil
}

func (orm *inmem) Drop() error {
	return orm.Migrate()
}
