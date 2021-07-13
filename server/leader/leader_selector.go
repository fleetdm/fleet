package leader

import (
	"github.com/fleetdm/fleet/v4/server"
	"sync"
	"time"
)

type leaderSelector struct {
	m             sync.Mutex
	close         chan interface{}
	leader        bool
	renewInterval time.Duration
	expiration    time.Duration
	locker        Locker
	name          string
	owner         string
}

// LeaderSelector represents an object that, via a Locker object, obtains
// leadership among other instances. This is meant to be used once per
// instance.
type LeaderSelector interface {
	AmILeader() bool
	ShutDown()
}

// Locker represents an object that can obtain an atomic lock on a resource
// in a non blocking manner for an owner, with an expiration time.
type Locker interface {
	// Lock tries to get an atomic lock on an instance named with `name`
	// and an `owner` identified by a random string per instance.
	// Subsequently locking the same resource name for the same owner
	// renews the lock expiration.
	// It returns true, nil if it managed to obtain a lock on the instance.
	// false and potentially an error otherwise.
	// This must not be blocking.
	Lock(name string, owner string, expiration time.Duration) (bool, error)
	// Unlock tries to unlock the lock by that `name` for the specified `owner`.
	Unlock(name string, owner string) error
}

// NewLeaderSelector creates a new leader selector and kickstarts the selection process.
// The selector will constantly try to become leader. The lock on the leadership will expire
// after `expiration` time, and it'll try to renew it every `renewInterval` time.
// Setting expiration lower than renewInterval would allow for more chances of leadership to
// shift between instances. Depending on whether that's a wanted effect or not, you can set
// these parameters accordingly.
func NewLeaderSelector(locker Locker, expiration time.Duration, renewInterval time.Duration) LeaderSelector {
	owner, err := server.GenerateRandomText(64)
	if err != nil {
		panic(err)
	}
	ls := &leaderSelector{
		close:         make(chan interface{}),
		renewInterval: renewInterval,
		expiration:    expiration,
		locker:        locker,
		owner:         owner,
	}
	go ls.worker()
	return ls
}

// tryGetLeadership gets or extends the lock on leadership using Locker
// and sets the internal value to true if leadership is obtained.
func (ls *leaderSelector) tryGetLeadership() {
	ls.m.Lock()
	defer ls.m.Unlock()
	locked, err := ls.locker.Lock(ls.name, ls.owner, ls.expiration)
	if err != nil {
		ls.leader = false
		return
	}
	ls.leader = locked
}

// worker is a goroutine that tries every renewInterval to get the
// leadership lock and unlocks it if exiting.
func (ls *leaderSelector) worker() {
	ticker := time.NewTicker(ls.renewInterval)
	for {
		select {
		case <-ticker.C:
			ls.tryGetLeadership()
		case <-ls.close:
			ls.locker.Unlock(ls.name, ls.owner)
			return
		}
	}
}

// AmILeader returns true if this instance currently has the leadership
// lock
func (ls *leaderSelector) AmILeader() bool {
	ls.m.Lock()
	defer ls.m.Unlock()

	return ls.leader
}

// ShutDown ends the worker loop
func (ls *leaderSelector) ShutDown() {
	select {
	case _, ok := <-ls.close:
		if !ok {
			return
		}
	default:
		ls.m.Lock()
		defer ls.m.Unlock()
		ls.leader = false
		close(ls.close)
	}
}
