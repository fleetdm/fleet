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

type LeaderSelector interface {
	AmILeader() bool
	ShutDown()
}

type Locker interface {
	Lock(name string, owner string, expiration time.Duration) (bool, error)
	Unlock(name string, owner string) error
}

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
	ls.run()
	return ls
}

func (ls *leaderSelector) run() {
	go ls.worker()
}

func (ls *leaderSelector) tryGetLeadership() {
	ls.m.Lock()
	defer ls.m.Unlock()
	if ls.leader {
		return
	}
	locked, err := ls.locker.Lock(ls.name, ls.owner, ls.expiration)
	if err != nil {
		ls.leader = false
		return
	}
	ls.leader = locked
}

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

func (ls *leaderSelector) AmILeader() bool {
	ls.m.Lock()
	defer ls.m.Unlock()

	return ls.leader
}

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
