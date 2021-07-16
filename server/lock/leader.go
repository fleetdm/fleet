package lock

import (
	"time"

	"github.com/fleetdm/fleet/v4/server"
)

var owner = ""

func init() {
	var err error
	owner, err = server.GenerateRandomText(64)
	if err != nil {
		panic(err)
	}
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
	// Unlock tries to unlock the lock by that `name` for the specified
	// `owner`. Unlocking when not holding the lock shouldn't error
	Unlock(name string, owner string) error
}

const (
	LockKeyLeader = "leader"
)

func Leader(locker Locker, expiration time.Duration) (bool, error) {
	return locker.Lock(LockKeyLeader, owner, expiration)
}
