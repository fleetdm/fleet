package datastore

import (
	"reflect"
	"sync"

	"github.com/kolide/kolide-ose/server/kolide"
)

type inmem struct {
	kolide.Datastore
	Driver  string
	mtx     sync.RWMutex
	nextIDs map[interface{}]uint

	users                map[uint]*kolide.User
	sessions             map[uint]*kolide.Session
	passwordResets       map[uint]*kolide.PasswordResetRequest
	invites              map[uint]*kolide.Invite
	labels               map[uint]*kolide.Label
	labelQueryExecutions map[uint]*kolide.LabelQueryExecution
	queries              map[uint]*kolide.Query
	packs                map[uint]*kolide.Pack
	hosts                map[uint]*kolide.Host

	orginfo *kolide.OrgInfo
}

func (orm *inmem) Name() string {
	return "inmem"
}

func (orm *inmem) Migrate() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.nextIDs = make(map[interface{}]uint)
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	orm.passwordResets = make(map[uint]*kolide.PasswordResetRequest)
	orm.invites = make(map[uint]*kolide.Invite)
	orm.labels = make(map[uint]*kolide.Label)
	orm.labelQueryExecutions = make(map[uint]*kolide.LabelQueryExecution)
	orm.queries = make(map[uint]*kolide.Query)
	orm.packs = make(map[uint]*kolide.Pack)
	orm.hosts = make(map[uint]*kolide.Host)
	return nil
}

func (orm *inmem) Drop() error {
	return orm.Migrate()
}

// getLimitOffsetSliceBounds returns the bounds that should be used for
// re-slicing the results to comply with the requested ListOptions. Lack of
// generics forces us to do this rather than reslicing in this method.
func (orm *inmem) getLimitOffsetSliceBounds(opt kolide.ListOptions, length int) (low uint, high uint) {
	if opt.PerPage == 0 {
		// PerPage value of 0 indicates unlimited
		return 0, uint(length)
	}

	offset := opt.Page * opt.PerPage
	max := offset + opt.PerPage
	if offset > uint(length) {
		offset = uint(length)
	}
	if max > uint(length) {
		max = uint(length)
	}
	return offset, max
}

// nextID returns the next ID value that should be used for a struct of the
// given type
func (orm *inmem) nextID(val interface{}) uint {
	valType := reflect.TypeOf(val)
	orm.nextIDs[valType]++
	return orm.nextIDs[valType]
}
